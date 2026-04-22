package storage

import (
	"context"
	"sync"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
)

type Storage interface {
	SetGauge(ctx context.Context, name string, value float64) error
	GetGauge(ctx context.Context, name string) (float64, bool, error)
	SetCounter(ctx context.Context, name string, value int64) error
	GetCounter(ctx context.Context, name string) (int64, bool, error)
	GetAllGauges(ctx context.Context) (map[string]float64, error)
	GetAllCounters(ctx context.Context) (map[string]int64, error)
}

type BatchUpdater interface {
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
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

func (s *MemStorage) SetGauge(ctx context.Context, name string, value float64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.gauges[name] = value
	return nil
}

func (s *MemStorage) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	select {
	case <-ctx.Done():
		return 0, false, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.gauges[name]
	return val, ok, nil
}

func (s *MemStorage) SetCounter(ctx context.Context, name string, value int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if old, exists := s.counters[name]; exists {
		value += old
	}
	s.counters[name] = value

	return nil
}

func (s *MemStorage) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	select {
	case <-ctx.Done():
		return 0, false, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.counters[name]
	return val, ok, nil
}

func (s *MemStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		res[k] = v
	}

	return res, nil
}

func (s *MemStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		res[k] = v
	}

	return res, nil
}
