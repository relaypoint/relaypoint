# Troubleshooting

This guide helps you diagnose and resolve common issues with Relaypoint.

## Quick Diagnostics

Run these commands to quickly assess the gateway status:

```bash
# Check if Relaypoint is running
curl -s http://localhost:8080/health

# Check metrics endpoint
curl -s http://localhost:9090/metrics | head -20

# Check stats
curl -s http://localhost:8080/stats | jq

# Test a route
curl -v http://localhost:8080/your/route
```

## Startup Issues

### Configuration File Not Found

**Error**:

```
Failed to load configuration: failed to read config file: open relaypoint.yml: no such file or directory
```

**Solution**:

```bash
# Specify the correct path
./relaypoint -config /path/to/your/config.yml

# Or create a config in the current directory
cat > relaypoint.yml << 'EOF'
server:
  port: 8080
upstreams:
  - name: backend
    targets:
      - url: http://localhost:3000
routes:
  - name: default
    path: /**
    upstream: backend
EOF
```

### Invalid Configuration

**Error**:

```
Invalid configuration: at least one route must be defined
```

**Solution**: Ensure your configuration has required fields:

```yaml
# Minimum valid configuration
server:
  port: 8080

upstreams:
  - name: backend # Required: upstream name
    targets:
      - url: http://localhost:3000 # Required: at least one target

routes:
  - name: default # Optional but recommended
    path: /** # Required: path pattern
    upstream: backend # Required: must match an upstream name
```

### Port Already in Use

**Error**:

```
server error: listen tcp :8080: bind: address already in use
```

**Solution**:

```bash
# Find what's using the port
lsof -i :8080

# Kill the process or use a different port
./relaypoint -config config.yml  # Change port in config

# Or in config:
server:
  port: 8081
```

### Invalid Upstream URL

**Error**:

```
invalid upstream URL http://backend: parse "http://backend": invalid character " " in host name
```

**Solution**: Ensure URLs are properly formatted:

```yaml
upstreams:
  - name: backend
    targets:
      # Good
      - url: http://backend:3000
      - url: http://192.168.1.100:3000
      - url: http://localhost:3000

      # Bad
      - url: backend:3000 # Missing scheme
      - url: http://back end:3000 # Space in hostname
```

## Request Routing Issues

### 404 Not Found

**Symptoms**: All requests return 404.

**Causes**:

1. No matching route
2. Route path doesn't match request
3. Host-based routing mismatch

**Debugging**:

```bash
# Check your routes
cat config.yml | grep -A5 "routes:"

# Test with verbose curl
curl -v http://localhost:8080/api/users

# Check if any routes are configured
curl -s http://localhost:9090/metrics | grep requests_total
```

**Solution**:

```yaml
routes:
  # Use /** to match all paths
  - name: catchall
    path: /**
    upstream: backend

  # Or be explicit
  - name: api
    path: /api/** # Note the ** for wildcard
    upstream: backend
```

### 502 Bad Gateway

**Symptoms**: Requests return 502 errors.

**Causes**:

1. Backend is not running
2. Wrong backend URL
3. Backend rejecting connections

**Debugging**:

```bash
# Test backend directly
curl -v http://localhost:3000/health

# Check backend connectivity from Relaypoint host
nc -zv localhost 3000

# Check error metrics
curl -s http://localhost:9090/metrics | grep errors_total
```

**Solution**:

1. Verify backend is running
2. Check target URLs in configuration
3. Ensure no firewall blocking connections

### 503 Service Unavailable

**Symptoms**: Requests return 503 errors.

**Causes**:

1. All backends are unhealthy
2. Health checks failing

**Debugging**:

```bash
# Check upstream health
curl -s http://localhost:9090/metrics | grep upstream_healthy

# Test backend health endpoint directly
curl -v http://localhost:3000/health
```

**Solution**:

1. Fix backend health issues
2. Verify health check path is correct
3. Increase health check timeout if needed

### Wrong Backend Receiving Requests

