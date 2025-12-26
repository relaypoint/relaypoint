# Metrics & Monitoring

Relaypoint provides comprehensive observability through Prometheus metrics and a JSON stats endpoint.

## Overview

Relaypoint exposes:

- **Prometheus metrics** at `/metrics` (default port 9090)
- **JSON stats** at `/stats` (on the main gateway port)
- **Health endpoint** at `/health` (on the main gateway port)

## Enabling Metrics

```yaml
metrics:
  enabled: true # Enable metrics (default: true)
  port: 9090 # Metrics server port (default: 9090)
  path: "/metrics" # Metrics endpoint path (default: /metrics)
  latency_buckets: # Optional: Custom histogram buckets
    - 0.005
    - 0.01
    - 0.025
    - 0.05
    - 0.1
    - 0.25
    - 0.5
    - 1.0
    - 2.5
    - 5.0
    - 10.0
```

## Available Metrics

### Request Metrics

#### `gateway_requests_total`

Total number of requests processed.

| Label | Description                              |
| ----- | ---------------------------------------- |
| `key` | Format: `{route}_{method}_{status_code}` |

```promql
# Total requests
sum(gateway_requests_total)

# Requests by route
sum by (key) (gateway_requests_total)

# Request rate (requests/second)
rate(gateway_requests_total[5m])

# Requests by status code
sum by (status) (gateway_requests_total{key=~".*_200"})
```

#### `gateway_request_duration_seconds`

Request latency histogram.

| Label | Description                |
| ----- | -------------------------- |
| `key` | Format: `{route}_{method}` |

```promql
# Average latency
rate(gateway_request_duration_seconds_sum[5m]) /
rate(gateway_request_duration_seconds_count[5m])

# 95th percentile latency
histogram_quantile(0.95,
  rate(gateway_request_duration_seconds_bucket[5m]))

# 99th percentile latency
histogram_quantile(0.99,
  rate(gateway_request_duration_seconds_bucket[5m]))
```

#### `gateway_requests_in_flight`

Current number of requests being processed.

| Label | Description           |
| ----- | --------------------- |
| `key` | Route name or pattern |

```promql
# Current in-flight requests
gateway_requests_in_flight

# Total in-flight requests
sum(gateway_requests_in_flight)
```

### Error Metrics

#### `gateway_errors_total`

Total number of errors.

| Label | Description                    |
| ----- | ------------------------------ |
| `key` | Format: `{route}_{error_type}` |

Error types:

- `not_found` - Route not matched
- `upstream_not_found` - Upstream not configured
- `no_healthy_upstream` - All backends unhealthy
- `proxy_error` - Error proxying to backend

```promql
# Total errors
sum(gateway_errors_total)

# Error rate
rate(gateway_errors_total[5m])

# Errors by type
sum by (key) (gateway_errors_total)
```

### Rate Limiting Metrics

#### `gateway_rate_limit_hits_total`

Total number of rate-limited requests.

| Label | Description                    |
| ----- | ------------------------------ |
| `key` | Format: `{route}_{limit_type}` |

Limit types:

- `route` - Route-level rate limit
- `apikey` - API key rate limit
- `ip` - Per-IP rate limit

```promql
# Total rate limit hits
sum(gateway_rate_limit_hits_total)

# Rate limit hits per minute
rate(gateway_rate_limit_hits_total[1m]) * 60

# Rate limits by type
sum by (key) (gateway_rate_limit_hits_total)
```

### API Key Metrics

#### `gateway_api_key_requests_total`

Requests per API key.

| Label | Description                            |
| ----- | -------------------------------------- |
| `key` | Format: `{api_key_name}_{status_code}` |

```promql
# Requests by API key
sum by (key) (gateway_api_key_requests_total)

# Top API keys by request volume
topk(10, sum by (key) (rate(gateway_api_key_requests_total[5m])))
```

### Upstream Health Metrics

#### `gateway_upstream_healthy`

Backend health status gauge.

| Label | Description                       |
| ----- | --------------------------------- |
| `key` | Format: `{upstream}_{target_url}` |

Values:

- `1` = Healthy
- `0` = Unhealthy

```promql
# All backend health status
gateway_upstream_healthy

# Unhealthy backends
gateway_upstream_healthy == 0

# Healthy backend count per upstream
sum by (upstream) (gateway_upstream_healthy)
```

