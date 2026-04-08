package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

type captureWriter struct {
	w      http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (c *captureWriter) Header() http.Header {
	return c.w.Header()
}

func (c *captureWriter) WriteHeader(code int) {
	c.status = code
}

func (c *captureWriter) Write(p []byte) (int, error) {
	return c.buf.Write(p)
}

func signHashMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isRequestSigned(r) {
				next.ServeHTTP(w, r)
				return
			}

			cw := &captureWriter{w: w, status: http.StatusOK}
			next.ServeHTTP(cw, r)

			bodyOut := cw.buf.Bytes()
			sum := sha256.Sum256(append(bodyOut, []byte(key)...))
			w.Header().Set("HashSHA256", hex.EncodeToString(sum[:]))

			if cw.status != 0 {
				w.WriteHeader(cw.status)
			}
			_, _ = w.Write(bodyOut)
		})
	}
}
