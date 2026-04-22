package service

import (
	"context"
	"errors"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

var ErrMetricNotFound = errors.New("metric not found")
var ErrBadMetric = errors.New("bad metric")

type MetricsService struct {
	repository storage.Storage
}

func NewMetricsService(repository storage.Storage) *MetricsService {
	return &MetricsService{repository: repository}
}

func (s *MetricsService) SetGauge(ctx context.Context, name string, value float64) error {
	if name == "" {
		return ErrBadMetric
	}

	return s.repository.SetGauge(ctx, name, value)
}

func (s *MetricsService) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	if name == "" {
		return 0, false, ErrBadMetric
	}

	return s.repository.GetGauge(ctx, name)
}

func (s *MetricsService) SetCounter(ctx context.Context, name string, value int64) error {
	if name == "" {
		return ErrBadMetric
	}

	return s.repository.SetCounter(ctx, name, value)
}

func (s *MetricsService) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	if name == "" {
		return 0, false, ErrBadMetric
	}

	return s.repository.GetCounter(ctx, name)
}

func (s *MetricsService) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	return s.repository.GetAllGauges(ctx)
}

func (s *MetricsService) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	return s.repository.GetAllCounters(ctx)
}

func (s *MetricsService) UpdateMetric(ctx context.Context, metric models.Metrics) error {
	if metric.ID == "" || metric.MType == "" {
		return ErrBadMetric
	}

	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return ErrBadMetric
		}

		return s.repository.SetGauge(ctx, metric.ID, *metric.Value)
	case models.Counter:
		if metric.Delta == nil {
			return ErrBadMetric
		}

		return s.repository.SetCounter(ctx, metric.ID, *metric.Delta)
	default:
		return ErrBadMetric
	}
}

func (s *MetricsService) UpdateBatch(ctx context.Context, metrics []models.Metrics) error {
	for i := range metrics {
		metric := metrics[i]

		if metric.ID == "" || metric.MType == "" {
			return ErrBadMetric
		}

		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return ErrBadMetric
			}
		case models.Counter:
			if metric.Delta == nil {
				return ErrBadMetric
			}
		default:
			return ErrBadMetric
		}
	}

	if batchRepository, ok := s.repository.(storage.BatchUpdater); ok {
		return batchRepository.UpdateBatch(ctx, metrics)
	}

	for i := range metrics {
		if err := s.UpdateMetric(ctx, metrics[i]); err != nil {
			return err
		}
	}

	return nil
}

func (s *MetricsService) GetMetric(ctx context.Context, metric models.Metrics) (models.Metrics, error) {
	if metric.ID == "" || metric.MType == "" {
		return metric, ErrBadMetric
	}

	switch metric.MType {
	case models.Gauge:
		value, ok, err := s.repository.GetGauge(ctx, metric.ID)
		if err != nil {
			return metric, err
		}
		if !ok {
			return metric, ErrMetricNotFound
		}

		metric.Value = &value
		return metric, nil
	case models.Counter:
		delta, ok, err := s.repository.GetCounter(ctx, metric.ID)
		if err != nil {
			return metric, err
		}
		if !ok {
			return metric, ErrMetricNotFound
		}

		metric.Delta = &delta
		return metric, nil
	default:
		return metric, ErrBadMetric
	}
}
