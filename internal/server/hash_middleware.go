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
			h := strings.TrimSpace(r.Header.Get("HashSHA256"))
			if h == "" {
				h = strings.TrimSpace(r.Header.Get("Hash"))
			}

			if strings.EqualFold(h, "none") {
				next.ServeHTTP(w, r)
				return
			}

			requestIsSigned := h != ""

			if (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) && requestIsSigned {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
				_ = r.Body.Close()

				sum := sha256.Sum256(append(body, []byte(key)...))
				want := hex.EncodeToString(sum[:])
				if h != want {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}

				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			if !requestIsSigned {
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

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		})
	}
}
