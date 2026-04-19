package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"

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
	st := storage.NewMemStorage()
	h := NewHandler(st)

	body := []byte(`[{"id":"Alloc","type":"gauge","value":123.45},{"id":"PollCount","type":"counter","delta":10}]`)
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateMetricsJSON(w, req)

	gauge, _ := st.GetGauge("Alloc")
	counter, _ := st.GetCounter("PollCount")

	fmt.Println(w.Code)
	fmt.Printf("%.2f\n", gauge)
	fmt.Println(counter)
}

func ExampleHandler_GetMetricJSON() {
	st := storage.NewMemStorage()
	st.SetGauge("Alloc", 123.45)
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
	st := storage.NewMemStorage()
	h := NewHandler(st)

	r := mux.NewRouter()
	r.HandleFunc("/update/{type}/{name}/{value}", h.UpdateMetric).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	value, _ := st.GetGauge("Alloc")

	fmt.Println(w.Code)
	fmt.Printf("%.2f\n", value)
}

func ExampleHandler_GetMetricValue() {
	st := storage.NewMemStorage()
	st.SetCounter("PollCount", 10)
	h := NewHandler(st)

	r := mux.NewRouter()
	r.HandleFunc("/value/{type}/{name}", h.GetMetricValue).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

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
