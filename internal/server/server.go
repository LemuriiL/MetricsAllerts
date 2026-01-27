package server

import (
	"net/http"

	"github.com/LemuriiL/MetricsAllerts/internal/storage"
	"github.com/gorilla/mux"
)

type Server struct {
	handler *Handler
}

func New(storage storage.Storage) *Server {
	return &Server{
		handler: NewHandler(storage),
	}
}

func (s *Server) Run(addr string) error {
	r := mux.NewRouter()
	r.SkipClean(true)

	r.Use(loggingMiddleware)

	r.HandleFunc("/update/{type}/{name}/{value}", s.handler.UpdateMetric).Methods("POST")
	r.HandleFunc("/value/{type}/{name}", s.handler.GetMetricValue).Methods("GET")
	r.HandleFunc("/", s.handler.GetAllMetrics).Methods("GET")
	r.HandleFunc("/update", s.handler.UpdateMetricJSON).Methods("POST")
	r.HandleFunc("/update/", s.handler.UpdateMetricJSON).Methods("POST")
	r.HandleFunc("/value", s.handler.GetMetricJSON).Methods("POST")
	r.HandleFunc("/value/", s.handler.GetMetricJSON).Methods("POST")

	return http.ListenAndServe(addr, r)
}
