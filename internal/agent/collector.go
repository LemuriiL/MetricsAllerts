package agent

import (
	"github.com/LemuriiL/MetricsAllerts/internal/model"
	"math/rand"
	"runtime"
)

type Collector struct {
	pollCount   int64
	randomValue float64
}

func NewCollector() *Collector {
	return &Collector{
		pollCount:   0,
		randomValue: 0.0,
	}
}

func (c *Collector) Collect() []models.Metrics {
	c.pollCount++
	c.randomValue = rand.Float64() * 100.0

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	gauge := func(name string, value float64) models.Metrics {
		return models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		}
	}

	counter := func(name string, value int64) models.Metrics {
		return models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &value,
		}
	}

	metrics := []models.Metrics{
		gauge("Alloc", float64(memStats.Alloc)),
		gauge("BuckHashSys", float64(memStats.BuckHashSys)),
		gauge("Frees", float64(memStats.Frees)),
		gauge("GCCPUFraction", memStats.GCCPUFraction),
		gauge("GCSys", float64(memStats.GCSys)),
		gauge("HeapAlloc", float64(memStats.HeapAlloc)),
		gauge("HeapIdle", float64(memStats.HeapIdle)),
		gauge("HeapInuse", float64(memStats.HeapInuse)),
		gauge("HeapObjects", float64(memStats.HeapObjects)),
		gauge("HeapReleased", float64(memStats.HeapReleased)),
		gauge("HeapSys", float64(memStats.HeapSys)),
		gauge("LastGC", float64(memStats.LastGC)),
		gauge("Lookups", float64(memStats.Lookups)),
		gauge("MCacheInuse", float64(memStats.MCacheInuse)),
		gauge("MCacheSys", float64(memStats.MCacheSys)),
		gauge("MSpanInuse", float64(memStats.MSpanInuse)),
		gauge("MSpanSys", float64(memStats.MSpanSys)),
		gauge("Mallocs", float64(memStats.Mallocs)),
		gauge("NextGC", float64(memStats.NextGC)),
		gauge("NumForcedGC", float64(memStats.NumForcedGC)),
		gauge("NumGC", float64(memStats.NumGC)),
		gauge("OtherSys", float64(memStats.OtherSys)),
		gauge("PauseTotalNs", float64(memStats.PauseTotalNs)),
		gauge("StackInuse", float64(memStats.StackInuse)),
		gauge("StackSys", float64(memStats.StackSys)),
		gauge("Sys", float64(memStats.Sys)),
		gauge("TotalAlloc", float64(memStats.TotalAlloc)),
		gauge("RandomValue", c.randomValue),
		counter("PollCount", c.pollCount),
	}

	return metrics
}
