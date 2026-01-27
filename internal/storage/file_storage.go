package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/LemuriiL/MetricsAllerts/internal/model"
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

func (s *FileStorage) SetGauge(name string, value float64) {
	s.base.SetGauge(name, value)
	if s.syncWrite {
		s.Save()
	}
}

func (s *FileStorage) GetGauge(name string) (float64, bool) {
	return s.base.GetGauge(name)
}

func (s *FileStorage) SetCounter(name string, value int64) {
	s.base.SetCounter(name, value)
	if s.syncWrite {
		s.Save()
	}
}

func (s *FileStorage) GetCounter(name string) (int64, bool) {
	return s.base.GetCounter(name)
}

func (s *FileStorage) GetAllGauges() map[string]float64 {
	return s.base.GetAllGauges()
}

func (s *FileStorage) GetAllCounters() map[string]int64 {
	return s.base.GetAllCounters()
}

func (s *FileStorage) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	gauges := s.base.GetAllGauges()
	counters := s.base.GetAllCounters()

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
		d := v
		res = append(res, models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &d,
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

func (s *FileStorage) Restore() error {
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
	for _, m := range items {
		switch m.MType {
		case models.Gauge:
			if m.Value != nil {
				s.base.SetGauge(m.ID, *m.Value)
			}
		case models.Counter:
			if m.Delta != nil {
				s.base.SetCounter(m.ID, *m.Delta)
			}
		}
	}
	s.syncWrite = prev

	return nil
}
