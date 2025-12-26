# Health Checks

Relaypoint continuously monitors backend health to ensure requests are only routed to healthy servers.

## Overview

Health checks provide:

- **Automatic failure detection** - Unhealthy backends are removed from rotation
- **Automatic recovery** - Recovered backends are added back
- **Zero downtime** - Traffic shifts seamlessly between backends
- **Visibility** - Health status exposed via metrics

## Basic Configuration

Enable health checks for an upstream:

```yaml
upstreams:
  - name: api-service
    targets:
      - url: http://backend-1:3000
      - url: http://backend-2:3000
      - url: http://backend-3:3000
    load_balance: round_robin
    health_check:
      path: /health # Required: Health check endpoint
      interval: 10s # Check every 10 seconds
      timeout: 2s # Timeout for health check request
```

## Configuration Options

| Option     | Type     | Default  | Description                      |
| ---------- | -------- | -------- | -------------------------------- |
| `path`     | string   | Required | Health check endpoint path       |
| `interval` | duration | `10s`    | Time between health checks       |
| `timeout`  | duration | `2s`     | Request timeout for health check |

## How Health Checks Work

```
┌─────────────────────────────────────────────────────┐
│                    Relaypoint                        │
│                                                     │
│   Health Checker (runs every interval)              │
│   ┌─────────────────────────────────────────────┐   │
│   │  For each upstream:                         │   │
│   │    For each target:                         │   │
│   │      1. Send GET request to health path     │   │
│   │      2. Wait for response (up to timeout)   │   │
│   │      3. Check status code (2xx/3xx = OK)    │   │
│   │      4. Update target health status         │   │
│   └─────────────────────────────────────────────┘   │
│                                                     │
│   Load Balancer                                     │
│   ┌─────────────────────────────────────────────┐   │
│   │  Only routes to targets where:              │   │
│   │    healthy = true                           │   │
│   └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

### Success Criteria

A health check is considered **successful** if:

1. Connection is established within timeout
2. Response is received within timeout
3. HTTP status code is 2xx or 3xx

### Failure Handling

When a backend fails:

```
Time 0:00  - Health check fails
Time 0:00  - Backend marked unhealthy
Time 0:00  - Traffic stops routing to this backend
Time 0:10  - Next health check runs
Time 0:10  - If successful, backend marked healthy
Time 0:10  - Traffic resumes to this backend
```

## Health Check Endpoint Requirements

Your backend services should implement a health endpoint that:

### Minimal Implementation

```go
// Go example
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"healthy"}`))
})
```

```python
# Python Flask example
@app.route('/health')
def health():
    return jsonify({"status": "healthy"}), 200
```

```javascript
// Node.js Express example
app.get("/health", (req, res) => {
  res.json({ status: "healthy" });
});
```

### Comprehensive Implementation

A robust health check should verify:

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Check database connection
    if err := db.Ping(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "unhealthy",
            "error":  "database unavailable",
        })
        return
    }

    // Check cache connection
    if err := cache.Ping(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "unhealthy",
            "error":  "cache unavailable",
        })
        return
    }

    // All checks passed
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
    })
}
```

## Multiple Upstreams

Configure different health checks per upstream:

```yaml
upstreams:
  # API service - standard health check
  - name: api-service
    targets:
      - url: http://api-1:3000
      - url: http://api-2:3000
    health_check:
      path: /health
      interval: 10s
      timeout: 2s

  # Database proxy - quick checks
  - name: db-proxy
    targets:
      - url: http://db-proxy-1:5432
      - url: http://db-proxy-2:5432
    health_check:
      path: /ping
      interval: 5s
      timeout: 1s

  # Slow service - longer timeout
  - name: report-service
    targets:
      - url: http://reports:3000
    health_check:
      path: /healthz
      interval: 30s
      timeout: 10s
```

## Monitoring Health Status

### Metrics

Health status is exposed via Prometheus metrics:

```bash
curl http://localhost:9090/metrics | grep upstream_healthy
```

Output:

```
# HELP gateway_upstream_healthy Whether upstream is healthy
# TYPE gateway_upstream_healthy gauge
gateway_upstream_healthy{key="api-service_http://backend-1:3000"} 1
gateway_upstream_healthy{key="api-service_http://backend-2:3000"} 1
gateway_upstream_healthy{key="api-service_http://backend-3:3000"} 0
```

Values:

- `1` = Healthy
- `0` = Unhealthy

### Alerting on Unhealthy Backends

Prometheus alerting rule:

```yaml
groups:
  - name: relaypoint
    rules:
      - alert: BackendUnhealthy
        expr: gateway_upstream_healthy == 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Backend {{ $labels.key }} is unhealthy"

      - alert: AllBackendsUnhealthy
        expr: sum by (upstream) (gateway_upstream_healthy) == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "All backends for {{ $labels.upstream }} are unhealthy"
```

### Grafana Dashboard

Query examples:

```promql
# Healthy backend count per upstream
sum by (upstream) (gateway_upstream_healthy)

# Unhealthy backend count per upstream
count by (upstream) (gateway_upstream_healthy == 0)

