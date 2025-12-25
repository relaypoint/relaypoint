package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            8080,
			Host:            "0.0.0.0",
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		RateLimit: RateLimitConfig{
			Enabled:         true,
			DefaultRPS:      100,
			DefaultBurst:    200,
			PerIP:           true,
			PerAPIKey:       true,
			CleanupInterval: 5 * time.Minute,
		},
		Metrics: MetricsConfig{
			Enabled:        true,
			Port:           9090,
			Path:           "/metrics",
			LatencyBuckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if len(c.Routes) == 0 {
		return fmt.Errorf("at least one route must be defined")
	}

	upstreamMap := make(map[string]bool)
	for _, u := range c.Upstreams {
		if u.Name == "" {
			return fmt.Errorf("upstream name cannot be empty")
		}
		if len(u.Targets) == 0 {
			return fmt.Errorf("upstream %s must have at least one target", u.Name)
		}
		upstreamMap[u.Name] = true
	}

	for _, r := range c.Routes {
		if r.Path == "" {
			return fmt.Errorf("route path cannot be empty")
		}
		if r.Upstream == "" {
			return fmt.Errorf("route %s must specify an upstream", r.Name)
		}
		if !upstreamMap[r.Upstream] {
			return fmt.Errorf("route %s references unknown upstream %s", r.Name, r.Upstream)
		}
	}

	return nil
}
