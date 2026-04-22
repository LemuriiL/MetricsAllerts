package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/audit"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
	"github.com/gorilla/mux"
)

// Handler отвечает за обработку http запросов с метриками
type Handler struct {
	storage storage.Storage
	db      *sql.DB
	auditor *audit.Broadcaster
}

// NewHandler создает handler без базы
func NewHandler(s storage.Storage) *Handler {
	return &Handler{storage: s}
}

// NewHandlerWithDB создает handler с подключенной базой
func NewHandlerWithDB(s storage.Storage, db *sql.DB) *Handler {
	return &Handler{storage: s, db: db}
}

// SetAuditor подключает аудит событий для handler
func (h *Handler) SetAuditor(a *audit.Broadcaster) {
	h.auditor = a
}

// UpdateMetric обновляет метрику через url параметры
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

	h.emitAudit(r, []string{metricName})
	w.WriteHeader(http.StatusOK)
}

// GetMetricValue возвращает значение метрики
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

// GetAllMetrics возвращает все метрики в html
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

// Ping проверяет доступность базы данных
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

func (h *Handler) emitAudit(r *http.Request, metrics []string) {
	if h.auditor == nil || len(metrics) == 0 {
		return
	}

	h.auditor.Publish(r.Context(), audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   append([]string(nil), metrics...),
		IPAddress: requestIP(r),
	})
}

func requestIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}

	if ip := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); ip != "" {
		parts := strings.Split(ip, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
