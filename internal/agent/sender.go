package agent

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/model"
)

type Sender struct {
	serverAddr string
	client     *http.Client
}

func NewSender(serverAddr string) *Sender {
	return &Sender{
		serverAddr: serverAddr,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *Sender) Send(metric models.Metrics) error {
	var valueStr string
	if metric.MType == models.Gauge && metric.Value != nil {
		valueStr = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
	} else if metric.MType == models.Counter && metric.Delta != nil {
		valueStr = strconv.FormatInt(*metric.Delta, 10)
	} else {
		return fmt.Errorf("invalid metric type or value: %s", metric.ID)
	}

	url := fmt.Sprintf("%s/update/%s/%s/%s", s.serverAddr, metric.MType, metric.ID, valueStr)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}
