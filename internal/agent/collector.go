package agent

import (
	"math/rand"
	"runtime"
	"strconv"
	"sync"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Collector struct {
	mu        sync.RWMutex
	gauges    map[string]float64
	counters  map[string]int64
	pollCount int64
}

func NewCollector() *Collector {
	return &Collector{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (c *Collector) CollectRuntime() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.pollCount++
	c.counters["PollCount"] = c.pollCount
	c.gauges["RandomValue"] = rand.Float64() * 100.0

	c.gauges["Alloc"] = float64(memStats.Alloc)
	c.gauges["BuckHashSys"] = float64(memStats.BuckHashSys)
	c.gauges["Frees"] = float64(memStats.Frees)
	c.gauges["GCCPUFraction"] = memStats.GCCPUFraction
	c.gauges["GCSys"] = float64(memStats.GCSys)
	c.gauges["HeapAlloc"] = float64(memStats.HeapAlloc)
	c.gauges["HeapIdle"] = float64(memStats.HeapIdle)
	c.gauges["HeapInuse"] = float64(memStats.HeapInuse)
	c.gauges["HeapObjects"] = float64(memStats.HeapObjects)
	c.gauges["HeapReleased"] = float64(memStats.HeapReleased)
	c.gauges["HeapSys"] = float64(memStats.HeapSys)
	c.gauges["LastGC"] = float64(memStats.LastGC)
	c.gauges["Lookups"] = float64(memStats.Lookups)
	c.gauges["MCacheInuse"] = float64(memStats.MCacheInuse)
	c.gauges["MCacheSys"] = float64(memStats.MCacheSys)
	c.gauges["MSpanInuse"] = float64(memStats.MSpanInuse)
	c.gauges["MSpanSys"] = float64(memStats.MSpanSys)
	c.gauges["Mallocs"] = float64(memStats.Mallocs)
	c.gauges["NextGC"] = float64(memStats.NextGC)
	c.gauges["NumForcedGC"] = float64(memStats.NumForcedGC)
	c.gauges["NumGC"] = float64(memStats.NumGC)
	c.gauges["OtherSys"] = float64(memStats.OtherSys)
	c.gauges["PauseTotalNs"] = float64(memStats.PauseTotalNs)
	c.gauges["StackInuse"] = float64(memStats.StackInuse)
	c.gauges["StackSys"] = float64(memStats.StackSys)
	c.gauges["Sys"] = float64(memStats.Sys)
	c.gauges["TotalAlloc"] = float64(memStats.TotalAlloc)
}

func (c *Collector) CollectGopsutil() {
	vm, err := mem.VirtualMemory()
	if err == nil {
		c.mu.Lock()
		c.gauges["TotalMemory"] = float64(vm.Total)
		c.gauges["FreeMemory"] = float64(vm.Free)
		c.mu.Unlock()
	}

	p, err := cpu.Percent(0, true)
	if err == nil {
		c.mu.Lock()
		for i := range p {
			c.gauges["CPUutilization"+strconv.Itoa(i+1)] = p[i]
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
