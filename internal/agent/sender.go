package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
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
	return s.postJSONWithRetry(url, body)
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
	err = s.postJSONWithRetry(url, body)
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

func (s *Sender) postJSONWithRetry(url string, body []byte) error {
	waits := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	err := s.postJSON(url, body)
	if err == nil {
		return nil
	}

	for i := 0; i < len(waits); i++ {
		if !isRetryableHTTPError(err) {
			return err
		}
		time.Sleep(waits[i])
		err = s.postJSON(url, body)
		if err == nil {
			return nil
		}
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
	s := err.Error()
	return bytes.Contains([]byte(s), []byte("404")) || bytes.Contains([]byte(s), []byte("405")) || bytes.Contains([]byte(s), []byte("endpoint not supported"))
}

func isRetryableHTTPError(err error) bool {
	if err == nil {
		return false
	}

	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}

	if errors.Is(err, io.EOF) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	msg := err.Error()
	if bytes.Contains([]byte(msg), []byte("connection refused")) {
		return true
	}
	if bytes.Contains([]byte(msg), []byte("EOF")) {
		return true
	}

	return false
}
