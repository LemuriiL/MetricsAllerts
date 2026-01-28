package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/storage"
	"github.com/gorilla/mux"
)

type Handler struct {
	storage storage.Storage
	db      *sql.DB
}

func NewHandler(s storage.Storage) *Handler {
	return &Handler{storage: s}
}

func NewHandlerWithDB(s storage.Storage, db *sql.DB) *Handler {
	return &Handler{storage: s, db: db}
}

func (h *Handler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	metricType := vars["type"]
	metricName := vars["name"]
	metricValueStr := vars["value"]

	log.Printf("update metric: type=%s name=%s value=%s", metricType, metricName, metricValueStr)

	if metricName == "" {
		http.NotFound(w, r)
		return
	}

	switch metricType {
	case "gauge":
		val, err := strconv.ParseFloat(metricValueStr, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		h.storage.SetGauge(metricName, val)

	case "counter":
		val, err := strconv.ParseInt(metricValueStr, 10, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		h.storage.SetCounter(metricName, val)

	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetricValue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	metricType := vars["type"]
	metricName := vars["name"]

	switch metricType {
	case "gauge":
		if val, ok := h.storage.GetGauge(metricName); ok {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "%g", val)
			return
		}
	case "counter":
		if val, ok := h.storage.GetCounter(metricName); ok {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "%d", val)
			return
		}
	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	http.NotFound(w, r)
}

func (h *Handler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, "<h1>All Metrics</h1>")
	fmt.Fprintln(w, "<h2>Gauges</h2><ul>")
	for name, value := range gauges {
		fmt.Fprintf(w, "<li>%s: %g</li>", name, value)
	}
	fmt.Fprintln(w, "</ul><h2>Counters</h2><ul>")
	for name, value := range counters {
		fmt.Fprintf(w, "<li>%s: %d</li>", name, value)
	}
	fmt.Fprintln(w, "</ul>")
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		log.Printf("db ping failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