## JSON Stats Endpoint

The `/stats` endpoint provides real-time statistics in JSON format:

```bash
curl http://localhost:8080/stats
```

Response:

```json
[
  {
    "key": "users-api",
    "request_count": 15234,
    "error_count": 12,
    "p50_latency_ms": 5.2,
    "p90_latency_ms": 12.8,
    "p99_latency_ms": 45.3
  },
  {
    "key": "orders-api",
    "request_count": 8432,
    "error_count": 3,
    "p50_latency_ms": 8.1,
    "p90_latency_ms": 22.4,
    "p99_latency_ms": 89.7
  }
]
```

## Prometheus Integration

### Scrape Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: "relaypoint"
    static_configs:
      - targets: ["relaypoint:9090"]
    scrape_interval: 15s
    metrics_path: /metrics
```

### Kubernetes ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: relaypoint
  labels:
    app: relaypoint
spec:
  selector:
    matchLabels:
      app: relaypoint
  endpoints:
    - port: metrics
      interval: 15s
      path: /metrics
```

## Grafana Dashboards

### Overview Dashboard

Create panels for:

1. **Request Rate**

```promql
sum(rate(gateway_requests_total[5m]))
```

2. **Error Rate**

```promql
sum(rate(gateway_errors_total[5m])) /
sum(rate(gateway_requests_total[5m])) * 100
```

3. **Latency (p50, p90, p99)**

```promql
histogram_quantile(0.50, sum(rate(gateway_request_duration_seconds_bucket[5m])) by (le))
histogram_quantile(0.90, sum(rate(gateway_request_duration_seconds_bucket[5m])) by (le))
histogram_quantile(0.99, sum(rate(gateway_request_duration_seconds_bucket[5m])) by (le))
```

4. **Requests In Flight**

```promql
sum(gateway_requests_in_flight)
```

5. **Backend Health**

```promql
sum(gateway_upstream_healthy)
```

6. **Rate Limit Hits**

```promql
sum(rate(gateway_rate_limit_hits_total[5m]))
```

### Sample Dashboard JSON

```json
{
  "title": "Relaypoint Gateway",
  "panels": [
    {
      "title": "Request Rate",
      "type": "graph",
      "targets": [
        {
          "expr": "sum(rate(gateway_requests_total[5m]))",
          "legendFormat": "Requests/sec"
        }
      ]
    },
    {
      "title": "Latency Percentiles",
      "type": "graph",
      "targets": [
        {
          "expr": "histogram_quantile(0.50, sum(rate(gateway_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "p50"
        },
        {
          "expr": "histogram_quantile(0.90, sum(rate(gateway_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "p90"
        },
        {
          "expr": "histogram_quantile(0.99, sum(rate(gateway_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "p99"
        }
      ]
    },
    {
      "title": "Backend Health",
      "type": "stat",
      "targets": [
        {
          "expr": "sum(gateway_upstream_healthy)",
          "legendFormat": "Healthy Backends"
        }
      ]
    }
  ]
}
```

## Alerting Rules

### Prometheus Alerting Rules

```yaml
# alerts.yml
groups:
  - name: relaypoint
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: |
          sum(rate(gateway_errors_total[5m])) / 
          sum(rate(gateway_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate (> 5%)"
          description: "Error rate is {{ $value | humanizePercentage }}"

      # Very high error rate
      - alert: CriticalErrorRate
        expr: |
          sum(rate(gateway_errors_total[5m])) / 
          sum(rate(gateway_requests_total[5m])) > 0.25
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Critical error rate (> 25%)"

      # High latency
      - alert: HighLatency
        expr: |
          histogram_quantile(0.95, 
            sum(rate(gateway_request_duration_seconds_bucket[5m])) by (le)
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High p95 latency (> 1s)"
          description: "p95 latency is {{ $value | humanizeDuration }}"

      # Backend unhealthy
      - alert: BackendUnhealthy
        expr: gateway_upstream_healthy == 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Backend is unhealthy"
          description: "Backend {{ $labels.key }} is down"

      # All backends unhealthy
      - alert: AllBackendsDown
        expr: sum by (upstream) (gateway_upstream_healthy) == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "All backends for {{ $labels.upstream }} are down"

      # High rate limiting
      - alert: HighRateLimiting
        expr: |
          sum(rate(gateway_rate_limit_hits_total[5m])) / 
          sum(rate(gateway_requests_total[5m])) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High rate limiting (> 10% of requests)"

      # No requests (service might be down)
      - alert: NoRequests
        expr: sum(rate(gateway_requests_total[5m])) == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "No requests received in 5 minutes"
```

