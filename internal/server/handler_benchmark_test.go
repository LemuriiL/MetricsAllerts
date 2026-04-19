package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

func BenchmarkUpdateMetricJSON(b *testing.B) {
	st := storage.NewMemStorage()
	h := NewHandler(st)

	body := []byte(`{"id":"Alloc","type":"gauge","value":123.45}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.UpdateMetricJSON(w, req)
	}
}

func BenchmarkGetMetricJSON(b *testing.B) {
	st := storage.NewMemStorage()
	st.SetGauge("Alloc", 123.45)
	h := NewHandler(st)

	body := []byte(`{"id":"Alloc","type":"gauge"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.GetMetricJSON(w, req)
	}
}

func BenchmarkUpdateMetricsJSON(b *testing.B) {
	st := storage.NewMemStorage()
	h := NewHandler(st)

	body := []byte(`[
		{"id":"Alloc","type":"gauge","value":123.45},
		{"id":"HeapAlloc","type":"gauge","value":456.78},
		{"id":"PollCount","type":"counter","delta":10}
	]`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.UpdateMetricsJSON(w, req)
	}
}
