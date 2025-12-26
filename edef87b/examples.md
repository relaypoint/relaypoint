# Examples

This page provides complete, ready-to-use configuration examples for common use cases.

## Basic API Gateway

The simplest possible configuration for proxying requests to a backend:

```yaml
# basic.yml
server:
  port: 8080

upstreams:
  - name: backend
    targets:
      - url: http://localhost:3000

routes:
  - name: all-traffic
    path: /**
    upstream: backend
```

## Load-Balanced Microservices

Route to multiple microservices with load balancing and health checks:

```yaml
# microservices.yml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

metrics:
  enabled: true
  port: 9090

upstreams:
  - name: user-service
    targets:
      - url: http://users-1:3000
      - url: http://users-2:3000
      - url: http://users-3:3000
    load_balance: round_robin
    health_check:
      path: /health
      interval: 10s
      timeout: 2s

  - name: order-service
    targets:
      - url: http://orders-1:3000
      - url: http://orders-2:3000
    load_balance: least_conn
    health_check:
      path: /health
      interval: 10s
      timeout: 2s

  - name: product-service
    targets:
      - url: http://products:3000
    health_check:
      path: /health
      interval: 10s
      timeout: 2s

routes:
  - name: users
    path: /api/users/**
    upstream: user-service
    strip_path: false

  - name: orders
    path: /api/orders/**
    upstream: order-service
    strip_path: false

  - name: products
    path: /api/products/**
    upstream: product-service
    strip_path: false
```

## Rate-Limited Public API

Public API with tiered rate limiting:

```yaml
# public-api.yml
server:
  port: 8080

metrics:
  enabled: true
  port: 9090

rate_limit:
  enabled: true
  default_rps: 60 # Anonymous users: 60 req/sec
  default_burst: 120
  per_ip: true
  per_api_key: true
  cleanup_interval: 5m

upstreams:
  - name: api
    targets:
      - url: http://api-1:3000
      - url: http://api-2:3000
    load_balance: round_robin
    health_check:
      path: /health
      interval: 10s

routes:
  # Public read endpoints - higher limits
  - name: public-read
    path: /api/v1/**
    methods: [GET, HEAD]
    upstream: api
    rate_limit:
      enabled: true
      requests_per_second: 100
      burst_size: 200

  # Write endpoints - stricter limits
  - name: public-write
    path: /api/v1/**
    methods: [POST, PUT, PATCH, DELETE]
    upstream: api
    rate_limit:
      enabled: true
      requests_per_second: 20
      burst_size: 40

  # Authentication - very strict (prevent brute force)
  - name: auth
    path: /api/v1/auth/**
    upstream: api
    rate_limit:
      enabled: true
      requests_per_second: 5
      burst_size: 10

api_keys:
  # Free tier
  - key: "pk_free_sample123"
    name: "free-tier"
    requests_per_second: 60
    burst_size: 120
    enabled: true

  # Pro tier
  - key: "pk_pro_sample456"
    name: "pro-tier"
    requests_per_second: 600
    burst_size: 1200
    enabled: true

  # Enterprise tier
  - key: "pk_ent_sample789"
    name: "enterprise-tier"
    requests_per_second: 6000
    burst_size: 12000
    enabled: true
```

## Multi-Tenant SaaS

Route based on subdomain for multi-tenant applications:

```yaml
# multi-tenant.yml
server:
  port: 8080

upstreams:
  - name: tenant-service
    targets:
      - url: http://tenant-app:3000
    load_balance: round_robin
    health_check:
      path: /health
      interval: 10s

  - name: main-app
    targets:
      - url: http://main-app:3000
    load_balance: round_robin
    health_check:
      path: /health
      interval: 10s

  - name: admin-service
    targets:
      - url: http://admin:3000
    health_check:
      path: /health
      interval: 10s

routes:
  # Admin panel
  - name: admin
    host: admin.example.com
    path: /**
    upstream: admin-service
    rate_limit:
      enabled: true
      requests_per_second: 100
      burst_size: 200

  # API for all tenants
  - name: tenant-api
    host: "*.example.com"
    path: /api/**
    upstream: tenant-service
    headers:
      X-Tenant-Source: "subdomain"

  # Main marketing site
  - name: main
    host: www.example.com
    path: /**
    upstream: main-app

  # Catch-all
  - name: default
    path: /**
    upstream: main-app
```

## Canary Deployment

Gradually roll out a new version:

```yaml
# canary.yml
server:
  port: 8080

upstreams:
  # Stable version (90% traffic)
  - name: api-stable
    targets:
      - url: http://api-v1-1:3000
      - url: http://api-v1-2:3000
      - url: http://api-v1-3:3000
    load_balance: round_robin
    health_check:
      path: /health
      interval: 5s

  # Canary version (10% traffic)
  - name: api-canary
    targets:
      - url: http://api-v2:3000
    health_check:
      path: /health
      interval: 5s

  # Combined weighted upstream
  - name: api-weighted
    targets:
      - url: http://api-v1-1:3000
        weight: 3
      - url: http://api-v1-2:3000
        weight: 3
      - url: http://api-v1-3:3000
        weight: 3
      - url: http://api-v2:3000
        weight: 1 # 10% of traffic
    load_balance: weighted_round_robin
    health_check:
      path: /health
      interval: 5s

routes:
  - name: api
    path: /api/**
    upstream: api-weighted
```

