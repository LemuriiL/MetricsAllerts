package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
)

type FileStorage struct {
	base      *MemStorage
	path      string
	syncWrite bool
	mu        sync.RWMutex
}

func NewFileStorage(path string, syncWrite bool) *FileStorage {
	return &FileStorage{
		base:      NewMemStorage(),
		path:      path,
		syncWrite: syncWrite,
	}
}

func (s *FileStorage) SetGauge(ctx context.Context, name string, value float64) error {
	if err := s.base.SetGauge(ctx, name, value); err != nil {
		return err
	}

	if s.syncWrite {
		return s.Save(ctx)
	}

	return nil
}

func (s *FileStorage) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	return s.base.GetGauge(ctx, name)
}

func (s *FileStorage) SetCounter(ctx context.Context, name string, value int64) error {
	if err := s.base.SetCounter(ctx, name, value); err != nil {
		return err
	}

	if s.syncWrite {
		return s.Save(ctx)
	}

	return nil
}

func (s *FileStorage) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	return s.base.GetCounter(ctx, name)
}

func (s *FileStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	return s.base.GetAllGauges(ctx)
}

func (s *FileStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	return s.base.GetAllCounters(ctx)
}

func (s *FileStorage) Save(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gauges, err := s.base.GetAllGauges(ctx)
	if err != nil {
		return err
	}

	counters, err := s.base.GetAllCounters(ctx)
	if err != nil {
		return err
	}

	res := make([]models.Metrics, 0, len(gauges)+len(counters))

	for name, v := range gauges {
		val := v
		res = append(res, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &val,
		})
	}

	for name, v := range counters {
		delta := v
		res = append(res, models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &delta,
		})
	}

	data, err := json.Marshal(res)
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, s.path)
}

func (s *FileStorage) Restore(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	var items []models.Metrics
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}

	prev := s.syncWrite
	s.syncWrite = false

	for _, metric := range items {
		switch metric.MType {
		case models.Gauge:
			if metric.Value != nil {
				if err := s.base.SetGauge(ctx, metric.ID, *metric.Value); err != nil {
					s.syncWrite = prev
					return err
				}
			}
		case models.Counter:
			if metric.Delta != nil {
				if err := s.base.SetCounter(ctx, metric.ID, *metric.Delta); err != nil {
					s.syncWrite = prev
					return err
				}
			}
		}
	}

	s.syncWrite = prev
	return nil
}
