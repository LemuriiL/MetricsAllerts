package server

import (
	"bytes"
	"crypto/rsa"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/cryptoutil"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	n, err := w.ResponseWriter.Write(b)
	w.size += n

	return n, err
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lw := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(lw, r)

		status := lw.status
		if status == 0 {
			status = http.StatusOK
		}

		slog.Info(
			"request handled",
			"uri", r.RequestURI,
			"method", r.Method,
			"duration", time.Since(start).String(),
			"status", status,
			"size", lw.size,
		)
	})
}

func decryptMiddleware(privateKeyProvider func() (*rsa.PrivateKey, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			encryptedKey := r.Header.Get(cryptoutil.HeaderEncryptedKey)
			nonce := r.Header.Get(cryptoutil.HeaderNonce)

			if encryptedKey == "" && nonce == "" {
				next.ServeHTTP(w, r)
				return
			}

			if encryptedKey == "" || nonce == "" {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			privateKey, err := privateKeyProvider()
			if err != nil {
				slog.Error("failed to load private key", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			plaintext, err := cryptoutil.Decrypt(privateKey, body, encryptedKey, nonce)
			if err != nil {
				slog.Error("failed to decrypt request body", "error", err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(plaintext))
			r.ContentLength = int64(len(plaintext))

			next.ServeHTTP(w, r)
		})
	}
}

func trustedSubnetMiddleware(cidr string) func(http.Handler) http.Handler {
	if strings.TrimSpace(cidr) == "" {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
			if realIP == "" {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			ip := net.ParseIP(realIP)
			if ip == nil || !network.Contains(ip) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
