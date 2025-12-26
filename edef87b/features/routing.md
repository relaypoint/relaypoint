# Routing

Relaypoint provides a powerful and flexible routing system to direct incoming requests to the appropriate backend services.

## Basic Routing

At its simplest, a route maps a URL path to an upstream:

```yaml
routes:
  - name: api
    path: /api
    upstream: backend-service
```

This routes exact matches of `/api` to the `backend-service` upstream.

## Path Patterns

### Exact Matching

Matches only the exact path:

```yaml
routes:
  - name: users-list
    path: /api/users
    upstream: user-service
```

| Request Path     | Match |
| ---------------- | ----- |
| `/api/users`     | ✓ Yes |
| `/api/users/`    | ✗ No  |
| `/api/users/123` | ✗ No  |

### Single-Segment Wildcard (`*`)

Matches exactly one path segment:

```yaml
routes:
  - name: user-by-id
    path: /api/users/*
    upstream: user-service
```

| Request Path            | Match |
| ----------------------- | ----- |
| `/api/users/123`        | ✓ Yes |
| `/api/users/abc`        | ✓ Yes |
| `/api/users`            | ✗ No  |
| `/api/users/123/orders` | ✗ No  |

### Multi-Segment Wildcard (`**`)

Matches zero or more path segments:

```yaml
routes:
  - name: api-catchall
    path: /api/**
    upstream: api-service
```

| Request Path                | Match |
| --------------------------- | ----- |
| `/api`                      | ✓ Yes |
| `/api/users`                | ✓ Yes |
| `/api/users/123`            | ✓ Yes |
| `/api/users/123/orders/456` | ✓ Yes |
| `/other`                    | ✗ No  |

### Named Parameters

Capture path segments as named parameters:

```yaml
routes:
  # Colon syntax
  - name: user-detail
    path: /api/users/:userId
    upstream: user-service

  # Curly brace syntax (alternative)
  - name: order-detail
    path: /api/orders/{orderId}
    upstream: order-service

  # Multiple parameters
  - name: user-order
    path: /api/users/:userId/orders/:orderId
    upstream: order-service
```

| Pattern                          | Request Path          | Captured Parameters     |
| -------------------------------- | --------------------- | ----------------------- |
| `/users/:id`                     | `/users/123`          | `id=123`                |
| `/users/:userId/orders/:orderId` | `/users/42/orders/99` | `userId=42, orderId=99` |

## Route Priority

When multiple routes could match a request, Relaypoint uses priority ordering:

1. **More specific paths** take precedence over less specific ones
2. **Exact segments** beat wildcards
3. **Single wildcards (`*`)** beat multi-segment wildcards (`**`)
4. **Named parameters** are treated like single wildcards

### Example Priority

```yaml
routes:
  # Priority 1: Exact match (highest)
  - name: users-list
    path: /api/v1/users
    upstream: users

  # Priority 2: Specific path with wildcard
  - name: user-by-id
    path: /api/v1/users/*
    upstream: users

  # Priority 3: Less specific path
  - name: v1-api
    path: /api/v1/**
    upstream: api

  # Priority 4: Catch-all (lowest)
  - name: catchall
    path: /**
    upstream: default
```

| Request Path        | Matched Route |
| ------------------- | ------------- |
| `/api/v1/users`     | `users-list`  |
| `/api/v1/users/123` | `user-by-id`  |
| `/api/v1/orders`    | `v1-api`      |
| `/api/v2/anything`  | `catchall`    |

## Host-Based Routing

Route requests based on the `Host` header:

```yaml
routes:
  # Route by specific host
  - name: api-routes
    host: api.example.com
    path: /**
    upstream: api-service

  # Wildcard host matching
  - name: tenant-routes
    host: "*.example.com"
    path: /**
    upstream: tenant-service

  # Default (no host specified matches all)
  - name: default
    path: /**
    upstream: default-service
```

| Host                  | Request Path | Matched Route   |
| --------------------- | ------------ | --------------- |
| `api.example.com`     | `/users`     | `api-routes`    |
| `tenant1.example.com` | `/data`      | `tenant-routes` |
| `other.com`           | `/anything`  | `default`       |

## Method Filtering

Restrict routes to specific HTTP methods:

```yaml
routes:
  # Read operations
  - name: users-read
    path: /api/users/**
    methods:
      - GET
      - HEAD
    upstream: user-read-service

  # Write operations
  - name: users-write
    path: /api/users/**
    methods:
      - POST
      - PUT
      - PATCH
      - DELETE
    upstream: user-write-service
```

If `methods` is not specified or empty, all HTTP methods are allowed.

## Path Stripping

Remove the matched path prefix before forwarding to the upstream:

```yaml
routes:
  - name: user-service
    path: /api/v1/users/**
    upstream: user-service
    strip_path: true
```

