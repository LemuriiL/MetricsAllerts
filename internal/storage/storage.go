package storage

import "sync"

type Storage interface {
	SetGauge(name string, value float64)
	GetGauge(name string) (float64, bool)
	SetCounter(name string, value int64)
	GetCounter(name string) (int64, bool)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
}

type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MemStorage) SetGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

func (s *MemStorage) GetGauge(name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.gauges[name]
	return val, ok
}

func (s *MemStorage) SetCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if old, exists := s.counters[name]; exists {
		value += old
	}
	s.counters[name] = value
}

func (s *MemStorage) GetCounter(name string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.counters[name]
	return val, ok
}

func (s *MemStorage) GetAllGauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		res[k] = v
	}
	return res
}

func (s *MemStorage) GetAllCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		res[k] = v
	}
	return res
}
