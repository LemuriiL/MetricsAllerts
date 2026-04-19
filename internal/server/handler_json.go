package server

import (
	"bytes"
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

// UpdateMetricJSON обновляет метрику через json
func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
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

	h.emitAudit(r, []string{m.ID})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

// UpdateMetricsJSON обновляет сразу несколько метрик
func (h *Handler) UpdateMetricsJSON(w http.ResponseWriter, r *http.Request) {
	var ms []models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&ms); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	if len(ms) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	names := make([]string, 0, len(ms))

	if bu, ok := h.storage.(batchUpdater); ok {
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
			case models.Counter:
				if m.Delta == nil {
					writeJSONError(w, http.StatusBadRequest, "bad request")
					return
				}
			default:
				writeJSONError(w, http.StatusBadRequest, "bad request")
				return
			}
			names = append(names, m.ID)
		}

		if err := bu.UpdateBatch(ms); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		h.emitAudit(r, names)
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
		names = append(names, m.ID)
	}

	h.emitAudit(r, names)
	w.WriteHeader(http.StatusOK)
}

// GetMetricJSON возвращает метрику через json
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

	var req models.Metrics
	if err := json.Unmarshal(raw, &req); err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
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
