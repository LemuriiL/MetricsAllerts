package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
	"github.com/LemuriiL/MetricsAllerts/internal/service"
)

type errResp struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errResp{Error: msg})
}

func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metric models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	if err := h.service.UpdateMetric(r.Context(), metric); err != nil {
		if errors.Is(err, service.ErrBadMetric) {
			writeJSONError(w, http.StatusBadRequest, "bad request")
			return
		}

		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.emitAudit(r, []string{metric.ID})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metric)
}

func (h *Handler) UpdateMetricsJSON(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	if len(metrics) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.service.UpdateBatch(r.Context(), metrics); err != nil {
		if errors.Is(err, service.ErrBadMetric) {
			writeJSONError(w, http.StatusBadRequest, "bad request")
			return
		}

		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	names := make([]string, 0, len(metrics))
	for i := range metrics {
		names = append(names, metrics[i].ID)
	}

	h.emitAudit(r, names)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	if len(bytes.TrimSpace(raw)) == 0 {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	var metric models.Metrics

	if err := json.Unmarshal(raw, &metric); err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	metric, err = h.service.GetMetric(r.Context(), metric)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBadMetric):
			writeJSONError(w, http.StatusBadRequest, "bad request")
		case errors.Is(err, service.ErrMetricNotFound):
			writeJSONError(w, http.StatusNotFound, "not found")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal error")
		}

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metric)
}
