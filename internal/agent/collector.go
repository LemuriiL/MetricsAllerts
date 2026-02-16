package agent

import (
	"math/rand"
	"runtime"
	"sync"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Collector struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64

	pollCount int64
}

func NewCollector() *Collector {
	return &Collector{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (c *Collector) setGauge(name string, v float64) {
	c.gauges[name] = v
}

func (c *Collector) setCounter(name string, v int64) {
	c.counters[name] = v
}

func (c *Collector) CollectRuntime() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.pollCount++
	c.setCounter("PollCount", c.pollCount)
	c.setGauge("RandomValue", rand.Float64()*100.0)

	c.setGauge("Alloc", float64(memStats.Alloc))
	c.setGauge("BuckHashSys", float64(memStats.BuckHashSys))
	c.setGauge("Frees", float64(memStats.Frees))
	c.setGauge("GCCPUFraction", memStats.GCCPUFraction)
	c.setGauge("GCSys", float64(memStats.GCSys))
	c.setGauge("HeapAlloc", float64(memStats.HeapAlloc))
	c.setGauge("HeapIdle", float64(memStats.HeapIdle))
	c.setGauge("HeapInuse", float64(memStats.HeapInuse))
	c.setGauge("HeapObjects", float64(memStats.HeapObjects))
	c.setGauge("HeapReleased", float64(memStats.HeapReleased))
	c.setGauge("HeapSys", float64(memStats.HeapSys))
	c.setGauge("LastGC", float64(memStats.LastGC))
	c.setGauge("Lookups", float64(memStats.Lookups))
	c.setGauge("MCacheInuse", float64(memStats.MCacheInuse))
	c.setGauge("MCacheSys", float64(memStats.MCacheSys))
	c.setGauge("MSpanInuse", float64(memStats.MSpanInuse))
	c.setGauge("MSpanSys", float64(memStats.MSpanSys))
	c.setGauge("Mallocs", float64(memStats.Mallocs))
	c.setGauge("NextGC", float64(memStats.NextGC))
	c.setGauge("NumForcedGC", float64(memStats.NumForcedGC))
	c.setGauge("NumGC", float64(memStats.NumGC))
	c.setGauge("OtherSys", float64(memStats.OtherSys))
	c.setGauge("PauseTotalNs", float64(memStats.PauseTotalNs))
	c.setGauge("StackInuse", float64(memStats.StackInuse))
	c.setGauge("StackSys", float64(memStats.StackSys))
	c.setGauge("Sys", float64(memStats.Sys))
	c.setGauge("TotalAlloc", float64(memStats.TotalAlloc))
}

func (c *Collector) CollectGopsutil() {
	vm, err := mem.VirtualMemory()
	if err == nil {
		c.mu.Lock()
		c.setGauge("TotalMemory", float64(vm.Total))
		c.setGauge("FreeMemory", float64(vm.Free))
		c.mu.Unlock()
	}

	p, err := cpu.Percent(0, true)
	if err == nil {
		c.mu.Lock()
		for i := range p {
			name := "CPUutilization" + itoa(i+1)
			c.setGauge(name, p[i])
		}
		c.mu.Unlock()
	}
}

func (c *Collector) Snapshot() []models.Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]models.Metrics, 0, len(c.gauges)+len(c.counters))
	for k, v := range c.gauges {
		val := v
		out = append(out, models.Metrics{
			ID:    k,
			MType: models.Gauge,
			Value: &val,
		})
	}
	for k, v := range c.counters {
		d := v
		out = append(out, models.Metrics{
			ID:    k,
			MType: models.Counter,
			Delta: &d,
		})
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + (n % 10))
		n /= 10
	}
	return string(b[i:])
}
