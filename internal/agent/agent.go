package agent

import (
	"log"
	"time"
)

type Agent struct {
	collector    *Collector
	sender       *Sender
	pollTicker   *time.Ticker
	reportTicker *time.Ticker
	stopCh       chan struct{}
}

func NewAgent(serverAddr string, pollInterval, reportInterval time.Duration) *Agent {
	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)

	return &Agent{
		collector:    NewCollector(),
		sender:       NewSender(serverAddr),
		pollTicker:   pollTicker,
		reportTicker: reportTicker,
		stopCh:       make(chan struct{}),
	}
}

func (a *Agent) Stop() {
	if a.pollTicker != nil {
		a.pollTicker.Stop()
	}
	if a.reportTicker != nil {
		a.reportTicker.Stop()
	}
	select {
	case <-a.stopCh:
	default:
		close(a.stopCh)
	}
}

func (a *Agent) Run() {
	metrics := a.collector.Collect()
	for _, m := range metrics {
		if err := a.sender.Send(m); err != nil {
			log.Printf("failed to send metric %s: %v", m.ID, err)
		}
	}

	for {
		select {
		case <-a.stopCh:
			return
		case <-a.pollTicker.C:
			a.collector.Collect()
		case <-a.reportTicker.C:
			metrics := a.collector.Collect()
			for _, m := range metrics {
				if err := a.sender.Send(m); err != nil {
					log.Printf("failed to send metric %s: %v", m.ID, err)
				}
			}
		}
	}
}
