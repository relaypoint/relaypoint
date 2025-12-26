# Configuration Reference

This document provides a complete reference for all Relaypoint configuration options.

## Configuration File

Relaypoint uses YAML for configuration. By default, it looks for `relaypoint.yml` in the current directory.

```bash
# Use default configuration file
./relaypoint

# Specify configuration file
./relaypoint -config /path/to/config.yml
```

## Complete Configuration Example

```yaml
# =============================================================================
# SERVER CONFIGURATION
# =============================================================================
server:
  port: 8080 # Port to listen on (default: 8080)
  host: "0.0.0.0" # Host to bind to (default: 0.0.0.0)
  read_timeout: 30s # Maximum duration for reading request (default: 30s)
  write_timeout: 30s # Maximum duration for writing response (default: 30s)
  shutdown_timeout: 10s # Graceful shutdown timeout (default: 10s)

# =============================================================================
# METRICS CONFIGURATION
# =============================================================================
metrics:
  enabled: true # Enable metrics endpoint (default: true)
  port: 9090 # Metrics server port (default: 9090)
  path: "/metrics" # Metrics endpoint path (default: /metrics)
  latency_buckets: # Histogram buckets for latency metrics (optional)
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

# =============================================================================
# RATE LIMITING CONFIGURATION
# =============================================================================
rate_limit:
  enabled: true # Enable rate limiting (default: true)
  default_rps: 100 # Default requests per second (default: 100)
  default_burst: 200 # Default burst size (default: 200)
  per_ip: true # Enable per-IP rate limiting (default: true)
  per_api_key: true # Enable per-API-key rate limiting (default: true)
  cleanup_interval: 5m # Interval to clean up stale limiters (default: 5m)

# =============================================================================
# UPSTREAMS (Backend Services)
# =============================================================================
upstreams:
  - name: user-service # Unique identifier for this upstream (required)
    targets: # List of backend servers (required, at least one)
      - url: http://localhost:3001 # Backend URL (required)
        weight: 2 # Weight for weighted load balancing (default: 1)
      - url: http://localhost:3002
        weight: 1
    load_balance:
      round_robin # Load balancing strategy (default: round_robin)
      # Options: round_robin, least_conn, random, weighted_round_robin
    health_check: # Health check configuration (optional)
      path: /health # Health check endpoint path (required if health_check defined)
      interval: 10s # Check interval (default: 10s)
      timeout: 2s # Request timeout (default: 2s)

  - name: order-service
    targets:
      - url: http://localhost:3003
    load_balance: least_conn
    health_check:
      path: /healthz
      interval: 5s
      timeout: 1s

# =============================================================================
# ROUTES
# =============================================================================
routes:
  - name: users-api # Route name for identification (optional but recommended)
    host: api.example.com # Host matching (optional, empty matches all hosts)
    path: /api/v1/users # URL path pattern (required)
    methods: # Allowed HTTP methods (optional, empty allows all)
      - GET
      - POST
    upstream: user-service # Target upstream name (required)
    strip_path: false # Remove matched path prefix before proxying (default: false)
    headers: # Headers to add to upstream request (optional)
      X-Custom-Header: "value"
      X-Service-Name: "users"
    rate_limit: # Route-specific rate limit (optional)
      enabled: true
      requests_per_second: 50
      burst_size: 100
    timeout: 30s # Request timeout for this route (optional)
    retry_count: 3 # Number of retries on failure (optional)

  - name: users-detail
    path: /api/v1/users/:id # Path with parameter
    upstream: user-service

  - name: orders-api
    path: /api/v1/orders/** # Wildcard matching
    upstream: order-service
    strip_path: true

# =============================================================================
# API KEYS
# =============================================================================
api_keys:
  - key: "pk_live_abc123def456" # The API key value (required)
    name: "production-web-app" # Human-readable name (required)
    requests_per_second: 1000 # Rate limit for this key (required)
    burst_size: 2000 # Burst size for this key (optional, defaults to rps * 2)
    enabled: true # Whether key is active (default: true)

  - key: "pk_test_xyz789"
    name: "development-app"
    requests_per_second: 100
    burst_size: 200
    enabled: true
```

## Configuration Sections

### Server

| Field              | Type     | Default     | Description                                         |
| ------------------ | -------- | ----------- | --------------------------------------------------- |
| `port`             | integer  | `8080`      | Port number for the gateway to listen on            |
| `host`             | string   | `"0.0.0.0"` | Host/IP address to bind to                          |
| `read_timeout`     | duration | `30s`       | Maximum time to read the entire request             |
| `write_timeout`    | duration | `30s`       | Maximum time to write the response                  |
| `shutdown_timeout` | duration | `10s`       | Time to wait for active connections during shutdown |

### Metrics

| Field             | Type      | Default      | Description                            |
| ----------------- | --------- | ------------ | -------------------------------------- |
| `enabled`         | boolean   | `true`       | Enable Prometheus metrics endpoint     |
| `port`            | integer   | `9090`       | Port for the metrics server            |
| `path`            | string    | `"/metrics"` | Path for the metrics endpoint          |
| `latency_buckets` | []float64 | See below    | Histogram bucket boundaries in seconds |

Default latency buckets: `[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]`

### Rate Limit

