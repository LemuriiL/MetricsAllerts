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

	r.HandleFunc("/update/{type}/{name}/{value}", s.handler.UpdateMetric).Methods("POST")
	r.HandleFunc("/value/{type}/{name}", s.handler.GetMetricValue).Methods("GET")
	r.HandleFunc("/", s.handler.GetAllMetrics).Methods("GET")

	if len(addr) >= 9 && addr[:9] == "localhost" {
		addr = "127.0.0.1" + addr[9:]
	}

	return http.ListenAndServe(addr, r)
}
