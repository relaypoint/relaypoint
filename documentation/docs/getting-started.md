# Getting Started

This guide will help you get Relaypoint up and running in under 5 minutes.

## Prerequisites

- A Linux, macOS, or Windows system
- One or more backend services to proxy requests to
- (Optional) Go 1.24+ if building from source

## Step 1: Download Relaypoint

### Option A: Download Pre-built Binary

Download the latest release for your platform:

```bash
# Linux (amd64)
curl -L https://github.com/relaypoint/relaypoint/releases/latest/download/relaypoint-linux-amd64.tar.gz | tar xz

# Linux (arm64)
curl -L https://github.com/relaypoint/relaypoint/releases/latest/download/relaypoint-linux-arm64.tar.gz | tar xz

# macOS (Intel)
curl -L https://github.com/relaypoint/relaypoint/releases/latest/download/relaypoint-darwin-amd64.tar.gz | tar xz

# macOS (Apple Silicon)
curl -L https://github.com/relaypoint/relaypoint/releases/latest/download/relaypoint-darwin-arm64.tar.gz | tar xz
```

### Option B: Build from Source

```bash
git clone https://github.com/relaypoint/relaypoint.git
cd relaypoint
make build
```

The binary will be created at `dist/relaypoint`.

## Step 2: Create Configuration

Create a file named `relaypoint.yml`:

```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

metrics:
  enabled: true
  port: 9090

upstreams:
  - name: my-backend
    targets:
      - url: http://localhost:3000
    load_balance: round_robin
    health_check:
      path: /health
      interval: 10s
      timeout: 2s

routes:
  - name: api-routes
    path: /api/**
    upstream: my-backend
```

## Step 3: Start Relaypoint

```bash
./relaypoint -config relaypoint.yml
```

You should see output like:

```
{"level":"INFO","msg":"Starting RelayPoint","config":"relaypoint.yml"}
{"level":"INFO","msg":"configuration loaded","routes":1,"upstreams":1,"rate_limiting":true}
{"level":"INFO","msg":"metrics server starting","port":9090,"path":"/metrics"}
{"level":"INFO","msg":"relaypoint API Gateway starting","address":"0.0.0.0:8080"}
```

## Step 4: Test Your Setup

```bash
# Test the gateway
curl http://localhost:8080/api/hello

# Check health
curl http://localhost:8080/health

# View metrics
curl http://localhost:9090/metrics

# View stats
curl http://localhost:8080/stats
```

## Step 5: Add More Features

Now that you have the basics working, explore additional features:

### Add Load Balancing

```yaml
upstreams:
  - name: my-backend
    targets:
      - url: http://localhost:3001
        weight: 2
      - url: http://localhost:3002
        weight: 1
      - url: http://localhost:3003
        weight: 1
    load_balance: weighted_round_robin
```

### Add Rate Limiting

```yaml
rate_limit:
  enabled: true
  default_rps: 100
  default_burst: 200
  per_ip: true
  per_api_key: true

routes:
  - name: api-routes
    path: /api/**
    upstream: my-backend
    rate_limit:
      enabled: true
      requests_per_second: 50
      burst_size: 100
```

### Add API Keys

```yaml
api_keys:
  - key: "pk_live_abc123"
    name: "production-app"
    requests_per_second: 1000
    burst_size: 2000
    enabled: true
```

## Next Steps

- [Configuration Reference](./configuration.md) - Complete configuration options
- [Routing](./routing.md) - Advanced routing patterns
- [Load Balancing](./load-balancing.md) - Load balancing strategies
- [Rate Limiting](./rate-limiting.md) - Protect your APIs
- [Metrics](./metrics.md) - Monitoring and observability

## Getting Help

If you run into issues:

1. Check the [Troubleshooting Guide](./troubleshooting.md)
2. Search [GitHub Issues](https://github.com/relaypoint/relaypoint/issues)
3. Ask in [GitHub Discussions](https://github.com/relaypoint/relaypoint/discussions)
