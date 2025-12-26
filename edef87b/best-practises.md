# Best Practices

This guide covers recommendations for running Relaypoint in production environments.

## Configuration Best Practices

### Use Descriptive Names

```yaml
# Good: Descriptive names
routes:
  - name: user-authentication
    path: /api/v1/auth/**
    upstream: auth-service

  - name: user-profile-management
    path: /api/v1/users/**
    upstream: user-service

# Bad: Generic or missing names
routes:
  - path: /api/v1/auth/**
    upstream: svc1

  - path: /api/v1/users/**
    upstream: svc2
```

### Order Routes by Specificity

```yaml
routes:
  # 1. Most specific routes first
  - name: user-by-id
    path: /api/users/:id
    upstream: user-service

  # 2. Then pattern matches
  - name: users
    path: /api/users/**
    upstream: user-service

  # 3. Catch-all last
  - name: default
    path: /**
    upstream: default-service
```

### Separate Concerns

```yaml
# Separate read and write operations
routes:
  - name: api-read
    path: /api/**
    methods: [GET, HEAD, OPTIONS]
    upstream: api-read-replicas
    rate_limit:
      requests_per_second: 1000

  - name: api-write
    path: /api/**
    methods: [POST, PUT, PATCH, DELETE]
    upstream: api-primary
    rate_limit:
      requests_per_second: 100
```

### Use Health Checks

```yaml
upstreams:
  - name: critical-service
    targets:
      - url: http://backend-1:3000
      - url: http://backend-2:3000
    health_check:
      path: /health # Always configure health checks
      interval: 10s
      timeout: 2s
```

## Security Best Practices

### Never Commit Secrets

```yaml
# Bad: Secrets in config
api_keys:
  - key: "pk_live_abc123secret"
    name: "production"

# Better: Use environment variables
# config.template.yml
api_keys:
  - key: "${API_KEY_PRODUCTION}"
    name: "production"
```

Process with envsubst:

```bash
envsubst < config.template.yml > config.yml
```

### Use Strong API Keys

```bash
# Generate secure keys
openssl rand -base64 32

# Or use a UUID
uuidgen
```

### Limit Burst Capacity

```yaml
rate_limit:
  enabled: true
  default_rps: 100
  default_burst: 200 # 2x RPS is reasonable
  # Avoid: default_burst: 10000  (too high)
```

### Protect Sensitive Endpoints

```yaml
routes:
  # Authentication - very strict
  - name: login
    path: /api/auth/login
    upstream: auth-service
    rate_limit:
      enabled: true
      requests_per_second: 5
      burst_size: 10

  # Admin endpoints - require API key and strict limits
  - name: admin
    path: /admin/**
    upstream: admin-service
    rate_limit:
      enabled: true
      requests_per_second: 10
      burst_size: 20
```

### Set Appropriate Timeouts

```yaml
server:
  read_timeout: 30s # Prevent slow loris attacks
  write_timeout: 30s # Limit response time
  shutdown_timeout: 10s # Graceful shutdown

routes:
  - name: quick-api
    path: /api/fast/**
    timeout: 5s # Fast endpoints should be fast

  - name: reports
    path: /api/reports/**
    timeout: 120s # Allow time for heavy operations
```

## High Availability

### Multiple Backend Instances

```yaml
upstreams:
  - name: api
    targets:
      - url: http://api-1:3000
      - url: http://api-2:3000
      - url: http://api-3:3000 # At least 3 for HA
    load_balance: round_robin
    health_check:
      path: /health
      interval: 5s
      timeout: 2s
```

### Use Appropriate Load Balancing

| Scenario                               | Strategy               |
| -------------------------------------- | ---------------------- |
| Identical servers, consistent requests | `round_robin`          |
| Variable request times                 | `least_conn`           |
| Different server capacities            | `weighted_round_robin` |

### Quick Health Checks

```yaml
health_check:
  path: /health
  interval: 5s # Frequent checks for quick detection
  timeout: 2s # Short timeout
```

### Multiple Relaypoint Instances

For true HA, run multiple Relaypoint instances behind a load balancer:

```
                    ┌──────────────────┐
                    │   Load Balancer  │
                    └────────┬─────────┘
                             │
           ┌─────────────────┼─────────────────┐
           │                 │                 │
    ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐
    │ Relaypoint  │   │ Relaypoint  │   │ Relaypoint  │
    │   Node 1    │   │   Node 2    │   │   Node 3    │
    └─────────────┘   └─────────────┘   └─────────────┘
```

## Performance Optimization

### Tune Burst for Your Traffic Pattern

```yaml
# Bursty traffic (mobile apps, batch jobs)
rate_limit:
  default_rps: 100
  default_burst: 1000    # 10x RPS for bursts

# Steady traffic (web apps, APIs)
rate_limit:
  default_rps: 100
  default_burst: 200     # 2x RPS
```

### Use Least Connections for Variable Workloads

```yaml
upstreams:
  - name: processing-service
    targets:
      - url: http://processor-1:3000
      - url: http://processor-2:3000
    load_balance: least_conn # Better for variable request times
```

### Optimize Health Check Intervals

```yaml
# Critical, fast services
health_check:
  interval: 5s
  timeout: 1s

# Less critical, slower services
health_check:
  interval: 30s
  timeout: 5s
```

### Right-Size Timeouts

