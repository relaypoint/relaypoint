# Load Balancing

Relaypoint provides multiple load balancing strategies to distribute traffic across your backend servers efficiently.

## Overview

Load balancing ensures high availability and optimal performance by distributing requests across multiple backend instances. Relaypoint supports four load balancing strategies:

| Strategy               | Best For              | Description                                     |
| ---------------------- | --------------------- | ----------------------------------------------- |
| `round_robin`          | General use           | Equal distribution across all backends          |
| `least_conn`           | Variable workloads    | Routes to server with fewest active connections |
| `random`               | Simple distribution   | Random server selection                         |
| `weighted_round_robin` | Heterogeneous servers | Distribution based on server capacity           |

## Round Robin (Default)

Distributes requests evenly across all healthy backends in order.

```yaml
upstreams:
  - name: api-service
    targets:
      - url: http://backend-1:3000
      - url: http://backend-2:3000
      - url: http://backend-3:3000
    load_balance: round_robin
```

### How It Works

```
Request 1 → Backend 1
Request 2 → Backend 2
Request 3 → Backend 3
Request 4 → Backend 1
Request 5 → Backend 2
...
```

### When to Use

- All backend servers have similar capacity
- Request processing time is consistent
- Simple, predictable distribution is desired

### Advantages

- Simple and predictable
- Equal distribution of requests
- No additional tracking overhead

### Disadvantages

- Doesn't account for server load
- May overload slower servers
- Doesn't consider request complexity

## Least Connections

Routes requests to the backend with the fewest active connections.

```yaml
upstreams:
  - name: api-service
    targets:
      - url: http://backend-1:3000
      - url: http://backend-2:3000
      - url: http://backend-3:3000
    load_balance: least_conn
```

### How It Works

```
Backend 1: 5 connections  ← New request goes here (lowest)
Backend 2: 8 connections
Backend 3: 12 connections
```

### When to Use

- Requests have variable processing times
- Some requests are more resource-intensive
- Backend servers have similar capacity

### Advantages

- Adapts to actual server load
- Better handling of slow requests
- Prevents overloading busy servers

### Disadvantages

- Slight overhead for connection tracking
- New servers may be overwhelmed initially
- Doesn't account for server capacity

## Random

Selects a random healthy backend for each request.

```yaml
upstreams:
  - name: api-service
    targets:
      - url: http://backend-1:3000
      - url: http://backend-2:3000
      - url: http://backend-3:3000
    load_balance: random
```

### How It Works

Each request is routed to a randomly selected healthy backend.

### When to Use

- Simple load distribution without state
- Testing and development environments
- When statistical distribution is acceptable

### Advantages

- No coordination needed
- Simple implementation
- Works well at scale

### Disadvantages

- Distribution may be uneven short-term
- No consideration of server load
- May cause hot spots temporarily

## Weighted Round Robin

Distributes requests based on assigned weights, allowing more powerful servers to handle more traffic.

```yaml
upstreams:
  - name: api-service
    targets:
      - url: http://large-server:3000
        weight: 5 # Handles 5x more traffic
      - url: http://medium-server:3000
        weight: 3 # Handles 3x more traffic
      - url: http://small-server:3000
        weight: 1 # Baseline
    load_balance: weighted_round_robin
```

### How It Works

With weights 5:3:1, over 9 requests:

```
Requests 1-5 → large-server (weight 5)
Requests 6-8 → medium-server (weight 3)
Request 9    → small-server (weight 1)
```

### When to Use

- Backend servers have different capacities
- Migrating traffic gradually between versions
- Implementing canary deployments

### Advantages

- Accounts for server capacity differences
- Fine-grained traffic control
- Useful for gradual rollouts

### Disadvantages

- Requires knowledge of server capacity
- More complex configuration
- Weights need adjustment as capacity changes

## Configuring Multiple Upstreams

Different services can use different strategies:

```yaml
upstreams:
  # High-throughput API - use round robin
  - name: api-service
    targets:
      - url: http://api-1:3000
      - url: http://api-2:3000
      - url: http://api-3:3000
    load_balance: round_robin

  # Long-running operations - use least connections
  - name: processing-service
    targets:
      - url: http://processor-1:3000
      - url: http://processor-2:3000
    load_balance: least_conn

  # Mixed capacity cluster - use weighted
  - name: compute-service
    targets:
      - url: http://compute-large:3000
        weight: 4
      - url: http://compute-small:3000
        weight: 1
    load_balance: weighted_round_robin
```

## Health-Aware Load Balancing

All load balancing strategies automatically skip unhealthy backends:

```yaml
upstreams:
  - name: api-service
    targets:
      - url: http://backend-1:3000
      - url: http://backend-2:3000
      - url: http://backend-3:3000
    load_balance: round_robin
    health_check:
      path: /health
      interval: 10s
      timeout: 2s
```

