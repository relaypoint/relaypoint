package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/relaypoint/relaypoint/internal/config"
	"github.com/relaypoint/relaypoint/internal/health"
	"github.com/relaypoint/relaypoint/internal/loadbalancer"
	"github.com/relaypoint/relaypoint/internal/proxy"
)

func main() {
	configPath := flag.String("config", "relaypoint.yml", "Path to the configuration file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting RelayPoint", "config", *configPath)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger.Info("configuration loaded", "routes", len(cfg.Routes), "upstreams", len(cfg.Upstreams), "rate_limiting", cfg.RateLimit.Enabled)

	p, err := proxy.New(cfg)
	if err != nil {
		logger.Error("Failed to create proxy", "error", err)
		os.Exit(1)
	}
	defer p.Stop()

	upstreams := make(map[string]loadbalancer.LoadBalancer)
	healthConfigs := make(map[string]*config.HealthCheck)
	for _, u := range cfg.Upstreams {
		if u.HealthCheck != nil {
			healthConfigs[u.Name] = u.HealthCheck
		}
	}

	if len(healthConfigs) > 0 {
		// Get upstreams from proxy - we need to expose this
		// For now, skip health checker setup
		logger.Info("Health checks configured", "upstreams", len(healthConfigs))
	}

	mux := http.NewServeMux()
	mux.Handle("/", p)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := p.UsageStats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	var metricsServer *http.Server
	if cfg.Metrics.Enabled {
		metricsMux := http.NewServeMux()
		metricsMux.Handle(cfg.Metrics.Path, p.Metrics().Handler())
		metricsServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Metrics.Port),
			Handler: metricsMux,
		}
		go func() {
			logger.Info("metrics server starting", "port", cfg.Metrics.Port, "path", cfg.Metrics.Path)
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("metrics server error", "error", err)
			}
		}()
	}

	var checker *health.Checker
	_ = checker // suppress unused variable for now
	_ = upstreams

	go func() {
		logger.Info("relaypoint API Gateway starting", "address", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if metricsServer != nil {
		metricsServer.Shutdown(ctx)
	}

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	logger.Info("server gracefully stopped")

}
