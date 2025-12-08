package agent

import (
	"time"
)

type Agent struct {
	collector  *Collector
	sender     *Sender
	pollTick   <-chan time.Time
	reportTick <-chan time.Time
	stopCh     chan struct{}
}

func NewAgent(serverAddr string, pollInterval, reportInterval time.Duration) *Agent {
	return &Agent{
		collector:  NewCollector(),
		sender:     NewSender(serverAddr),
		pollTick:   time.NewTicker(pollInterval).C,
		reportTick: time.NewTicker(reportInterval).C,
		stopCh:     make(chan struct{}),
	}
}

func (a *Agent) Stop() {
	close(a.stopCh)
}

func (a *Agent) Run() {
	for {
		select {
		case <-a.stopCh:
			return
		case <-a.pollTick:
			a.collector.Collect()
		case <-a.reportTick:
			metrics := a.collector.Collect()
			for _, m := range metrics {
				_ = a.sender.Send(m)
			}
		}
	}
}