| `strip_path` | Request Path        | Upstream Path       |
| ------------ | ------------------- | ------------------- |
| `false`      | `/api/v1/users/123` | `/api/v1/users/123` |
| `true`       | `/api/v1/users/123` | `/123`              |

This is useful when your backend services don't expect the gateway prefix.

## Header Injection

Add custom headers to upstream requests:

```yaml
routes:
  - name: api
    path: /api/**
    upstream: api-service
    headers:
      X-Gateway: "relaypoint"
      X-Request-Source: "public"
      X-Service-Version: "v1"
```

These headers are added to every request forwarded to the upstream.

## Request Forwarding Headers

Relaypoint automatically adds standard proxy headers:

| Header              | Description                              |
| ------------------- | ---------------------------------------- |
| `X-Forwarded-For`   | Client IP address (appended to existing) |
| `X-Forwarded-Host`  | Original `Host` header                   |
| `X-Forwarded-Proto` | Original protocol (`http` or `https`)    |
| `X-Real-IP`         | Client IP address                        |

## Route-Specific Rate Limiting

Apply rate limits to specific routes:

```yaml
routes:
  - name: expensive-operation
    path: /api/process
    upstream: processor
    rate_limit:
      enabled: true
      requests_per_second: 10
      burst_size: 20

  - name: normal-api
    path: /api/**
    upstream: api-service
    # No rate limit - uses global defaults
```

See [Rate Limiting](./rate-limiting.md) for more details.

## Route-Specific Timeouts

Set custom timeouts per route:

```yaml
routes:
  - name: quick-api
    path: /api/fast/**
    upstream: fast-service
    timeout: 5s

  - name: slow-operation
    path: /api/reports/**
    upstream: report-service
    timeout: 120s
```

## Combining Patterns

Create sophisticated routing rules by combining patterns:

```yaml
upstreams:
  - name: user-service
    targets:
      - url: http://users:3000

  - name: order-service
    targets:
      - url: http://orders:3000

  - name: admin-service
    targets:
      - url: http://admin:3000

routes:
  # Admin routes with strict rate limiting
  - name: admin-api
    host: admin.example.com
    path: /api/**
    methods:
      - GET
      - POST
      - PUT
      - DELETE
    upstream: admin-service
    rate_limit:
      enabled: true
      requests_per_second: 100
      burst_size: 150
    headers:
      X-Admin-Request: "true"

  # User service - read operations
  - name: users-read
    path: /api/v1/users/**
    methods:
      - GET
    upstream: user-service

  # User service - write operations (stricter rate limit)
  - name: users-write
    path: /api/v1/users/**
    methods:
      - POST
      - PUT
      - DELETE
    upstream: user-service
    rate_limit:
      enabled: true
      requests_per_second: 20
      burst_size: 30

  # Order service with path stripping
  - name: orders
    path: /api/v1/orders/**
    upstream: order-service
    strip_path: true
    timeout: 60s
```

## Common Routing Patterns

### Versioned API

```yaml
routes:
  - name: v2-api
    path: /api/v2/**
    upstream: api-v2

  - name: v1-api
    path: /api/v1/**
    upstream: api-v1

  - name: latest-api
    path: /api/**
    upstream: api-v2 # Default to latest
```

### Microservices Gateway

```yaml
routes:
  - name: users
    path: /users/**
    upstream: user-service
    strip_path: true

  - name: products
    path: /products/**
    upstream: product-service
    strip_path: true

  - name: orders
    path: /orders/**
    upstream: order-service
    strip_path: true
```

### Multi-Tenant Routing

```yaml
routes:
  # Tenant-specific subdomains
  - name: tenant-api
    host: "*.app.example.com"
    path: /api/**
    upstream: tenant-service

  # Main application
  - name: main-api
    host: app.example.com
    path: /api/**
    upstream: main-service
```

### Read/Write Splitting

```yaml
routes:
  - name: read-operations
    path: /api/**
    methods:
      - GET
      - HEAD
      - OPTIONS
    upstream: read-replicas

  - name: write-operations
    path: /api/**
    methods:
      - POST
      - PUT
      - PATCH
      - DELETE
    upstream: primary-database
```

## Debugging Routes

To understand which route is matching your requests:

1. **Check the logs** - Relaypoint logs route matches at debug level
2. **Use the stats endpoint** - `GET /stats` shows request counts per route
3. **Check metrics** - Prometheus metrics include route labels

```bash
# View stats
curl http://localhost:8080/stats | jq

# Check specific route in metrics
curl http://localhost:9090/metrics | grep gateway_requests_total
```

## Next Steps

- [Load Balancing](./load-balancing.md) - Configure backend load balancing
- [Rate Limiting](./rate-limiting.md) - Protect your APIs
- [Health Checks](./health-checks.md) - Monitor backend health
