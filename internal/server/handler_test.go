package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LemuriiL/MetricsAllerts/internal/storage"
	"github.com/stretchr/testify/assert"
)

func setupRouter(handler *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /update/{type}/{name}/{value}", handler.UpdateMetric)
	mux.HandleFunc("GET /value/{type}/{name}", handler.GetMetricValue)
	mux.HandleFunc("GET /", handler.GetAllMetrics)

	return mux
}

func TestUpdateMetric(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemStorage()
	handler := NewHandler(store)
	router := setupRouter(handler)

	tests := []struct {
		name             string
		method           string
		url              string
		expectedStatus   int
		expectLocation   bool
		shouldCheckValue bool
	}{
		{"valid gauge", "POST", "/update/gauge/cpu/42.5", http.StatusOK, false, true},
		{"valid counter", "POST", "/update/counter/req/10", http.StatusOK, false, true},
		{"empty name", "POST", "/update/gauge//42.5", http.StatusMovedPermanently, true, false},
		{"invalid type", "POST", "/update/xxx/a/1", http.StatusBadRequest, false, false},
		{"invalid number", "POST", "/update/gauge/a/abc", http.StatusBadRequest, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectLocation {
				assert.Contains(t, w.Header(), "Location")
			} else {
				assert.NotContains(t, w.Header(), "Location")
			}
		})
	}

	val, ok, err := store.GetGauge(ctx, "cpu")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 42.5, val)

	counter, ok, err := store.GetCounter(ctx, "req")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(10), counter)
}

func TestGetMetricValue(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemStorage()

	_ = store.SetGauge(ctx, "mem", 1024.5)
	_ = store.SetCounter(ctx, "calls", 42)

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
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
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
	ctx := context.Background()
	store := storage.NewMemStorage()

	_ = store.SetGauge(ctx, "temp", 36.6)
	_ = store.SetCounter(ctx, "hits", 100)

	handler := NewHandler(store)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")

	body := w.Body.String()

	assert.Contains(t, body, "All Metrics")
	assert.Contains(t, body, "temp: 36.6")
	assert.Contains(t, body, "hits: 100")
}
