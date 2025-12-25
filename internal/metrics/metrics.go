package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	// Counters
	requestsTotal  map[string]*atomic.Int64
	errorsTotal    map[string]*atomic.Int64
	rateLimitHits  map[string]*atomic.Int64
	apiKeyRequests map[string]*atomic.Int64

	// Gauges
	upstreamHealth   map[string]*atomic.Int64
	requestsInFlight map[string]*atomic.Int64

	// Histograms
	requestDuration  map[string]*histogram
	upstreamDuration map[string]*histogram

	buckets []float64
	mu      sync.RWMutex
}

type histogram struct {
	buckets []float64
	counts  []atomic.Int64
	sum     atomic.Int64 // microseconds
	count   atomic.Int64
}

func newHistogram(buckets []float64) *histogram {
	return &histogram{
		buckets: buckets,
		counts:  make([]atomic.Int64, len(buckets)+1),
	}
}

func (h *histogram) observe(value float64) {
	idx := sort.SearchFloat64s(h.buckets, value)
	h.counts[idx].Add(1)
	h.sum.Add(int64(value * 1e6)) // convert to microseconds
	h.count.Add(1)
}

type Config struct {
	LatencyBuckets []float64
}

func DefaultConfig() Config {
	return Config{
		LatencyBuckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
	}
}

func New(cfg Config) *Metrics {
	if len(cfg.LatencyBuckets) == 0 {
		cfg = DefaultConfig()
	}

	return &Metrics{
		requestsTotal:    make(map[string]*atomic.Int64),
		errorsTotal:      make(map[string]*atomic.Int64),
		rateLimitHits:    make(map[string]*atomic.Int64),
		apiKeyRequests:   make(map[string]*atomic.Int64),
		upstreamHealth:   make(map[string]*atomic.Int64),
		requestsInFlight: make(map[string]*atomic.Int64),
		requestDuration:  make(map[string]*histogram),
		upstreamDuration: make(map[string]*histogram),
		buckets:          cfg.LatencyBuckets,
	}
}

func (m *Metrics) getOrCreateCounter(counters map[string]*atomic.Int64, key string) *atomic.Int64 {
	m.mu.RLock()
	counter, ok := counters[key]
	m.mu.RUnlock()

	if !ok {
		m.mu.Lock()
		counter, ok = counters[key]
		if !ok {
			counter = &atomic.Int64{}
			counters[key] = counter
		}
		m.mu.Unlock()
	}
	return counter
}

func (m *Metrics) getOrCreateHistogram(histograms map[string]*histogram, key string) *histogram {
	m.mu.RLock()
	hist, ok := histograms[key]
	m.mu.RUnlock()

	if !ok {
		m.mu.Lock()
		hist, ok = histograms[key]
		if !ok {
			hist = newHistogram(m.buckets)
			histograms[key] = hist
		}
		m.mu.Unlock()
	}
	return hist
}

// Handler returns an HTTP handler that serves the metrics in Prometheus format
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		m.writePrometheusMetrics(w)
	})
}

