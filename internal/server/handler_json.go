package server

import (
	"encoding/json"
	"io"
	"net/http"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
)

type batchUpdater interface {
	UpdateBatch([]models.Metrics) error
}

type errResp struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errResp{Error: msg})
}

func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics
	err := json.NewDecoder(r.Body).Decode(&m)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	if m.ID == "" || m.MType == "" {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	switch m.MType {
	case models.Gauge:
		if m.Value == nil {
			writeJSONError(w, http.StatusBadRequest, "bad request")
			return
		}
		h.storage.SetGauge(m.ID, *m.Value)
	case models.Counter:
		if m.Delta == nil {
			writeJSONError(w, http.StatusBadRequest, "bad request")
			return
		}
		h.storage.SetCounter(m.ID, *m.Delta)
	default:
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

func (h *Handler) UpdateMetricsJSON(w http.ResponseWriter, r *http.Request) {
	var ms []models.Metrics
	err := json.NewDecoder(r.Body).Decode(&ms)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	if len(ms) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	if bu, ok := h.storage.(batchUpdater); ok {
		if err := bu.UpdateBatch(ms); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	for i := range ms {
		m := ms[i]
		if m.ID == "" || m.MType == "" {
			writeJSONError(w, http.StatusBadRequest, "bad request")
			return
		}
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				writeJSONError(w, http.StatusBadRequest, "bad request")
				return
			}
			h.storage.SetGauge(m.ID, *m.Value)
		case models.Counter:
			if m.Delta == nil {
				writeJSONError(w, http.StatusBadRequest, "bad request")
				return
			}
			h.storage.SetCounter(m.ID, *m.Delta)
		default:
			writeJSONError(w, http.StatusBadRequest, "bad request")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	var req models.Metrics
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		if err == io.EOF {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	if req.ID == "" || req.MType == "" {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	switch req.MType {
	case models.Gauge:
		if v, ok := h.storage.GetGauge(req.ID); ok {
			req.Value = &v
		} else {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
	case models.Counter:
		if v, ok := h.storage.GetCounter(req.ID); ok {
			req.Delta = &v
		} else {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
	default:
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(req)
}
