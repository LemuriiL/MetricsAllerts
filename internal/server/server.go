package server

import (
	"crypto/rsa"
	"database/sql"
	"net/http"
	"sync"

	"github.com/LemuriiL/MetricsAllerts/internal/audit"
	"github.com/LemuriiL/MetricsAllerts/internal/cryptoutil"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

type Server struct {
	handler       *Handler
	key           string
	cryptoKeyPath string

	privateKey     *rsa.PrivateKey
	privateKeyErr  error
	privateKeyOnce sync.Once
}

func New(storage storage.Storage, db *sql.DB, key string, cryptoKeyPath string) *Server {
	return &Server{
		handler:       NewHandlerWithDB(storage, db),
		key:           key,
		cryptoKeyPath: cryptoKeyPath,
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
	mux.HandleFunc("POST /update/", s.handler.UpdateMetricJSON)
	mux.HandleFunc("POST /updates", s.handler.UpdateMetricsJSON)
	mux.HandleFunc("POST /updates/", s.handler.UpdateMetricsJSON)
	mux.HandleFunc("POST /value", s.handler.GetMetricJSON)
	mux.HandleFunc("POST /value/", s.handler.GetMetricJSON)

	var h http.Handler = mux

	if s.cryptoKeyPath != "" {
		h = decryptMiddleware(s.getPrivateKey)(h)
	}

	if s.key != "" {
		h = verifyHashMiddleware(s.key)(h)
		h = signHashMiddleware(s.key)(h)
	}

	h = gzipMiddleware(h)
	h = loggingMiddleware(h)

	return http.ListenAndServe(addr, h)
}

func (s *Server) getPrivateKey() (*rsa.PrivateKey, error) {
	s.privateKeyOnce.Do(func() {
		s.privateKey, s.privateKeyErr = cryptoutil.LoadPrivateKey(s.cryptoKeyPath)
	})

	return s.privateKey, s.privateKeyErr
}
