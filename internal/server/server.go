package server

import (
	"database/sql"
	"net/http"

	"github.com/LemuriiL/MetricsAllerts/internal/storage"
	"github.com/gorilla/mux"
)

type Server struct {
	handler *Handler
	key     string
}

func New(storage storage.Storage, db *sql.DB, key string) *Server {
	return &Server{
		handler: NewHandlerWithDB(storage, db),
		key:     key,
	}
}

func (s *Server) Run(addr string) error {
	r := mux.NewRouter()
	r.SkipClean(true)

	r.Use(loggingMiddleware)
	r.Use(gzipMiddleware)
	if s.key != "" {
		r.Use(hashMiddleware(s.key))
	}

	r.HandleFunc("/ping", s.handler.Ping).Methods(http.MethodGet)

	r.HandleFunc("/update/{type}/{name}/{value}", s.handler.UpdateMetric).Methods(http.MethodPost)
	r.HandleFunc("/value/{type}/{name}", s.handler.GetMetricValue).Methods(http.MethodGet)
	r.HandleFunc("/", s.handler.GetAllMetrics).Methods(http.MethodGet)

	r.HandleFunc("/update", s.handler.UpdateMetricJSON).Methods(http.MethodPost)
	r.HandleFunc("/update/", s.handler.UpdateMetricJSON).Methods(http.MethodPost)

	r.HandleFunc("/updates", s.handler.UpdateMetricsJSON).Methods(http.MethodPost)
	r.HandleFunc("/updates/", s.handler.UpdateMetricsJSON).Methods(http.MethodPost)

	r.HandleFunc("/value", s.handler.GetMetricJSON).Methods(http.MethodPost)
	r.HandleFunc("/value/", s.handler.GetMetricJSON).Methods(http.MethodPost)

	return http.ListenAndServe(addr, r)
}
