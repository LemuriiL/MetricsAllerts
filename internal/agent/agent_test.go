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
	collector.CollectRuntime()
	collector.CollectGopsutil()
	metrics := collector.Snapshot()

	assert.NotEmpty(t, metrics)
}

func TestAgentRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewAgentWithKeyAndLimit(server.URL, 50*time.Millisecond, 100*time.Millisecond, "", 2)

	done := make(chan bool)
	go func() {
		agent.Run()
		done <- true
	}()

	time.Sleep(250 * time.Millisecond)
	agent.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("agent did not stop")
	}
}