When a backend fails health checks:

```
Before failure:
  Request 1 → Backend 1
  Request 2 → Backend 2
  Request 3 → Backend 3

After Backend 2 fails:
  Request 1 → Backend 1
  Request 2 → Backend 3  # Backend 2 skipped
  Request 3 → Backend 1
```

See [Health Checks](./health-checks.md) for detailed configuration.

## Connection Tracking

For `least_conn` strategy, Relaypoint tracks active connections per backend:

```
┌─────────────────┐
│   Relaypoint    │
│                 │
│ Backend 1: 5    │──→ Backend 1 (5 active)
│ Backend 2: 3    │──→ Backend 2 (3 active)
│ Backend 3: 8    │──→ Backend 3 (8 active)
└─────────────────┘
```

Connections are:

- Incremented when a request starts
- Decremented when the response completes (or fails)

## Choosing a Strategy

Use this decision tree:

```
Are your servers different capacities?
├─ Yes → Use weighted_round_robin
└─ No
    │
    Do requests have variable processing times?
    ├─ Yes → Use least_conn
    └─ No
        │
        Need predictable distribution?
        ├─ Yes → Use round_robin
        └─ No → Use random
```

### Strategy Comparison

| Scenario                               | Recommended Strategy                    |
| -------------------------------------- | --------------------------------------- |
| Identical servers, consistent requests | `round_robin`                           |
| Identical servers, variable requests   | `least_conn`                            |
| Mixed server capacities                | `weighted_round_robin`                  |
| Canary deployment (90/10 split)        | `weighted_round_robin` with weights 9:1 |
| Simple setup, stateless                | `random`                                |
| Database read replicas                 | `least_conn`                            |
| Static file servers                    | `round_robin`                           |
| API with some slow endpoints           | `least_conn`                            |

## Canary Deployments

Use weighted round robin for gradual rollouts:

```yaml
# Stage 1: 10% traffic to new version
upstreams:
  - name: api-service
    targets:
      - url: http://api-v1:3000
        weight: 9
      - url: http://api-v2:3000 # New version
        weight: 1
    load_balance: weighted_round_robin
```

```yaml
# Stage 2: 50% traffic to new version
upstreams:
  - name: api-service
    targets:
      - url: http://api-v1:3000
        weight: 1
      - url: http://api-v2:3000
        weight: 1
    load_balance: weighted_round_robin
```

```yaml
# Stage 3: 100% traffic to new version
upstreams:
  - name: api-service
    targets:
      - url: http://api-v2:3000
    load_balance: round_robin
```

## Blue-Green Deployments

Switch traffic between environments:

```yaml
# Blue environment active
upstreams:
  - name: api-service
    targets:
      - url: http://blue-1:3000
      - url: http://blue-2:3000
    load_balance: round_robin

# Switch to green (update config and reload)
upstreams:
  - name: api-service
    targets:
      - url: http://green-1:3000
      - url: http://green-2:3000
    load_balance: round_robin
```

## Monitoring Load Balancing

### Metrics

Relaypoint exposes metrics for monitoring load distribution:

```bash
curl http://localhost:9090/metrics | grep upstream
```

Key metrics:

- `gateway_requests_total{upstream="..."}` - Requests per upstream
- `gateway_upstream_healthy{upstream="...",target="..."}` - Backend health status
- `gateway_request_duration_seconds{upstream="..."}` - Latency per upstream

### Stats Endpoint

```bash
curl http://localhost:8080/stats
```

Returns request counts and latency percentiles per route.

## Troubleshooting

### Uneven Distribution

**Symptom**: One backend receives significantly more traffic.

**Causes**:

1. Using `round_robin` with backends of different speeds
2. Health check failures on some backends
3. Connection pooling affecting `least_conn`

**Solutions**:

1. Switch to `least_conn` for variable workloads
2. Check health check logs
3. Review backend logs for errors

### Backend Overload

**Symptom**: Specific backend becomes overloaded.

**Causes**:

1. Weight configuration too high
2. Backend slower than expected
3. Health checks not detecting issues

**Solutions**:

1. Adjust weights
2. Use `least_conn` strategy
3. Tune health check intervals

### All Traffic to One Backend

**Symptom**: Other backends receive no traffic.

**Causes**:

1. Other backends marked unhealthy
2. Configuration error in targets
3. DNS resolution issues

**Solutions**:

1. Check health endpoint on backends
2. Verify target URLs
3. Test direct backend connectivity

## Next Steps

- [Health Checks](./health-checks.md) - Configure backend health monitoring
- [Rate Limiting](./rate-limiting.md) - Protect your APIs
- [Metrics](./metrics.md) - Monitor your gateway
