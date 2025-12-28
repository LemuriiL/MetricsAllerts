package storage

type Storage interface {
	SetGauge(name string, value float64)
	GetGauge(name string) (float64, bool)
	SetCounter(name string, value int64)
	GetCounter(name string) (int64, bool)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
}

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

func (s *MemStorage) GetAllGauges() map[string]float64 {
	return s.gauges
}

func (s *MemStorage) GetAllCounters() map[string]int64 {
	return s.counters
}
