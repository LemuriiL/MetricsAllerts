package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
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
	body, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/update", s.serverAddr)
	return s.postJSON(url, body)
}

func (s *Sender) SendBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/updates", s.serverAddr)
	err = s.postJSON(url, body)
	if err == nil {
		return nil
	}

	if isEndpointNotSupported(err) {
		for i := range metrics {
			if err2 := s.Send(metrics[i]); err2 != nil {
				return err2
			}
		}
		return nil
	}

	return err
}

func (s *Sender) postJSON(url string, body []byte) error {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(body); err != nil {
		_ = gw.Close()
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return fmt.Errorf("endpoint not supported: %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status: %d body=%s", resp.StatusCode, string(b))
	}

	return nil
}

func isEndpointNotSupported(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsStatus(msg, "404") || containsStatus(msg, "405") || containsText(msg, "endpoint not supported")
}

func containsStatus(s string, code string) bool {
	return containsText(s, code)
}

func containsText(s string, sub string) bool {
	return len(sub) > 0 && indexOf(s, sub) >= 0
}

func indexOf(s string, sub string) int {
	return bytes.Index([]byte(s), []byte(sub))
}

func metricToLegacyURL(serverAddr string, metric models.Metrics) (string, error) {
	var valueStr string
	if metric.MType == models.Gauge && metric.Value != nil {
		valueStr = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
	} else if metric.MType == models.Counter && metric.Delta != nil {
		valueStr = strconv.FormatInt(*metric.Delta, 10)
	} else {
		return "", fmt.Errorf("invalid metric type or value: %s", metric.ID)
	}
	return fmt.Sprintf("%s/update/%s/%s/%s", serverAddr, metric.MType, metric.ID, valueStr), nil
}