| Field              | Type     | Default | Description                                  |
| ------------------ | -------- | ------- | -------------------------------------------- |
| `enabled`          | boolean  | `true`  | Enable rate limiting globally                |
| `default_rps`      | integer  | `100`   | Default requests per second                  |
| `default_burst`    | integer  | `200`   | Default burst size (token bucket capacity)   |
| `per_ip`           | boolean  | `true`  | Enable rate limiting per client IP           |
| `per_api_key`      | boolean  | `true`  | Enable rate limiting per API key             |
| `cleanup_interval` | duration | `5m`    | How often to clean up inactive rate limiters |

### Upstreams

| Field          | Type        | Required | Description                                      |
| -------------- | ----------- | -------- | ------------------------------------------------ |
| `name`         | string      | Yes      | Unique identifier for the upstream               |
| `targets`      | []Target    | Yes      | List of backend server targets                   |
| `load_balance` | string      | No       | Load balancing strategy (default: `round_robin`) |
| `health_check` | HealthCheck | No       | Health check configuration                       |

#### Target

| Field    | Type    | Required | Description                                        |
| -------- | ------- | -------- | -------------------------------------------------- |
| `url`    | string  | Yes      | Backend server URL (e.g., `http://localhost:3000`) |
| `weight` | integer | No       | Weight for weighted load balancing (default: 1)    |

#### HealthCheck

| Field      | Type     | Required | Description                                  |
| ---------- | -------- | -------- | -------------------------------------------- |
| `path`     | string   | Yes      | Health check endpoint path (e.g., `/health`) |
| `interval` | duration | No       | Time between health checks (default: `10s`)  |
| `timeout`  | duration | No       | Health check request timeout (default: `2s`) |

### Routes

| Field         | Type           | Required | Description                                        |
| ------------- | -------------- | -------- | -------------------------------------------------- |
| `name`        | string         | No       | Human-readable route name (recommended)            |
| `host`        | string         | No       | Host to match (empty matches all hosts)            |
| `path`        | string         | Yes      | URL path pattern to match                          |
| `methods`     | []string       | No       | HTTP methods to match (empty allows all)           |
| `upstream`    | string         | Yes      | Name of the upstream to route to                   |
| `strip_path`  | boolean        | No       | Remove matched prefix from path (default: `false`) |
| `headers`     | map            | No       | Headers to add to upstream requests                |
| `rate_limit`  | RouteRateLimit | No       | Route-specific rate limiting                       |
| `timeout`     | duration       | No       | Request timeout for this route                     |
| `retry_count` | integer        | No       | Number of retry attempts on failure                |

#### RouteRateLimit

| Field                 | Type    | Required | Description                                            |
| --------------------- | ------- | -------- | ------------------------------------------------------ |
| `enabled`             | boolean | No       | Enable rate limiting for this route (default: `false`) |
| `requests_per_second` | integer | Yes      | Maximum requests per second                            |
| `burst_size`          | integer | No       | Burst capacity (default: `requests_per_second * 2`)    |

### API Keys

| Field                 | Type    | Required | Description                                         |
| --------------------- | ------- | -------- | --------------------------------------------------- |
| `key`                 | string  | Yes      | The API key value (keep secret!)                    |
| `name`                | string  | Yes      | Human-readable identifier                           |
| `requests_per_second` | integer | Yes      | Rate limit for this key                             |
| `burst_size`          | integer | No       | Burst capacity (default: `requests_per_second * 2`) |
| `enabled`             | boolean | No       | Whether key is active (default: `true`)             |

## Path Pattern Syntax

Relaypoint supports several path matching patterns:

| Pattern       | Description                  | Example Match             |
| ------------- | ---------------------------- | ------------------------- |
| `/exact`      | Exact match                  | `/exact` only             |
| `/prefix/*`   | Single segment wildcard      | `/prefix/anything`        |
| `/prefix/**`  | Multi-segment wildcard       | `/prefix/any/path/here`   |
| `/users/:id`  | Named parameter              | `/users/123` (id = "123") |
| `/users/{id}` | Named parameter (alt syntax) | `/users/456` (id = "456") |

### Path Matching Priority

Routes are matched in priority order:

1. Exact matches (highest priority)
2. Specific path segments
3. Named parameters
4. Single wildcards (`*`)
5. Multi-segment wildcards (`**`) (lowest priority)

## Duration Format

Durations can be specified as:

- `100ms` - 100 milliseconds
- `5s` - 5 seconds
- `2m` - 2 minutes
- `1h` - 1 hour
- `1h30m` - 1 hour and 30 minutes

## Environment Variables

While Relaypoint primarily uses YAML configuration, you can use environment variables in your configuration by processing the file with `envsubst`:

```yaml
server:
  port: ${RELAYPOINT_PORT:-8080}

upstreams:
  - name: backend
    targets:
      - url: ${BACKEND_URL}
```

```bash
export BACKEND_URL=http://backend:3000
envsubst < relaypoint.yml.template > relaypoint.yml
./relaypoint -config relaypoint.yml
```

## Configuration Validation

Relaypoint validates configuration on startup:

- Server port must be between 1 and 65535
- At least one route must be defined
- Each upstream must have a unique name
- Each upstream must have at least one target
- Each route must reference an existing upstream
- Upstream target URLs must be valid

If validation fails, Relaypoint will exit with an error message indicating the problem.

## Next Steps

- [Routing](./features/routing.md) - Advanced routing patterns
- [Load Balancing](./features/load-balancing.md) - Load balancing strategies
- [Rate Limiting](./features/rate-limiting.md) - Rate limiting configuration