## Logging

Relaypoint outputs structured JSON logs:

```json
{"level":"INFO","msg":"Starting RelayPoint","config":"relaypoint.yml"}
{"level":"INFO","msg":"configuration loaded","routes":5,"upstreams":3}
{"level":"INFO","msg":"relaypoint API Gateway starting","address":"0.0.0.0:8080"}
{"level":"WARN","msg":"upstream unhealthy","upstream":"api-service","target":"http://backend-2:3000"}
```

### Log Levels

| Level   | Description                                   |
| ------- | --------------------------------------------- |
| `INFO`  | Normal operational messages                   |
| `WARN`  | Warning conditions (e.g., unhealthy backend)  |
| `ERROR` | Error conditions (e.g., configuration errors) |

### Log Aggregation

Forward logs to your preferred system:

```bash
# Send to file
./relaypoint -config relaypoint.yml 2>&1 | tee /var/log/relaypoint.log

# Send to journald (systemd)
./relaypoint -config relaypoint.yml

# Parse with jq
./relaypoint -config relaypoint.yml 2>&1 | jq -r '.msg'
```

## Health Endpoint

Check gateway health:

```bash
curl http://localhost:8080/health
```

Response:

```json
{ "status": "healthy" }
```

Use this for:

- Load balancer health checks
- Kubernetes liveness probes
- Monitoring systems

## Example Monitoring Stack

### Docker Compose Setup

```yaml
version: "3.8"

services:
  relaypoint:
    image: ghcr.io/relaypoint/relaypoint:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - ./relaypoint.yml:/etc/relaypoint/relaypoint.yml
    command: ["-config", "/etc/relaypoint/relaypoint.yml"]

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./alerts.yml:/etc/prometheus/alerts.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana

volumes:
  grafana-data:
```

### prometheus.yml

```yaml
global:
  scrape_interval: 15s

rule_files:
  - /etc/prometheus/alerts.yml

scrape_configs:
  - job_name: "relaypoint"
    static_configs:
      - targets: ["relaypoint:9090"]

alerting:
  alertmanagers:
    - static_configs:
        - targets: ["alertmanager:9093"]
```

## Custom Latency Buckets

Configure histogram buckets for your latency profile:

```yaml
metrics:
  enabled: true
  latency_buckets:
    # For low-latency APIs (sub-100ms expected)
    - 0.001 # 1ms
    - 0.005 # 5ms
    - 0.01 # 10ms
    - 0.025 # 25ms
    - 0.05 # 50ms
    - 0.1 # 100ms
    - 0.25 # 250ms
    - 0.5 # 500ms
    - 1.0 # 1s
```

```yaml
metrics:
  enabled: true
  latency_buckets:
    # For slower APIs (seconds expected)
    - 0.1 # 100ms
    - 0.5 # 500ms
    - 1.0 # 1s
    - 2.5 # 2.5s
    - 5.0 # 5s
    - 10.0 # 10s
    - 30.0 # 30s
    - 60.0 # 60s
```

## Best Practices

### 1. Set Up Dashboards First

Before deploying to production, have dashboards ready for:

- Request rate and error rate
- Latency percentiles
- Backend health
- Rate limiting

### 2. Configure Alerts

Essential alerts:

- High error rate (> 5%)
- High latency (p95 > threshold)
- Backend unhealthy
- No traffic (might indicate gateway down)

### 3. Use Labels Wisely

The default labels are designed for cardinality control. Avoid adding high-cardinality labels (like user IDs) to metrics.

### 4. Monitor the Monitor

Ensure Prometheus can reach the metrics endpoint:

```bash
curl http://localhost:9090/metrics
```

### 5. Retention Planning

Consider metric retention based on:

- Dashboard time ranges needed
- Storage capacity
- Compliance requirements

## Next Steps

- [Health Checks](./health-checks.md) - Monitor backend health
- [Troubleshooting](../troubleshooting.md) - Debug issues using metrics
- [Best Practices](../best-practises.md) - Production recommendations