func (m *Metrics) writePrometheusMetrics(w http.ResponseWriter) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Write request counters
	fmt.Fprintln(w, "# HELP gateway_requests_total Total number of requests processed")
	fmt.Fprintln(w, "# TYPE gateway_requests_total counter")
	for key, counter := range m.requestsTotal {
		fmt.Fprintf(w, "gateway_requests_total{key=\"%s\"} %d\n", key, counter.Load())
	}

	// Write error counters
	fmt.Fprintln(w, "# HELP gateway_errors_total Total number of errors")
	fmt.Fprintln(w, "# TYPE gateway_errors_total counter")
	for key, counter := range m.errorsTotal {
		fmt.Fprintf(w, "gateway_errors_total{key=\"%s\"} %d\n", key, counter.Load())
	}

	// Write rate limit counters
	fmt.Fprintln(w, "# HELP gateway_rate_limit_hits_total Total number of rate limit hits")
	fmt.Fprintln(w, "# TYPE gateway_rate_limit_hits_total counter")
	for key, counter := range m.rateLimitHits {
		fmt.Fprintf(w, "gateway_rate_limit_hits_total{key=\"%s\"} %d\n", key, counter.Load())
	}

	// Write API key request counters
	fmt.Fprintln(w, "# HELP gateway_api_key_requests_total Total requests per API key")
	fmt.Fprintln(w, "# TYPE gateway_api_key_requests_total counter")
	for key, counter := range m.apiKeyRequests {
		fmt.Fprintf(w, "gateway_api_key_requests_total{key=\"%s\"} %d\n", key, counter.Load())
	}

	// Write upstream health
	fmt.Fprintln(w, "# HELP gateway_upstream_healthy Whether upstream is healthy")
	fmt.Fprintln(w, "# TYPE gateway_upstream_healthy gauge")
	for key, gauge := range m.upstreamHealth {
		fmt.Fprintf(w, "gateway_upstream_healthy{key=\"%s\"} %d\n", key, gauge.Load())
	}

	// Write in-flight requests
	fmt.Fprintln(w, "# HELP gateway_requests_in_flight Number of requests in flight")
	fmt.Fprintln(w, "# TYPE gateway_requests_in_flight gauge")
	for key, gauge := range m.requestsInFlight {
		fmt.Fprintf(w, "gateway_requests_in_flight{key=\"%s\"} %d\n", key, gauge.Load())
	}

	// Write request duration histogram
	fmt.Fprintln(w, "# HELP gateway_request_duration_seconds Request duration in seconds")
	fmt.Fprintln(w, "# TYPE gateway_request_duration_seconds histogram")
	for key, hist := range m.requestDuration {
		var cumulative int64
		for i, bucket := range hist.buckets {
			cumulative += hist.counts[i].Load()
			fmt.Fprintf(w, "gateway_request_duration_seconds_bucket{key=\"%s\",le=\"%v\"} %d\n",
				key, bucket, cumulative)
		}
		cumulative += hist.counts[len(hist.buckets)].Load()
		fmt.Fprintf(w, "gateway_request_duration_seconds_bucket{key=\"%s\",le=\"+Inf\"} %d\n", key, cumulative)
		fmt.Fprintf(w, "gateway_request_duration_seconds_sum{key=\"%s\"} %f\n", key, float64(hist.sum.Load())/1e6)
		fmt.Fprintf(w, "gateway_request_duration_seconds_count{key=\"%s\"} %d\n", key, hist.count.Load())
	}
}

func (m *Metrics) RecordRequest(route, method string, status int, duration time.Duration) {
	key := route + "_" + method + "_" + strconv.Itoa(status)
	m.getOrCreateCounter(m.requestsTotal, key).Add(1)

	histKey := route + "_" + method
	m.getOrCreateHistogram(m.requestDuration, histKey).observe(duration.Seconds())
}

func (m *Metrics) RecordError(route, errorType string) {
	key := route + "_" + errorType
	m.getOrCreateCounter(m.errorsTotal, key).Add(1)
}

func (m *Metrics) RecordRateLimitHit(route, limitType string) {
	key := route + "_" + limitType
	m.getOrCreateCounter(m.rateLimitHits, key).Add(1)
}

func (m *Metrics) RecordUpstreamHealth(upstream, target string, healthy bool) {
	key := upstream + "_" + target
	val := int64(0)
	if healthy {
		val = 1
	}
	gauge := m.getOrCreateCounter(m.upstreamHealth, key)
	gauge.Store(val)
}

func (m *Metrics) RecordUpstreamDuration(upstream string, duration time.Duration) {
	m.getOrCreateHistogram(m.upstreamDuration, upstream).observe(duration.Seconds())
}

func (m *Metrics) RecordAPIKeyRequest(keyName string, status int) {
	key := keyName + "_" + strconv.Itoa(status)
	m.getOrCreateCounter(m.apiKeyRequests, key).Add(1)
}

func (m *Metrics) InFlightRequests(route string) func() {
	gauge := m.getOrCreateCounter(m.requestsInFlight, route)
	gauge.Add(1)
	return func() {
		gauge.Add(-1)
	}
}

