# Rate Limiting

Relaypoint provides flexible rate limiting to protect your APIs from abuse and ensure fair usage across clients.

## Overview

Rate limiting controls how many requests clients can make within a given time period. Relaypoint uses a **token bucket algorithm** that provides:

- Smooth request throttling
- Burst capacity for legitimate traffic spikes
- Per-client, per-route, and per-API-key limits

## How Token Bucket Works

```
┌─────────────────────────────────────┐
│         Token Bucket                │
│                                     │
│  Capacity: 100 tokens (burst)       │
│  Refill Rate: 50 tokens/second      │
│                                     │
│  ████████████████░░░░░░░░░░░░░░░░░  │
│  Current: 75 tokens                 │
│                                     │
│  Each request consumes 1 token      │
│  Tokens refill continuously         │
└─────────────────────────────────────┘
```

- **Tokens** represent available request capacity
- **Requests consume** tokens (1 token per request)
- **Tokens refill** at a steady rate
- **Burst size** is the maximum tokens (bucket capacity)
- When empty, requests are rejected with `429 Too Many Requests`

## Basic Configuration

Enable rate limiting globally:

```yaml
rate_limit:
  enabled: true
  default_rps: 100 # 100 requests per second
  default_burst: 200 # Allow bursts up to 200 requests
  per_ip: true # Rate limit per client IP
  per_api_key: true # Rate limit per API key
  cleanup_interval: 5m # Clean up inactive limiters
```

## Rate Limiting Layers

Relaypoint applies rate limits at multiple levels:

```
┌──────────────────────────────────────────────────┐
│                    Request                        │
└──────────────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────┐
│            1. Route Rate Limit                    │
│         (if configured for route)                 │
└──────────────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────┐
│           2. API Key Rate Limit                   │
│          (if API key provided)                    │
└──────────────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────┐
│             3. IP Rate Limit                      │
│            (default per-IP)                       │
└──────────────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────┐
│               Backend Service                     │
└──────────────────────────────────────────────────┘
```

If **any** layer rejects the request, a `429` response is returned.

## Per-Route Rate Limiting

Apply specific limits to individual routes:

```yaml
routes:
  # High-volume endpoint - generous limits
  - name: read-api
    path: /api/v1/data/**
    upstream: data-service
    methods:
      - GET
    rate_limit:
      enabled: true
      requests_per_second: 1000
      burst_size: 2000

  # Expensive operation - strict limits
  - name: process-api
    path: /api/v1/process
    upstream: processor
    methods:
      - POST
    rate_limit:
      enabled: true
      requests_per_second: 10
      burst_size: 20

  # Authentication - prevent brute force
  - name: auth-api
    path: /api/v1/auth/**
    upstream: auth-service
    rate_limit:
      enabled: true
      requests_per_second: 5
      burst_size: 10
```

## Per-IP Rate Limiting

Limit requests from individual IP addresses:

```yaml
rate_limit:
  enabled: true
  default_rps: 100
  default_burst: 200
  per_ip: true
```

### IP Detection

Relaypoint detects client IP in this order:

1. `X-Forwarded-For` header (first IP in chain)
2. `X-Real-IP` header
3. Direct connection IP

### Handling Proxies

If Relaypoint is behind a load balancer or proxy, ensure proper headers are forwarded:

```nginx
# Nginx example
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Real-IP $remote_addr;
```

## API Key Rate Limiting

Different API keys can have different limits:

```yaml
rate_limit:
  enabled: true
  per_api_key: true

api_keys:
  # Premium tier - high limits
  - key: "pk_live_premium_abc123"
    name: "premium-customer"
    requests_per_second: 10000
    burst_size: 20000
    enabled: true

  # Standard tier - moderate limits
  - key: "pk_live_standard_def456"
    name: "standard-customer"
    requests_per_second: 1000
    burst_size: 2000
    enabled: true

  # Free tier - strict limits
  - key: "pk_live_free_ghi789"
    name: "free-user"
    requests_per_second: 100
    burst_size: 150
    enabled: true

  # Disabled key
  - key: "pk_live_disabled_xyz"
    name: "suspended-account"
    enabled: false
```

### API Key Detection

Relaypoint looks for API keys in:

1. `Authorization: Bearer <key>` header
2. `Authorization: ApiKey <key>` header
3. `X-API-Key: <key>` header
4. `?api_key=<key>` query parameter

Example requests:

```bash
# Authorization header (preferred)
curl -H "Authorization: Bearer pk_live_abc123" http://localhost:8080/api

# X-API-Key header
curl -H "X-API-Key: pk_live_abc123" http://localhost:8080/api

# Query parameter (less secure)
curl "http://localhost:8080/api?api_key=pk_live_abc123"
```

## Rate Limit Response

When rate limited, clients receive:

```http
HTTP/1.1 429 Too Many Requests
Content-Type: text/plain
Retry-After: 1

Too Many Requests
```

The `Retry-After` header indicates when to retry (in seconds).

## Choosing RPS and Burst Values

### Understanding the Relationship

```
RPS (Requests Per Second) = Sustained rate
Burst = Maximum spike capacity
```

**Example**: RPS=100, Burst=500

- Steady traffic: 100 requests/second ✓
- Spike of 500 requests: Allowed (consumes burst)
- After spike: Must wait for tokens to refill

### Guidelines

| Use Case           | RPS  | Burst | Notes                       |
| ------------------ | ---- | ----- | --------------------------- |
| Public API         | 60   | 120   | 1 request/second average    |
| Authenticated user | 100  | 200   | 2x burst for flexibility    |
| Internal service   | 1000 | 2000  | Higher for trusted services |
| Webhook receiver   | 500  | 1000  | Handle notification bursts  |
| Login endpoint     | 5    | 10    | Prevent brute force         |
| Search endpoint    | 30   | 60    | Expensive operation         |

