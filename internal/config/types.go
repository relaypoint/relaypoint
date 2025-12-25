package config

import "time"

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Upstreams []Upstream      `yaml:"upstreams"`
	Routes    []Route         `yaml:"routes"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Metrics   MetricsConfig   `yaml:"metrics"`
	APIKeys   []APIKey        `yaml:"api_keys"`
}

type ServerConfig struct {
	Port            int           `yaml:"port"`
	Host            string        `yaml:"host"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type Upstream struct {
	Name        string       `yaml:"name"`
	Targets     []Target     `yaml:"targets"`
	HealthCheck *HealthCheck `yaml:"health_check,omitempty"`
	LoadBalance string       `yaml:"load_balance"` // round_robin, least_conn, random
}

type Target struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"`
}

type HealthCheck struct {
	Path     string        `yaml:"path"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

type Route struct {
	Name       string            `yaml:"name"`
	Host       string            `yaml:"host"`
	Path       string            `yaml:"path"`
	Methods    []string          `yaml:"methods,omitempty"`
	Upstream   string            `yaml:"upstream"`
	StripPath  bool              `yaml:"strip_path"`
	Headers    map[string]string `yaml:"headers,omitempty"`
	RateLimit  *RouteRateLimit   `yaml:"rate_limit,omitempty"`
	Timeout    time.Duration     `yaml:"timeout,omitempty"`
	RetryCount int               `yaml:"retry_count,omitempty"`
}

type RouteRateLimit struct {
	RequestsPerSecond int  `yaml:"requests_per_second"`
	BurstSize         int  `yaml:"burst_size"`
	Enabled           bool `yaml:"enabled"`
}

type RateLimitConfig struct {
	Enabled         bool          `yaml:"enabled"`
	DefaultRPS      int           `yaml:"default_rps"`
	DefaultBurst    int           `yaml:"default_burst"`
	PerIP           bool          `yaml:"per_ip"`
	PerAPIKey       bool          `yaml:"per_api_key"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

type MetricsConfig struct {
	Enabled        bool      `yaml:"enabled"`
	Port           int       `yaml:"port"`
	Path           string    `yaml:"path"`
	LatencyBuckets []float64 `yaml:"latency_buckets,omitempty"`
}

type APIKey struct {
	Key               string `yaml:"key"`
	Name              string `yaml:"name"`
	RequestsPerSecond int    `yaml:"requests_per_second"`
	BurstSize         int    `yaml:"burst_size"`
	Enabled           bool   `yaml:"enabled"`
}