**Symptoms**: Requests routed to unexpected backend.

**Causes**:

1. Route priority issues
2. Overlapping path patterns

**Debugging**:

```bash
# Check which route matched (via metrics)
curl http://localhost:8080/your/path
curl -s http://localhost:9090/metrics | grep requests_total | tail -5
```

**Solution**: Order routes from most specific to least specific:

```yaml
routes:
  # Most specific first
  - name: users-by-id
    path: /api/users/:id
    upstream: user-detail-service

  # Then broader patterns
  - name: users
    path: /api/users/**
    upstream: user-service

  # Catch-all last
  - name: default
    path: /**
    upstream: default-service
```

## Load Balancing Issues

### Traffic Not Distributed

**Symptoms**: All traffic goes to one backend.

**Causes**:

1. Other backends marked unhealthy
2. Only one target configured
3. Configuration error

**Debugging**:

```bash
# Check backend health
curl -s http://localhost:9090/metrics | grep upstream_healthy

# Make multiple requests and check response
for i in {1..10}; do
  curl -s http://localhost:8080/api/test | grep -o '"server":"[^"]*"'
done
```

**Solution**:

1. Fix unhealthy backends
2. Verify all targets are configured:

```yaml
upstreams:
  - name: api
    targets:
      - url: http://backend-1:3000
      - url: http://backend-2:3000 # Make sure all are listed
      - url: http://backend-3:3000
```

### Backend Overload

**Symptoms**: One backend getting too much traffic.

**Causes**:

1. Using round-robin with unequal backend capacity
2. Weights not configured correctly

**Solution**:

```yaml
upstreams:
  - name: api
    targets:
      - url: http://large-server:3000
        weight: 4 # More traffic
      - url: http://small-server:3000
        weight: 1 # Less traffic
    load_balance: weighted_round_robin
```

## Rate Limiting Issues

### Rate Limits Not Working

**Symptoms**: Requests exceeding limits not blocked.

**Causes**:

1. Rate limiting not enabled
2. High burst allowing spikes
3. Wrong limit type

**Debugging**:

```bash
# Check if rate limiting is enabled
grep -A5 "rate_limit:" config.yml

# Make rapid requests
for i in {1..100}; do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/api/test
done | sort | uniq -c
```

**Solution**:

```yaml
rate_limit:
  enabled: true # Must be true
  default_rps: 10 # Lower for testing
  default_burst: 10 # Lower burst
  per_ip: true

routes:
  - name: api
    path: /api/**
    upstream: backend
    rate_limit:
      enabled: true # Enable per-route
      requests_per_second: 5
      burst_size: 10
```

### Legitimate Users Being Blocked

**Symptoms**: Real users getting 429 errors.

**Causes**:

1. Limits too low
2. Shared IP (NAT/proxy)
3. Burst not high enough

**Solution**:

```yaml
rate_limit:
  default_rps: 100 # Increase RPS
  default_burst: 500 # Higher burst for spikes

# Or use API keys for legitimate users
api_keys:
  - key: "customer_key"
    name: "verified-customer"
    requests_per_second: 1000
    burst_size: 2000
```

### Rate Limit Metrics Not Showing

**Symptoms**: No rate limit metrics despite 429 responses.

**Causes**:

1. Metrics not enabled
2. Looking at wrong metrics

**Debugging**:

```bash
curl -s http://localhost:9090/metrics | grep rate_limit
```

**Solution**:

```yaml
metrics:
  enabled: true
```

## Health Check Issues

### All Backends Showing Unhealthy

**Symptoms**: `upstream_healthy` all showing 0.

**Causes**:

1. Health endpoint not implemented
2. Wrong health check path
3. Health check timeout too short

**Debugging**:

```bash
# Test health endpoint directly
curl -v http://backend:3000/health

# Check health check configuration
grep -A4 "health_check:" config.yml
```

**Solution**:

