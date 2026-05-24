package agent

import (
	"context"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
	pb "github.com/LemuriiL/MetricsAllerts/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type GRPCClient struct {
	addr   string
	conn   *grpc.ClientConn
	client pb.MetricsClient
}

func NewGRPCClient(addr string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &GRPCClient{
		addr:   addr,
		conn:   conn,
		client: pb.NewMetricsClient(conn),
	}, nil
}

func (c *GRPCClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}

	return c.conn.Close()
}

func (c *GRPCClient) UpdateMetrics(metrics []models.Metrics) error {
	if c == nil || c.client == nil || len(metrics) == 0 {
		return nil
	}

	ip, err := resolveOutboundIP(c.addr, "9090")
	if err != nil {
		return err
	}

	items := make([]*pb.Metric, 0, len(metrics))

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			value := 0.0
			if metric.Value != nil {
				value = *metric.Value
			}

			items = append(items, &pb.Metric{
				Id:    metric.ID,
				Type:  pb.Metric_GAUGE,
				Value: value,
			})
		case models.Counter:
			delta := int64(0)
			if metric.Delta != nil {
				delta = *metric.Delta
			}

			items = append(items, &pb.Metric{
				Id:    metric.ID,
				Type:  pb.Metric_COUNTER,
				Delta: delta,
			})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", ip)

	_, err = c.client.UpdateMetrics(ctx, &pb.UpdateMetricsRequest{
		Metrics: items,
	})

	return err
}
