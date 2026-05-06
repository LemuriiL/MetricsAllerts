package agent

import (
	"context"
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
}

type sendJob struct {
	metrics []models.Metrics
}

func NewAgent(serverAddr string, pollInterval, reportInterval time.Duration) *Agent {
	return NewAgentWithKeyAndLimitAndCrypto(serverAddr, pollInterval, reportInterval, "", 1, "")
}

func NewAgentWithKey(serverAddr string, pollInterval, reportInterval time.Duration, key string) *Agent {
	return NewAgentWithKeyAndLimitAndCrypto(serverAddr, pollInterval, reportInterval, key, 1, "")
}

func NewAgentWithKeyAndLimit(serverAddr string, pollInterval, reportInterval time.Duration, key string, rateLimit int) *Agent {
	return NewAgentWithKeyAndLimitAndCrypto(serverAddr, pollInterval, reportInterval, key, rateLimit, "")
}

func NewAgentWithKeyAndLimitAndCrypto(serverAddr string, pollInterval, reportInterval time.Duration, key string, rateLimit int, cryptoKeyPath string) *Agent {
	if rateLimit <= 0 {
		rateLimit = 1
	}

	return &Agent{
		collector:      NewCollector(),
		sender:         NewSenderWithKeyAndCryptoKey(serverAddr, key, cryptoKeyPath),
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		rateLimit:      rateLimit,
	}
}

func (a *Agent) Run(ctx context.Context) {
	jobs := make(chan sendJob, a.rateLimit*2)

	var workersWG sync.WaitGroup

	for i := 0; i < a.rateLimit; i++ {
		workersWG.Add(1)

		go func() {
			defer workersWG.Done()

			for job := range jobs {
				if len(job.metrics) == 0 {
					continue
				}

				if err := a.sender.SendBatch(job.metrics); err != nil {
					log.Printf("failed to send batch: %v", err)
				}
			}
		}()
	}

	var producersWG sync.WaitGroup

	producersWG.Add(1)
	go func() {
		defer producersWG.Done()

		t := time.NewTicker(a.pollInterval)
		defer t.Stop()

		a.collector.CollectRuntime()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				a.collector.CollectRuntime()
			}
		}
	}()

	producersWG.Add(1)
	go func() {
		defer producersWG.Done()

		t := time.NewTicker(a.pollInterval)
		defer t.Stop()

		a.collector.CollectGopsutil()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				a.collector.CollectGopsutil()
			}
		}
	}()

	producersWG.Add(1)
	go func() {
		defer producersWG.Done()

		t := time.NewTicker(a.reportInterval)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				ms := a.collector.Snapshot()
				if len(ms) == 0 {
					continue
				}

				select {
				case jobs <- sendJob{metrics: ms}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	<-ctx.Done()

	finalMetrics := a.collector.Snapshot()

	producersWG.Wait()

	if len(finalMetrics) > 0 {
		jobs <- sendJob{metrics: finalMetrics}
	}

	close(jobs)
	workersWG.Wait()
}
