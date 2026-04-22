package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/audit"
	models "github.com/LemuriiL/MetricsAllerts/internal/model"
	"github.com/LemuriiL/MetricsAllerts/internal/service"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

type metricService interface {
	SetGauge(ctx context.Context, name string, value float64) error
	GetGauge(ctx context.Context, name string) (float64, bool, error)
	SetCounter(ctx context.Context, name string, value int64) error
	GetCounter(ctx context.Context, name string) (int64, bool, error)
	GetAllGauges(ctx context.Context) (map[string]float64, error)
	GetAllCounters(ctx context.Context) (map[string]int64, error)
	UpdateMetric(ctx context.Context, metric models.Metrics) error
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
	GetMetric(ctx context.Context, metric models.Metrics) (models.Metrics, error)
}

type Handler struct {
	service metricService
	db      *sql.DB
	auditor *audit.Broadcaster
}

func NewHandler(repository storage.Storage) *Handler {
	return &Handler{service: service.NewMetricsService(repository)}
}

func NewHandlerWithDB(repository storage.Storage, db *sql.DB) *Handler {
	return &Handler{
		service: service.NewMetricsService(repository),
		db:      db,
	}
}

func (h *Handler) SetAuditor(a *audit.Broadcaster) {
	h.auditor = a
}

func (h *Handler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricType, metricName, metricValueStr, ok := parseUpdatePath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	slog.Info(
		"update metric",
		"type", metricType,
		"name", metricName,
		"value", metricValueStr,
	)

	if metricName == "" {
		http.NotFound(w, r)
		return
	}

	switch metricType {
	case models.Gauge:
		value, err := strconv.ParseFloat(metricValueStr, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if err := h.service.SetGauge(r.Context(), metricName, value); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	case models.Counter:
		value, err := strconv.ParseInt(metricValueStr, 10, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if err := h.service.SetCounter(r.Context(), metricName, value); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	h.emitAudit(r, []string{metricName})
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetricValue(w http.ResponseWriter, r *http.Request) {
	metricType, metricName, ok := parseValuePath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch metricType {
	case models.Gauge:
		value, ok, err := h.service.GetGauge(r.Context(), metricName)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if ok {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "%g", value)
			return
		}
	case models.Counter:
		value, ok, err := h.service.GetCounter(r.Context(), metricName)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if ok {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "%d", value)
			return
		}
	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	http.NotFound(w, r)
}

func (h *Handler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {
	gauges, err := h.service.GetAllGauges(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	counters, err := h.service.GetAllCounters(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprintln(w, "<html><body>")
	fmt.Fprintln(w, "<h1>All Metrics</h1>")
	fmt.Fprintln(w, "<h2>Gauges</h2>")
	fmt.Fprintln(w, "<ul>")

	for name, value := range gauges {
		fmt.Fprintf(w, "<li>%s: %g</li>", name, value)
	}

	fmt.Fprintln(w, "</ul>")
	fmt.Fprintln(w, "<h2>Counters</h2>")
	fmt.Fprintln(w, "<ul>")

	for name, value := range counters {
		fmt.Fprintf(w, "<li>%s: %d</li>", name, value)
	}

	fmt.Fprintln(w, "</ul>")
	fmt.Fprintln(w, "</body></html>")
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		slog.Error("db ping failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func parseUpdatePath(path string) (string, string, string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "update" {
		return "", "", "", false
	}

	return parts[1], parts[2], parts[3], true
}

func parseValuePath(path string) (string, string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "value" {
		return "", "", false
	}

	return parts[1], parts[2], true
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
