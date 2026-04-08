package agent

import (
	"log"
	"sync"
	"time"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
)

type Agent struct {
	collector      *Collector
	sender         *Sender
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type sendJob struct {
	metrics []models.Metrics
}

func NewAgent(serverAddr string, pollInterval, reportInterval time.Duration) *Agent {
	return NewAgentWithKeyAndLimit(serverAddr, pollInterval, reportInterval, "", 1)
}

func NewAgentWithKey(serverAddr string, pollInterval, reportInterval time.Duration, key string) *Agent {
	return NewAgentWithKeyAndLimit(serverAddr, pollInterval, reportInterval, key, 1)
}

func NewAgentWithKeyAndLimit(serverAddr string, pollInterval, reportInterval time.Duration, key string, rateLimit int) *Agent {
	if rateLimit <= 0 {
		rateLimit = 1
	}
	return &Agent{
		collector:      NewCollector(),
		sender:         NewSenderWithKey(serverAddr, key),
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		rateLimit:      rateLimit,
		stopCh:         make(chan struct{}),
	}
}

func (a *Agent) Stop() {
	select {
	case <-a.stopCh:
	default:
		close(a.stopCh)
	}
	a.wg.Wait()
}

func (a *Agent) Run() {
	jobs := make(chan sendJob, a.rateLimit*2)

	for i := 0; i < a.rateLimit; i++ {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			for {
				select {
				case <-a.stopCh:
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					if len(job.metrics) == 0 {
						continue
					}
					if err := a.sender.SendBatch(job.metrics); err != nil {
						log.Printf("failed to send batch: %v", err)
					}
				}
			}
		}()
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		t := time.NewTicker(a.pollInterval)
		defer t.Stop()

		a.collector.CollectRuntime()

		for {
			select {
			case <-a.stopCh:
				return
			case <-t.C:
				a.collector.CollectRuntime()
			}
		}
	}()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		t := time.NewTicker(a.pollInterval)
		defer t.Stop()

		a.collector.CollectGopsutil()

		for {
			select {
			case <-a.stopCh:
				return
			case <-t.C:
				a.collector.CollectGopsutil()
			}
		}
	}()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		t := time.NewTicker(a.reportInterval)
		defer t.Stop()

		for {
			select {
			case <-a.stopCh:
				return
			case <-t.C:
				ms := a.collector.Snapshot()
				select {
				case jobs <- sendJob{metrics: ms}:
				case <-a.stopCh:
					return
				}
			}
		}
	}()

	<-a.stopCh
	close(jobs)
	a.wg.Wait()
}
