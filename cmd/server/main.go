package main

import (
	"net/http"
	"strconv"
	"strings"
)

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

type Storage interface {
	SetGauge(name string, value float64)
	GetGauge(name string) (float64, bool)
	SetCounter(name string, value int64)
	GetCounter(name string) (int64, bool)
}

func (s *MemStorage) SetGauge(name string, value float64) {
	s.gauges[name] = value
}

func (s *MemStorage) GetGauge(name string) (float64, bool) {
	val, ok := s.gauges[name]
	return val, ok
}

func (s *MemStorage) SetCounter(name string, value int64) {
	if old, exists := s.counters[name]; exists {
		value += old
	}
	s.counters[name] = value
}

func (s *MemStorage) GetCounter(name string) (int64, bool) {
	val, ok := s.counters[name]
	return val, ok
}

var store = NewMemStorage()

func main() {
	http.HandleFunc("/update/", updateHandler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.NotFound(w, r)
		return
	}

	metricType := pathParts[2]
	metricName := pathParts[3]
	metricValueStr := pathParts[4]

	if metricName == "" {
		http.NotFound(w, r)
		return
	}

	switch metricType {
	case "gauge":
		val, err := strconv.ParseFloat(metricValueStr, 64)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		store.SetGauge(metricName, val)
	case "counter":
		val, err := strconv.ParseInt(metricValueStr, 10, 64)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		store.SetCounter(metricName, val)
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("Accepted.\n\n"))
}