## Read/Write Splitting

Route reads to replicas, writes to primary:

```yaml
# read-write-split.yml
server:
  port: 8080

upstreams:
  # Write to primary
  - name: primary
    targets:
      - url: http://db-primary:3000
    health_check:
      path: /health
      interval: 5s

  # Read from replicas
  - name: replicas
    targets:
      - url: http://db-replica-1:3000
      - url: http://db-replica-2:3000
      - url: http://db-replica-3:3000
    load_balance: least_conn
    health_check:
      path: /health
      interval: 5s

routes:
  # Read operations go to replicas
  - name: reads
    path: /api/**
    methods: [GET, HEAD, OPTIONS]
    upstream: replicas

  # Write operations go to primary
  - name: writes
    path: /api/**
    methods: [POST, PUT, PATCH, DELETE]
    upstream: primary
```

## API Versioning

Support multiple API versions:

```yaml
# versioned-api.yml
server:
  port: 8080

upstreams:
  - name: api-v1
    targets:
      - url: http://api-v1:3000
    health_check:
      path: /health
      interval: 10s

  - name: api-v2
    targets:
      - url: http://api-v2:3000
    health_check:
      path: /health
      interval: 10s

  - name: api-v3
    targets:
      - url: http://api-v3:3000
    health_check:
      path: /health
      interval: 10s

routes:
  # Version 3 (latest)
  - name: v3
    path: /api/v3/**
    upstream: api-v3
    strip_path: false

  # Version 2
  - name: v2
    path: /api/v2/**
    upstream: api-v2
    strip_path: false

  # Version 1 (deprecated, rate limited)
  - name: v1
    path: /api/v1/**
    upstream: api-v1
    strip_path: false
    rate_limit:
      enabled: true
      requests_per_second: 10
      burst_size: 20
    headers:
      X-API-Deprecated: "true"
      X-API-Sunset-Date: "2025-12-31"

  # Default to latest
  - name: default
    path: /api/**
    upstream: api-v3
```

## Kubernetes Ingress Replacement

Replace Kubernetes Ingress with Relaypoint:

```yaml
# k8s-gateway.yml
server:
  port: 8080

metrics:
  enabled: true
  port: 9090

upstreams:
  - name: frontend
    targets:
      - url: http://frontend-service.default.svc.cluster.local:80
    health_check:
      path: /health
      interval: 10s

  - name: backend-api
    targets:
      - url: http://api-service.default.svc.cluster.local:80
    load_balance: round_robin
    health_check:
      path: /healthz
      interval: 10s

  - name: websocket
    targets:
      - url: http://ws-service.default.svc.cluster.local:80
    health_check:
      path: /health
      interval: 10s

routes:
  # API routes
  - name: api
    path: /api/**
    upstream: backend-api
    timeout: 30s

  # WebSocket routes
  - name: websocket
    path: /ws/**
    upstream: websocket
    timeout: 3600s # Long timeout for websockets

  # Frontend (catch-all)
  - name: frontend
    path: /**
    upstream: frontend
```

## Docker Compose Full Stack

Complete example with mock backends:

```yaml
# docker-compose.yml
version: "3.8"

services:
  relaypoint:
    image: ghcr.io/relaypoint/relaypoint:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - ./relaypoint.yml:/etc/relaypoint/relaypoint.yml:ro
    command: ["-config", "/etc/relaypoint/relaypoint.yml"]
    depends_on:
      - users
      - orders
      - products

  users:
    image: nginx:alpine
    volumes:
      - ./mock/users:/usr/share/nginx/html:ro

  orders:
    image: nginx:alpine
    volumes:
      - ./mock/orders:/usr/share/nginx/html:ro

  products:
    image: nginx:alpine
    volumes:
      - ./mock/products:/usr/share/nginx/html:ro

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

```yaml
# relaypoint.yml
server:
  port: 8080

metrics:
  enabled: true
  port: 9090

rate_limit:
  enabled: true
  default_rps: 100
  default_burst: 200
  per_ip: true

upstreams:
  - name: users
    targets:
      - url: http://users:80

  - name: orders
    targets:
      - url: http://orders:80

  - name: products
    targets:
      - url: http://products:80

routes:
  - name: users-api
    path: /api/users/**
    upstream: users

  - name: orders-api
    path: /api/orders/**
    upstream: orders

  - name: products-api
    path: /api/products/**
    upstream: products
```

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "relaypoint"
    static_configs:
      - targets: ["relaypoint:9090"]
```

## Testing Your Configuration

After setting up any configuration:

```bash
# Start Relaypoint
./relaypoint -config your-config.yml

# Test health endpoint
curl http://localhost:8080/health

# Test routing
curl http://localhost:8080/api/users

# Check metrics
curl http://localhost:9090/metrics

# Check stats
curl http://localhost:8080/stats
```

## Next Steps

- [Configuration Reference](./configuration.md) - All configuration options
- [Troubleshooting](./troubleshooting.md) - Debug issues
- [Best Practices](./best-practices.md) - Production recommendations
