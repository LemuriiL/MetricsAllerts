package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LemuriiL/MetricsAllerts/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSenderSendGauge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL)
	val := 42.5
	metric := models.Metrics{
		ID:    "TestGauge",
		MType: models.Gauge,
		Value: &val,
	}

	err := sender.Send(metric)
	assert.NoError(t, err)
}

func TestSenderSendCounter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL)
	delta := int64(100)
	metric := models.Metrics{
		ID:    "TestCounter",
		MType: models.Counter,
		Delta: &delta,
	}

	err := sender.Send(metric)
	assert.NoError(t, err)
}
