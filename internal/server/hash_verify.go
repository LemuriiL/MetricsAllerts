package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

const verifiedHashHeader = "X-Internal-Hash-Verified"

func getHashHeader(r *http.Request) string {
	v := strings.TrimSpace(r.Header.Get("HashSHA256"))
	if v == "" {
		v = strings.TrimSpace(r.Header.Get("Hash"))
	}
	return v
}

func verifyHashMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := getHashHeader(r)
			if h == "" || strings.EqualFold(h, "none") {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

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
			r.Header.Set(verifiedHashHeader, "1")

			next.ServeHTTP(w, r)
		})
	}
}

func isRequestSigned(r *http.Request) bool {
	return r.Header.Get(verifiedHashHeader) == "1"
}