type UsageTracker struct {
	requestCounts map[string]*atomic.Int64
	errorCounts   map[string]*atomic.Int64
	latencies     map[string]*LatencyTracker
	mu            sync.RWMutex
}

type LatencyTracker struct {
	samples []float64
	maxSize int
	mu      sync.Mutex
}

func NewLatencyTracker(maxSize int) *LatencyTracker {
	return &LatencyTracker{
		samples: make([]float64, 0, maxSize),
		maxSize: maxSize,
	}
}

func (lt *LatencyTracker) Record(duration time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if len(lt.samples) >= lt.maxSize {
		lt.samples = lt.samples[lt.maxSize/4:]
	}

	lt.samples = append(lt.samples, duration.Seconds())
}

func (lt *LatencyTracker) Percentile(p float64) float64 {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if len(lt.samples) == 0 {
		return 0
	}

	sorted := make([]float64, len(lt.samples))
	copy(sorted, lt.samples)
	sort.Float64s(sorted)

	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		requestCounts: make(map[string]*atomic.Int64),
		errorCounts:   make(map[string]*atomic.Int64),
		latencies:     make(map[string]*LatencyTracker),
	}
}

func (ut *UsageTracker) getOrCreateCounter(counters map[string]*atomic.Int64, key string) *atomic.Int64 {
	ut.mu.RLock()
	counter, ok := counters[key]
	ut.mu.RUnlock()

	if !ok {
		ut.mu.Lock()
		counter, ok = counters[key]
		if !ok {
			counter = &atomic.Int64{}
			counters[key] = counter
		}
		ut.mu.Unlock()
	}
	return counter
}

func (ut *UsageTracker) RecordRequest(key string, duration time.Duration, isError bool) {
	ut.getOrCreateCounter(ut.requestCounts, key).Add(1)
	if isError {
		ut.getOrCreateCounter(ut.errorCounts, key).Add(1)
	}

	ut.mu.Lock()
	if _, ok := ut.latencies[key]; !ok {
		ut.latencies[key] = NewLatencyTracker(1000)
	}

	lt := ut.latencies[key]
	ut.mu.Unlock()

	lt.Record(duration)
}

type Stats struct {
	Key          string  `json:"key"`
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	P50Latency   float64 `json:"p50_latency_ms"`
	P90Latency   float64 `json:"p90_latency_ms"`
	P99Latency   float64 `json:"p99_latency_ms"`
}

func (ut *UsageTracker) GetStats() []Stats {
	ut.mu.RLock()
	defer ut.mu.RUnlock()

	stats := make([]Stats, 0, len(ut.requestCounts))
	for key, counter := range ut.requestCounts {
		s := Stats{
			Key:          key,
			RequestCount: counter.Load(),
		}
		if ec, ok := ut.errorCounts[key]; ok {
			s.ErrorCount = ec.Load()
		}
		if lt, ok := ut.latencies[key]; ok {
			s.P50Latency = lt.Percentile(0.50) * 1000
			s.P90Latency = lt.Percentile(0.90) * 1000
			s.P99Latency = lt.Percentile(0.99) * 1000
		}
		stats = append(stats, s)
	}
	return stats
}

func (m *Metrics) JSONHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		m.mu.RLock()
		defer m.mu.RUnlock()

		stats := map[string]interface{}{
			"requests_total":     counterMapToJSON(m.requestsTotal),
			"errors_total":       counterMapToJSON(m.errorsTotal),
			"rate_limit_hits":    counterMapToJSON(m.rateLimitHits),
			"api_key_requests":   counterMapToJSON(m.apiKeyRequests),
			"upstream_health":    counterMapToJSON(m.upstreamHealth),
			"requests_in_flight": counterMapToJSON(m.requestsInFlight),
		}
		_ = json.NewEncoder(w).Encode(stats)
	})
}

func counterMapToJSON(m map[string]*atomic.Int64) map[string]int64 {
	result := make(map[string]int64)
	for k, v := range m {
		result[k] = v.Load()
	}
	return result
}
