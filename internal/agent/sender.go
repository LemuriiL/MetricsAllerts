package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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
	body, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/update", s.serverAddr)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

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
