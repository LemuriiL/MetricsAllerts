package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
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

func (c *captureWriter) Flush() {
	if f, ok := c.w.(http.Flusher); ok {
		f.Flush()
	}
}

func hashMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			skipByHeader := func() bool {
				v := strings.TrimSpace(r.Header.Get("HashSHA256"))
				if v == "" {
					v = strings.TrimSpace(r.Header.Get("Hash"))
				}
				return strings.EqualFold(v, "none")
			}()

			if skipByHeader {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
				_ = r.Body.Close()

				if len(body) > 0 {
					got := strings.TrimSpace(r.Header.Get("HashSHA256"))
					if got == "" {
						got = strings.TrimSpace(r.Header.Get("Hash"))
					}
					if got == "" {
						http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
						return
					}

					sum := sha256.Sum256(append(body, []byte(key)...))
					want := hex.EncodeToString(sum[:])
					if got != want {
						http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
						return
					}
				}

				r.Body = io.NopCloser(bytes.NewReader(body))
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

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		})
	}
}
