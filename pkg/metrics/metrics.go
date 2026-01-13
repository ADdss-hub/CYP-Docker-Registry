// Package metrics provides monitoring metrics for CYP-Registry.
package metrics

import (
	"sync"
	"time"
)

// Metrics holds application metrics.
type Metrics struct {
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	mu         sync.RWMutex
}

// Counter represents a monotonically increasing counter.
type Counter struct {
	name   string
	value  int64
	labels map[string]string
	mu     sync.Mutex
}

// Gauge represents a value that can go up and down.
type Gauge struct {
	name   string
	value  float64
	labels map[string]string
	mu     sync.Mutex
}

// Histogram represents a distribution of values.
type Histogram struct {
	name    string
	buckets []float64
	counts  []int64
	sum     float64
	count   int64
	labels  map[string]string
	mu      sync.Mutex
}

var (
	globalMetrics *Metrics
	once          sync.Once
)

// Get returns the global metrics instance.
func Get() *Metrics {
	once.Do(func() {
		globalMetrics = &Metrics{
			counters:   make(map[string]*Counter),
			gauges:     make(map[string]*Gauge),
			histograms: make(map[string]*Histogram),
		}
	})
	return globalMetrics
}

// NewCounter creates a new counter.
func (m *Metrics) NewCounter(name string, labels map[string]string) *Counter {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := name + labelsToKey(labels)
	if c, ok := m.counters[key]; ok {
		return c
	}

	c := &Counter{
		name:   name,
		labels: labels,
	}
	m.counters[key] = c
	return c
}

// NewGauge creates a new gauge.
func (m *Metrics) NewGauge(name string, labels map[string]string) *Gauge {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := name + labelsToKey(labels)
	if g, ok := m.gauges[key]; ok {
		return g
	}

	g := &Gauge{
		name:   name,
		labels: labels,
	}
	m.gauges[key] = g
	return g
}

// NewHistogram creates a new histogram.
func (m *Metrics) NewHistogram(name string, buckets []float64, labels map[string]string) *Histogram {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := name + labelsToKey(labels)
	if h, ok := m.histograms[key]; ok {
		return h
	}

	h := &Histogram{
		name:    name,
		buckets: buckets,
		counts:  make([]int64, len(buckets)+1),
		labels:  labels,
	}
	m.histograms[key] = h
	return h
}

// Inc increments the counter by 1.
func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

// Add adds a value to the counter.
func (c *Counter) Add(v int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += v
}

// Value returns the current counter value.
func (c *Counter) Value() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

// Set sets the gauge value.
func (g *Gauge) Set(v float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = v
}

// Inc increments the gauge by 1.
func (g *Gauge) Inc() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value++
}

// Dec decrements the gauge by 1.
func (g *Gauge) Dec() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value--
}

// Add adds a value to the gauge.
func (g *Gauge) Add(v float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value += v
}

// Value returns the current gauge value.
func (g *Gauge) Value() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.value
}

// Observe records a value in the histogram.
func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.sum += v
	h.count++

	for i, bucket := range h.buckets {
		if v <= bucket {
			h.counts[i]++
			return
		}
	}
	h.counts[len(h.buckets)]++
}

// Sum returns the sum of all observed values.
func (h *Histogram) Sum() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sum
}

// Count returns the count of all observed values.
func (h *Histogram) Count() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count
}

// labelsToKey converts labels to a string key.
func labelsToKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	key := "{"
	for k, v := range labels {
		key += k + "=" + v + ","
	}
	return key[:len(key)-1] + "}"
}

// Pre-defined metrics
var (
	// HTTP metrics
	HTTPRequestsTotal     *Counter
	HTTPRequestDuration   *Histogram
	HTTPActiveConnections *Gauge

	// Registry metrics
	RegistryPushTotal    *Counter
	RegistryPullTotal    *Counter
	RegistryStorageBytes *Gauge

	// Security metrics
	AuthAttemptsTotal   *Counter
	AuthFailuresTotal   *Counter
	LockEventsTotal     *Counter
	IntrusionEventsTotal *Counter
)

// InitMetrics initializes pre-defined metrics.
func InitMetrics() {
	m := Get()

	HTTPRequestsTotal = m.NewCounter("http_requests_total", nil)
	HTTPRequestDuration = m.NewHistogram("http_request_duration_seconds", []float64{0.01, 0.05, 0.1, 0.5, 1, 5}, nil)
	HTTPActiveConnections = m.NewGauge("http_active_connections", nil)

	RegistryPushTotal = m.NewCounter("registry_push_total", nil)
	RegistryPullTotal = m.NewCounter("registry_pull_total", nil)
	RegistryStorageBytes = m.NewGauge("registry_storage_bytes", nil)

	AuthAttemptsTotal = m.NewCounter("auth_attempts_total", nil)
	AuthFailuresTotal = m.NewCounter("auth_failures_total", nil)
	LockEventsTotal = m.NewCounter("lock_events_total", nil)
	IntrusionEventsTotal = m.NewCounter("intrusion_events_total", nil)
}

// Timer is a helper for timing operations.
type Timer struct {
	start     time.Time
	histogram *Histogram
}

// NewTimer creates a new timer.
func NewTimer(h *Histogram) *Timer {
	return &Timer{
		start:     time.Now(),
		histogram: h,
	}
}

// ObserveDuration records the duration since the timer was created.
func (t *Timer) ObserveDuration() {
	if t.histogram != nil {
		t.histogram.Observe(time.Since(t.start).Seconds())
	}
}