### Burst Sizing Tips

- **Burst = 2x RPS**: Good default for most APIs
- **Burst = 10x RPS**: Handles traffic spikes from batch jobs
- **Burst = RPS**: Strict limiting, no spike allowance

## Configuration Examples

### Public API with Tiers

```yaml
rate_limit:
  enabled: true
  default_rps: 60 # Anonymous users
  default_burst: 120
  per_ip: true
  per_api_key: true

api_keys:
  # Tier 1: Free
  - key: "free_tier_key"
    name: "free"
    requests_per_second: 60
    burst_size: 120
    enabled: true

  # Tier 2: Pro
  - key: "pro_tier_key"
    name: "pro"
    requests_per_second: 600
    burst_size: 1200
    enabled: true

  # Tier 3: Enterprise
  - key: "enterprise_key"
    name: "enterprise"
    requests_per_second: 6000
    burst_size: 12000
    enabled: true

routes:
  # All routes use tier-based limits
  - name: api
    path: /api/**
    upstream: api-service
```

### Protecting Sensitive Endpoints

```yaml
routes:
  # Login - strict limit to prevent brute force
  - name: login
    path: /api/auth/login
    upstream: auth-service
    methods: [POST]
    rate_limit:
      enabled: true
      requests_per_second: 3
      burst_size: 5

  # Password reset - very strict
  - name: password-reset
    path: /api/auth/reset-password
    upstream: auth-service
    methods: [POST]
    rate_limit:
      enabled: true
      requests_per_second: 1
      burst_size: 3

  # Registration - moderate
  - name: register
    path: /api/auth/register
    upstream: auth-service
    methods: [POST]
    rate_limit:
      enabled: true
      requests_per_second: 5
      burst_size: 10
```

### Microservices with Different Limits

```yaml
routes:
  # Read-heavy service - high limits
  - name: catalog
    path: /catalog/**
    upstream: catalog-service
    rate_limit:
      enabled: true
      requests_per_second: 500
      burst_size: 1000

  # Write service - lower limits
  - name: orders
    path: /orders/**
    upstream: order-service
    rate_limit:
      enabled: true
      requests_per_second: 50
      burst_size: 100

  # Report generation - very low limits
  - name: reports
    path: /reports/**
    upstream: report-service
    rate_limit:
      enabled: true
      requests_per_second: 5
      burst_size: 10
```

## Monitoring Rate Limits

### Metrics

```bash
curl http://localhost:9090/metrics | grep rate_limit
```

Key metrics:

- `gateway_rate_limit_hits_total{route="...",type="..."}` - Count of rate-limited requests
- Types: `route`, `apikey`, `ip`

### Stats Endpoint

```bash
curl http://localhost:8080/stats
```

Shows request counts including rate-limited requests.

### Alerting

Set up alerts for high rate limit hits:

```yaml
# Prometheus alerting rule example
groups:
  - name: relaypoint
    rules:
      - alert: HighRateLimitHits
        expr: rate(gateway_rate_limit_hits_total[5m]) > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High rate limit hits detected"
```

## Best Practices

### 1. Start Generous, Then Tighten

```yaml
# Start with high limits
rate_limit:
  default_rps: 1000
  default_burst: 2000

# Monitor and adjust based on actual usage
rate_limit:
  default_rps: 200
  default_burst: 400
```

### 2. Document Your Limits

Provide clear documentation for API consumers:

```markdown
## Rate Limits

| Tier       | Requests/Second | Burst |
| ---------- | --------------- | ----- |
| Free       | 60              | 120   |
| Pro        | 600             | 1200  |
| Enterprise | 6000            | 12000 |

Rate-limited requests return `429 Too Many Requests` with a `Retry-After` header.
```

### 3. Different Limits for Different Operations

```yaml
routes:
  # Reads are cheap
  - path: /api/**
    methods: [GET]
    rate_limit:
      requests_per_second: 1000

  # Writes are expensive
  - path: /api/**
    methods: [POST, PUT, DELETE]
    rate_limit:
      requests_per_second: 100
```

### 4. Consider Time of Day

Use lower limits during known high-traffic periods to ensure service stability.

### 5. Exempt Internal Services

For service-to-service communication, use high limits:

```yaml
api_keys:
  - key: "internal_service_key"
    name: "internal-services"
    requests_per_second: 100000
    burst_size: 200000
    enabled: true
```

## Troubleshooting

### Legitimate Users Being Limited

**Symptoms**: Real users hitting rate limits unexpectedly.

**Solutions**:

1. Increase burst size for traffic spikes
2. Check if shared IP (NAT/proxy) is causing issues
3. Consider API key tiers instead of IP limiting

### Rate Limits Not Working

**Symptoms**: Requests exceeding limits not being blocked.

**Causes**:

1. Rate limiting not enabled
2. Route-specific limit not configured
3. API key has unlimited access

**Solutions**:

1. Verify `rate_limit.enabled: true` in config
2. Add explicit rate limit to route
3. Check API key configuration

### Memory Usage Growing

**Symptoms**: Memory increasing over time.

**Cause**: Rate limiter tracking many unique clients.

**Solution**: Ensure cleanup is configured:

```yaml
rate_limit:
  cleanup_interval: 5m # Clean up inactive limiters
```

## Next Steps

- [Health Checks](./health-checks.md) - Monitor backend health
- [Metrics](./metrics.md) - Monitor rate limiting
- [API Keys](./api-keys.md) - Manage API keys
