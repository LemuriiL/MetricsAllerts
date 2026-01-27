package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LemuriiL/MetricsAllerts/internal/storage"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

type mockStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func (m *mockStorage) SetGauge(name string, value float64) {
	m.gauges[name] = value
}

func (m *mockStorage) GetGauge(name string) (float64, bool) {
	v, ok := m.gauges[name]
	return v, ok
}

func (m *mockStorage) SetCounter(name string, value int64) {
	if old, ok := m.counters[name]; ok {
		value += old
	}
	m.counters[name] = value
}

func (m *mockStorage) GetCounter(name string) (int64, bool) {
	v, ok := m.counters[name]
	return v, ok
}

func (m *mockStorage) GetAllGauges() map[string]float64 {
	return m.gauges
}

func (m *mockStorage) GetAllCounters() map[string]int64 {
	return m.counters
}

func newMockStorage() storage.Storage {
	return &mockStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func setupRouter(handler *Handler) *mux.Router {
	r := mux.NewRouter()
	r.SkipClean(true)
	r.HandleFunc("/update/{type}/{name}/{value}", handler.UpdateMetric).Methods("POST")
	r.HandleFunc("/value/{type}/{name}", handler.GetMetricValue).Methods("GET")
	r.HandleFunc("/", handler.GetAllMetrics).Methods("GET")
	return r
}

func TestUpdateMetric(t *testing.T) {
	store := newMockStorage()
	handler := NewHandler(store)
	router := setupRouter(handler)

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
	}{
		{"valid gauge", "POST", "/update/gauge/cpu/42.5", http.StatusOK},
		{"valid counter", "POST", "/update/counter/req/10", http.StatusOK},
		{"empty name", "POST", "/update/gauge//42.5", http.StatusNotFound},
		{"invalid type", "POST", "/update/xxx/a/1", http.StatusBadRequest},
		{"invalid number", "POST", "/update/gauge/a/abc", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.NotContains(t, w.Header(), "Location")
		})
	}

	if val, ok := store.GetGauge("cpu"); ok {
		assert.Equal(t, 42.5, val)
	}
	if val, ok := store.GetCounter("req"); ok {
		assert.Equal(t, int64(10), val)
	}
}

func TestGetMetricValue(t *testing.T) {
	store := newMockStorage()
	store.SetGauge("mem", 1024.5)
	store.SetCounter("calls", 42)

	handler := NewHandler(store)
	router := setupRouter(handler)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{"existing gauge", "/value/gauge/mem", http.StatusOK, "1024.5"},
		{"existing counter", "/value/counter/calls", http.StatusOK, "42"},
		{"non-existing gauge", "/value/gauge/unknown", http.StatusNotFound, ""},
		{"non-existing counter", "/value/counter/unknown", http.StatusNotFound, ""},
		{"invalid type", "/value/xxx/a", http.StatusBadRequest, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, strings.TrimSuffix(w.Body.String(), "\n"))
			}
			assert.NotContains(t, w.Header(), "Location")
		})
	}
}

func TestGetAllMetrics(t *testing.T) {
	store := newMockStorage()
	store.SetGauge("temp", 36.6)
	store.SetCounter("hits", 100)

	handler := NewHandler(store)
	router := setupRouter(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	body := w.Body.String()
	assert.Contains(t, body, "<h1>All Metrics</h1>")
	assert.Contains(t, body, "temp: 36.6")
	assert.Contains(t, body, "hits: 100")
}
