package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

func ExampleHandler_UpdateMetricJSON() {
	st := storage.NewMemStorage()
	h := NewHandler(st)

	body := []byte(`{"id":"Alloc","type":"gauge","value":123.45}`)
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateMetricJSON(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
}

func ExampleHandler_UpdateMetricsJSON() {
	ctx := context.Background()
	st := storage.NewMemStorage()
	h := NewHandler(st)

	body := []byte(`[{"id":"Alloc","type":"gauge","value":123.45},{"id":"PollCount","type":"counter","delta":10}]`)
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateMetricsJSON(w, req)

	gauge, _, _ := st.GetGauge(ctx, "Alloc")
	counter, _, _ := st.GetCounter(ctx, "PollCount")

	fmt.Println(w.Code)
	fmt.Printf("%.2f\n", gauge)
	fmt.Println(counter)
}

func ExampleHandler_GetMetricJSON() {
	ctx := context.Background()
	st := storage.NewMemStorage()
	_ = st.SetGauge(ctx, "Alloc", 123.45)
	h := NewHandler(st)

	body := []byte(`{"id":"Alloc","type":"gauge"}`)
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.GetMetricJSON(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
}

func ExampleHandler_UpdateMetric() {
	ctx := context.Background()
	st := storage.NewMemStorage()
	h := NewHandler(st)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	w := httptest.NewRecorder()

	h.UpdateMetric(w, req)

	value, _, _ := st.GetGauge(ctx, "Alloc")

	fmt.Println(w.Code)
	fmt.Printf("%.2f\n", value)
}

func ExampleHandler_GetMetricValue() {
	ctx := context.Background()
	st := storage.NewMemStorage()
	_ = st.SetCounter(ctx, "PollCount", 10)
	h := NewHandler(st)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	w := httptest.NewRecorder()

	h.GetMetricValue(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
}

func ExampleMetrics() {
	value := 123.45

	m := models.Metrics{
		ID:    "Alloc",
		MType: models.Gauge,
		Value: &value,
	}

	fmt.Println(m.ID)
	fmt.Println(m.MType)
	fmt.Printf("%.2f\n", *m.Value)
}
