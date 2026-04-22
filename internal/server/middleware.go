package server

import (
	"log/slog"
	"net/http"
	"time"
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