# Health check success rate (requires additional metrics)
rate(gateway_health_checks_success_total[5m]) /
rate(gateway_health_checks_total[5m])
```

## Common Health Check Patterns

### Kubernetes-Style Probes

```yaml
upstreams:
  - name: k8s-service
    targets:
      - url: http://service:3000
    health_check:
      path: /healthz # Kubernetes convention
      interval: 10s
      timeout: 2s
```

### Liveness vs Readiness

If your service distinguishes between liveness and readiness:

```yaml
upstreams:
  # Use readiness for load balancing decisions
  - name: api-service
    targets:
      - url: http://api:3000
    health_check:
      path: /ready # Readiness endpoint
      interval: 5s
      timeout: 2s
```

### Deep Health Checks

For critical services, verify dependencies:

```yaml
upstreams:
  - name: critical-service
    targets:
      - url: http://critical:3000
    health_check:
      path: /health/deep # Checks all dependencies
      interval: 30s # Longer interval (expensive check)
      timeout: 10s # Longer timeout (multiple checks)
```

## Failure Scenarios

### Single Backend Failure

```
Before:
  Backend 1: ✓ Healthy  ←─┐
  Backend 2: ✓ Healthy  ←─┼── Round Robin
  Backend 3: ✓ Healthy  ←─┘

After Backend 2 fails:
  Backend 1: ✓ Healthy  ←─┐
  Backend 2: ✗ Unhealthy   │   Round Robin
  Backend 3: ✓ Healthy  ←─┘
```

Traffic automatically routes only to healthy backends.

### All Backends Fail

When all backends are unhealthy, Relaypoint:

1. Logs a warning
2. Returns `503 Service Unavailable` to clients
3. Continues health checking
4. Resumes traffic when any backend recovers

### Backend Recovery

```
Time 0:00  - Backend 2 recovers
Time 0:05  - Health check runs (interval = 5s)
Time 0:05  - Health check succeeds
Time 0:05  - Backend 2 marked healthy
Time 0:05  - Traffic resumes to Backend 2
```

## Best Practices

### 1. Keep Health Checks Lightweight

```go
// Good: Fast, simple check
func health(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}

// Bad: Slow, complex check on health endpoint
func health(w http.ResponseWriter, r *http.Request) {
    result := runExpensiveQuery()  // Don't do this
    // ...
}
```

### 2. Use Appropriate Intervals

| Service Type          | Recommended Interval |
| --------------------- | -------------------- |
| High-availability API | 5s                   |
| Standard API          | 10s                  |
| Background service    | 30s                  |
| Batch processing      | 60s                  |

### 3. Set Reasonable Timeouts

```yaml
# Good: Timeout < Interval
health_check:
  interval: 10s
  timeout: 2s

# Bad: Timeout >= Interval
health_check:
  interval: 10s
  timeout: 15s   # Health checks will overlap!
```

### 4. Match Health Check to Service Capabilities

```yaml
# Fast service - quick checks
- name: cache
  health_check:
    path: /ping
    interval: 5s
    timeout: 500ms

# Slow service - longer checks
- name: analytics
  health_check:
    path: /health
    interval: 60s
    timeout: 10s
```

### 5. Use Dedicated Health Endpoints

Don't use regular API endpoints for health checks:

```yaml
# Good: Dedicated health endpoint
health_check:
  path: /health

# Bad: Using regular endpoint
health_check:
  path: /api/users   # May be slow, rate-limited, auth-required
```

## Troubleshooting

### Backend Marked Unhealthy Incorrectly

**Symptoms**: Working backend shows as unhealthy.

**Causes**:

1. Health endpoint returns non-2xx status
2. Health endpoint too slow (timeout)
3. Network connectivity issues
4. Health endpoint requires authentication

**Solutions**:

1. Test health endpoint manually: `curl -v http://backend:3000/health`
2. Increase timeout if endpoint is slow
3. Check network/firewall between Relaypoint and backend
4. Ensure health endpoint doesn't require auth

### Health Checks Overloading Backend

**Symptoms**: High CPU/memory from health checks.

**Causes**:

1. Interval too short
2. Health check doing expensive operations

**Solutions**:

1. Increase interval (e.g., 5s → 30s)
2. Optimize health check implementation
3. Use simple ping endpoint instead of full health check

### Flapping Health Status

**Symptoms**: Backend alternates between healthy/unhealthy.

**Causes**:

1. Intermittent network issues
2. Backend at capacity
3. Timeout too aggressive

**Solutions**:

1. Investigate network stability
2. Scale backend or reduce traffic
3. Increase timeout

## Gateway Health Endpoint

Relaypoint exposes its own health endpoint:

```bash
curl http://localhost:8080/health
```

Response:

```json
{ "status": "healthy" }
```

This endpoint confirms Relaypoint itself is running, independent of backend health.

## Next Steps

- [Load Balancing](./load-balancing.md) - How health affects load balancing
- [Metrics](./metrics.md) - Monitor health status
- [Troubleshooting](./troubleshooting.md) - Debug health issues