```yaml
# Match timeout to expected response time
routes:
  - name: fast-api
    path: /api/lookup/**
    timeout: 2s # Fast lookups

  - name: standard-api
    path: /api/**
    timeout: 30s # Standard operations

  - name: reports
    path: /api/reports/**
    timeout: 300s # Long-running reports
```

## Monitoring Best Practices

### Set Up Dashboards

Essential panels:

1. Request rate (req/sec)
2. Error rate (%)
3. Latency percentiles (p50, p95, p99)
4. Backend health status
5. Rate limit hits

### Configure Alerts

```yaml
# Prometheus alerting rules
groups:
  - name: relaypoint
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: |
          sum(rate(gateway_errors_total[5m])) / 
          sum(rate(gateway_requests_total[5m])) > 0.01
        for: 5m
        labels:
          severity: warning

      # Backend down
      - alert: BackendDown
        expr: gateway_upstream_healthy == 0
        for: 1m
        labels:
          severity: critical

      # High latency
      - alert: HighLatency
        expr: |
          histogram_quantile(0.95, 
            rate(gateway_request_duration_seconds_bucket[5m])
          ) > 1
        for: 5m
        labels:
          severity: warning
```

### Log Aggregation

Forward logs to a central system:

```bash
# Send to file
./relaypoint -config config.yml 2>&1 >> /var/log/relaypoint/relaypoint.log

# With logrotate
/var/log/relaypoint/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
```

## Deployment Best Practices

### Use systemd for Linux

```ini
[Unit]
Description=Relaypoint API Gateway
After=network.target

[Service]
Type=simple
User=relaypoint
ExecStart=/usr/local/bin/relaypoint -config /etc/relaypoint/config.yml
Restart=always
RestartSec=5

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true

# Resource limits
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

### Graceful Shutdown

Configure appropriate shutdown timeout:

```yaml
server:
  shutdown_timeout: 30s # Allow in-flight requests to complete
```

### Configuration Validation

Validate before deploying:

```bash
# Test configuration
./relaypoint -config new-config.yml -validate

# Or start briefly
timeout 5 ./relaypoint -config new-config.yml
```

### Rolling Deployments

For zero-downtime updates:

1. Deploy new Relaypoint instance
2. Wait for health check pass
3. Add to load balancer
4. Remove old instance from load balancer
5. Shut down old instance

## Operational Runbooks

### Runbook: High Error Rate

1. Check error metrics: `curl localhost:9090/metrics | grep errors`
2. Identify error type (upstream_not_found, no_healthy_upstream, etc.)
3. Check backend health: `curl localhost:9090/metrics | grep upstream_healthy`
4. Test backend directly: `curl http://backend:3000/health`
5. Review recent configuration changes
6. Check backend logs

### Runbook: High Latency

1. Check latency metrics: `curl localhost:9090/metrics | grep duration`
2. Compare with backend directly: `time curl http://backend:3000/api/test`
3. Check in-flight requests: `curl localhost:9090/metrics | grep in_flight`
4. Review load balancing strategy
5. Check backend resource usage

### Runbook: Backend Unhealthy

1. Check which backends are unhealthy: `curl localhost:9090/metrics | grep upstream_healthy`
2. Test health endpoint: `curl http://backend:3000/health`
3. Check network connectivity: `nc -zv backend 3000`
4. Review backend logs
5. Verify health check configuration

## Capacity Planning

### Estimate Requirements

| Metric                 | Consideration                  |
| ---------------------- | ------------------------------ |
| Requests/second        | Each request uses minimal CPU  |
| Concurrent connections | Memory scales with connections |
| Unique rate limit keys | Memory for each unique IP/key  |

### Recommended Starting Resources

| Traffic        | CPU      | Memory |
| -------------- | -------- | ------ |
| < 1K req/s     | 1 core   | 128 MB |
| 1K-10K req/s   | 2 cores  | 256 MB |
| 10K-100K req/s | 4 cores  | 512 MB |
| > 100K req/s   | 8+ cores | 1+ GB  |

### Monitor Growth

Track over time:

- Request rate growth
- Unique IP count (rate limit buckets)
- Memory usage
- CPU usage

## Checklist: Production Readiness

Before going to production:

- [ ] **Configuration**

  - [ ] Descriptive route and upstream names
  - [ ] Routes ordered by specificity
  - [ ] Health checks configured for all upstreams
  - [ ] Appropriate timeouts set

- [ ] **Security**

  - [ ] No secrets in config files
  - [ ] Rate limiting enabled
  - [ ] Sensitive endpoints protected
  - [ ] API keys use secure values

- [ ] **High Availability**

  - [ ] Multiple backend instances
  - [ ] Health checks with quick detection
  - [ ] Multiple Relaypoint instances (if critical)

- [ ] **Monitoring**

  - [ ] Prometheus scraping metrics
  - [ ] Dashboards created
  - [ ] Alerts configured
  - [ ] Log aggregation set up

- [ ] **Operations**

  - [ ] systemd service configured
  - [ ] Graceful shutdown timeout set
  - [ ] Runbooks documented
  - [ ] Backup configuration stored

- [ ] **Testing**
  - [ ] Load tested at expected traffic
  - [ ] Failure scenarios tested
  - [ ] Configuration validated

## Next Steps

- [Examples](./examples.md) - Production-ready configurations
- [Troubleshooting](./troubleshooting.md) - Debug issues
- [Metrics](./features/metrics.md) - Set up monitoring
