package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/LemuriiL/MetricsAllerts/internal/model"
)

func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	switch m.MType {
	case models.Gauge:
		if m.Value == nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		h.storage.SetGauge(m.ID, *m.Value)
	case models.Counter:
		if m.Delta == nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		h.storage.SetCounter(m.ID, *m.Delta)
	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	switch m.MType {
	case models.Gauge:
		val, ok := h.storage.GetGauge(m.ID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		m.Value = &val
	case models.Counter:
		val, ok := h.storage.GetCounter(m.ID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		m.Delta = &val
	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}
