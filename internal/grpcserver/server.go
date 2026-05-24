package grpcserver

import (
	"context"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
	pb "github.com/LemuriiL/MetricsAllerts/internal/proto"
)

type metricUpdater interface {
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
}

type MetricsServer struct {
	pb.UnimplementedMetricsServer
	updater metricUpdater
}

func New(updater metricUpdater) *MetricsServer {
	return &MetricsServer{updater: updater}
}

func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	if req == nil || len(req.Metrics) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	metrics := make([]models.Metrics, 0, len(req.Metrics))

	for _, item := range req.Metrics {
		if item == nil {
			continue
		}

		switch item.Type {
		case pb.Metric_GAUGE:
			value := item.Value
			metrics = append(metrics, models.Metrics{
				ID:    item.Id,
				MType: models.Gauge,
				Value: &value,
			})
		case pb.Metric_COUNTER:
			delta := item.Delta
			metrics = append(metrics, models.Metrics{
				ID:    item.Id,
				MType: models.Counter,
				Delta: &delta,
			})
		}
	}

	if err := s.updater.UpdateBatch(ctx, metrics); err != nil {
		return nil, err
	}

	return &pb.UpdateMetricsResponse{}, nil
}
