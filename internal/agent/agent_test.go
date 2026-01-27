package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCollectorCollect(t *testing.T) {
	collector := NewCollector()
	metrics := collector.Collect()

	assert.NotEmpty(t, metrics)
	foundPollCount := false
	foundRandomValue := false
	for _, m := range metrics {
		if m.ID == "PollCount" {
			foundPollCount = true
			assert.Equal(t, "counter", m.MType)
			assert.NotNil(t, m.Delta)
		}
		if m.ID == "RandomValue" {
			foundRandomValue = true
			assert.Equal(t, "gauge", m.MType)
			assert.NotNil(t, m.Value)
		}
	}
	assert.True(t, foundPollCount)
	assert.True(t, foundRandomValue)
}

func TestAgentRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewAgent(server.URL, 50*time.Millisecond, 100*time.Millisecond)

	done := make(chan bool)
	go func() {
		agent.Run()
		done <- true
	}()

	time.Sleep(250 * time.Millisecond)
	agent.Stop()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("agent did not stop")
	}
}
