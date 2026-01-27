package server

import (
	"encoding/json"
	"net/http"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
)

type batchUpdater interface {
	UpdateBatch([]models.Metrics) error
}

func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if m.ID == "" || m.MType == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch m.MType {
	case models.Gauge:
		if m.Value == nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		h.storage.SetGauge(m.ID, *m.Value)
	case models.Counter:
		if m.Delta == nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		h.storage.SetCounter(m.ID, *m.Delta)
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

func (h *Handler) UpdateMetricsJSON(w http.ResponseWriter, r *http.Request) {
	var ms []models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&ms); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if len(ms) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	if bu, ok := h.storage.(batchUpdater); ok {
		if err := bu.UpdateBatch(ms); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	for i := range ms {
		m := ms[i]
		if m.ID == "" || m.MType == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			h.storage.SetGauge(m.ID, *m.Value)
		case models.Counter:
			if m.Delta == nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			h.storage.SetCounter(m.ID, *m.Delta)
		default:
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	var req models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.MType == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch req.MType {
	case models.Gauge:
		if v, ok := h.storage.GetGauge(req.ID); ok {
			req.Value = &v
		} else {
			http.NotFound(w, r)
			return
		}
	case models.Counter:
		if v, ok := h.storage.GetCounter(req.ID); ok {
			req.Delta = &v
		} else {
			http.NotFound(w, r)
			return
		}
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(req)
}
