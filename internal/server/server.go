package server

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/audit"
	"github.com/LemuriiL/MetricsAllerts/internal/cryptoutil"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

const (
	serverReadTimeout       = 5 * time.Second
	serverReadHeaderTimeout = 5 * time.Second
	serverWriteTimeout      = 10 * time.Second
	serverIdleTimeout       = 30 * time.Second
)

type Server struct {
	handler       *Handler
	key           string
	cryptoKeyPath string

	privateKey     *rsa.PrivateKey
	privateKeyErr  error
	privateKeyOnce sync.Once

	httpServer *http.Server
}

func New(storage storage.Storage, db *sql.DB, key string, cryptoKeyPath string) *Server {
	return &Server{
		handler:       NewHandlerWithDB(storage, db),
		key:           key,
		cryptoKeyPath: cryptoKeyPath,
	}
}

func NewWithoutSecurity(storage storage.Storage, db *sql.DB) *Server {
	return &Server{
		handler: NewHandlerWithDB(storage, db),
	}
}

func (s *Server) SetAuditor(a *audit.Broadcaster) {
	s.handler.SetAuditor(a)
}

func (s *Server) Run(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", s.handler.Ping)
	mux.HandleFunc("POST /update/{type}/{name}/{value}", s.handler.UpdateMetric)
	mux.HandleFunc("GET /value/{type}/{name}", s.handler.GetMetricValue)
	mux.HandleFunc("GET /", s.handler.GetAllMetrics)
	mux.HandleFunc("POST /update", s.handler.UpdateMetricJSON)
	mux.HandleFunc("POST /updates", s.handler.UpdateMetricsJSON)
	mux.HandleFunc("POST /value", s.handler.GetMetricJSON)

	var h http.Handler = mux

	if s.key != "" {
		h = verifyHashMiddleware(s.key)(h)
		h = signHashMiddleware(s.key)(h)
	}

	h = gzipMiddleware(h)

	if s.cryptoKeyPath != "" {
		h = decryptMiddleware(s.getPrivateKey)(h)
	}

	h = loggingMiddleware(h)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadTimeout:       serverReadTimeout,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}

	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	return s.httpServer.Shutdown(ctx)
}

func (s *Server) getPrivateKey() (*rsa.PrivateKey, error) {
	s.privateKeyOnce.Do(func() {
		s.privateKey, s.privateKeyErr = cryptoutil.LoadPrivateKey(s.cryptoKeyPath)
	})

	return s.privateKey, s.privateKeyErr
}