```yaml
upstreams:
  - name: api
    targets:
      - url: http://backend:3000
    health_check:
      path: /health # Verify this path exists
      interval: 10s
      timeout: 5s # Increase if backend is slow
```

### Health Check Flapping

**Symptoms**: Backend alternates healthy/unhealthy.

**Causes**:

1. Network instability
2. Backend at capacity
3. Timeout too aggressive

**Solution**:

```yaml
health_check:
  path: /health
  interval: 30s # Less frequent checks
  timeout: 10s # More generous timeout
```

## Metrics Issues

### Metrics Endpoint Not Available

**Symptoms**: `curl localhost:9090/metrics` fails.

**Causes**:

1. Metrics disabled
2. Wrong port
3. Metrics server failed to start

**Debugging**:

```bash
# Check if metrics port is listening
netstat -tlnp | grep 9090

# Check Relaypoint logs for errors
# (logs go to stderr)
```

**Solution**:

```yaml
metrics:
  enabled: true
  port: 9090
  path: /metrics
```

### Missing Metrics

**Symptoms**: Expected metrics not appearing.

**Causes**:

1. No traffic to generate metrics
2. Labels don't match query

**Debugging**:

```bash
# Get all metrics
curl -s http://localhost:9090/metrics | grep gateway

# Generate some traffic
for i in {1..10}; do curl -s http://localhost:8080/api/test; done

# Check again
curl -s http://localhost:9090/metrics | grep gateway
```

## Performance Issues

### High Latency

**Symptoms**: Requests taking longer than expected.

**Causes**:

1. Backend slow
2. Network issues
3. Connection pooling exhausted

**Debugging**:

```bash
# Check latency metrics
curl -s http://localhost:9090/metrics | grep duration

# Compare with direct backend request
time curl -s http://backend:3000/api/test
time curl -s http://localhost:8080/api/test
```

**Solution**:

1. Investigate backend performance
2. Check network between Relaypoint and backends
3. Tune timeouts:

```yaml
server:
  read_timeout: 60s
  write_timeout: 60s
```

### High Memory Usage

**Symptoms**: Relaypoint memory growing over time.

**Causes**:

1. Many unique rate limit keys (IPs)
2. Memory leak (please report!)

**Solution**:

```yaml
rate_limit:
  cleanup_interval: 1m # More frequent cleanup
```

### Connection Errors

**Symptoms**: Intermittent connection failures.

**Causes**:

1. Backend connection limits
2. Too many idle connections

**Solution**: Currently connection pool settings are built-in. Consider scaling backends or using load balancer in front of Relaypoint.

## Logging and Debugging

### Enable Debug Information

Check logs for detailed information:

```bash
# Run in foreground to see logs
./relaypoint -config config.yml 2>&1 | tee relaypoint.log

# Parse JSON logs
./relaypoint -config config.yml 2>&1 | jq -r '.msg'
```

### Common Log Messages

| Message                | Meaning                      |
| ---------------------- | ---------------------------- |
| `Starting RelayPoint`  | Gateway starting             |
| `configuration loaded` | Config parsed successfully   |
| `upstream unhealthy`   | Backend failed health check  |
| `server error`         | Fatal error, gateway stopped |

## Getting Help

If you can't resolve an issue:

1. **Search existing issues**: [GitHub Issues](https://github.com/relaypoint/relaypoint/issues)

2. **Gather information**:

   ```bash
   # Version
   ./relaypoint -version

   # Configuration (remove secrets)
   cat config.yml

   # Metrics
   curl -s http://localhost:9090/metrics > metrics.txt

   # Logs
   ./relaypoint -config config.yml 2>&1 | head -100
   ```

3. **Open an issue** with:
   - Relaypoint version
   - Configuration (sanitized)
   - Steps to reproduce
   - Expected vs actual behavior
   - Relevant logs/metrics

## Next Steps

- [Best Practices](./best-practices.md) - Prevent issues before they happen
- [Metrics](./features/metrics.md) - Set up monitoring
- [Configuration](./configuration.md) - Review all options
