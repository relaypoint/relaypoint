package health

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/relaypoint/relaypoint/internal/config"
	"github.com/relaypoint/relaypoint/internal/loadbalancer"
	"github.com/relaypoint/relaypoint/internal/metrics"
)

type Checker struct {
	upstreams map[string]loadbalancer.LoadBalancer
	configs   map[string]*config.HealthCheck
	metrics   *metrics.Metrics
	client    *http.Client
	stop      chan struct{}
	wg        sync.WaitGroup
	logger    *slog.Logger
}

func NewChecker(upstreams map[string]loadbalancer.LoadBalancer, configs map[string]*config.HealthCheck, m *metrics.Metrics, logger *slog.Logger) *Checker {
	return &Checker{
		upstreams: upstreams,
		configs:   configs,
		metrics:   m,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		stop:   make(chan struct{}),
		logger: logger,
	}
}

func (c *Checker) Start() {
	for name, lb := range c.upstreams {
		cfg := c.configs[name]
		if cfg == nil {
			continue
		}

		c.wg.Add(1)
		go c.checkLoop(name, lb, cfg)
	}
}

func (c *Checker) Stop() {
	close(c.stop)
	c.wg.Wait()
}

func (c *Checker) checkLoop(name string, lb loadbalancer.LoadBalancer, cfg *config.HealthCheck) {
	defer c.wg.Done()

	interval := cfg.Interval
	if interval <= 0 {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.checkAll(name, lb, cfg) // Initial check

	for {
		select {
		case <-ticker.C:
			c.checkAll(name, lb, cfg)
		case <-c.stop:
			return
		}
	}
}

func (c *Checker) checkAll(name string, lb loadbalancer.LoadBalancer, cfg *config.HealthCheck) {
	targets := lb.Targets()

	for _, target := range targets {
		healthy := c.checkTarget(target, cfg)
		lb.MarkHealthy(target, healthy)

		if c.metrics != nil {
			c.metrics.RecordUpstreamHealth(name, target.URL.String(), healthy)
		}

		if !healthy {
			c.logger.Warn("upstream unhealthy", "upstream", name, "target", target.URL.String())
		}
	}
}

func (c *Checker) checkTarget(target *loadbalancer.Target, cfg *config.HealthCheck) bool {
	url := target.URL.ResolveReference(&url.URL{Path: cfg.Path})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
