package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
)

type endpointNotSupportedError struct {
	status int
}

func (e endpointNotSupportedError) Error() string {
	return fmt.Sprintf("endpoint not supported: %d", e.status)
}

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

	u, err := url.JoinPath(s.serverAddr, "/update")
	if err != nil {
		return err
	}

	return s.postJSONWithRetry(context.Background(), u, body)
}

func (s *Sender) SendBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	u, err := url.JoinPath(s.serverAddr, "/updates")
	if err != nil {
		return err
	}

	err = s.postJSONWithRetry(context.Background(), u, body)
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

func (s *Sender) postJSONWithRetry(ctx context.Context, u string, body []byte) error {
	waits := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	err := s.postJSON(ctx, u, body)
	if err == nil {
		return nil
	}

	for i := 0; i < len(waits); i++ {
		if !isRetryableHTTPError(err) {
			return err
		}

		timer := time.NewTimer(waits[i])
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		err = s.postJSON(ctx, u, body)
		if err == nil {
			return nil
		}
	}

	return err
}

func (s *Sender) postJSON(ctx context.Context, u string, body []byte) error {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(body); err != nil {
		_ = gw.Close()
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &buf)
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
		return endpointNotSupportedError{status: resp.StatusCode}
	}

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status: %d body=%s", resp.StatusCode, string(b))
	}

	return nil
}

func isEndpointNotSupported(err error) bool {
	var e endpointNotSupportedError
	return errors.As(err, &e)
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
