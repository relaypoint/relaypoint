# API Keys

Relaypoint supports API key authentication for identifying clients and applying per-key rate limits.

## Overview

API keys in Relaypoint provide:

- **Client identification** - Track which application made a request
- **Per-key rate limiting** - Different limits for different clients
- **Usage tracking** - Metrics per API key
- **Access control** - Enable/disable keys instantly

## Basic Configuration

```yaml
api_keys:
  - key: "pk_live_abc123def456"
    name: "production-app"
    requests_per_second: 1000
    burst_size: 2000
    enabled: true
```

## Configuration Options

| Field                 | Type    | Required | Description                           |
| --------------------- | ------- | -------- | ------------------------------------- |
| `key`                 | string  | Yes      | The API key value (keep secret!)      |
| `name`                | string  | Yes      | Human-readable identifier             |
| `requests_per_second` | integer | Yes      | Rate limit for this key               |
| `burst_size`          | integer | No       | Burst capacity (default: 2x RPS)      |
| `enabled`             | boolean | No       | Whether key is active (default: true) |

## How API Keys Work

```
┌──────────────────────────────────────────────────────┐
│                     Request                           │
│    Authorization: Bearer pk_live_abc123               │
└──────────────────────────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────┐
│                   Relaypoint                          │
│                                                       │
│  1. Extract API key from request                      │
│  2. Look up key in configuration                      │
│  3. Check if key is enabled                          │
│  4. Apply key-specific rate limit                     │
│  5. Record metrics with key name                      │
│  6. Forward request to backend                        │
└──────────────────────────────────────────────────────┘
```

## Providing API Keys

Clients can provide API keys in multiple ways (checked in order):

### 1. Authorization Header (Recommended)

```bash
# Bearer token format
curl -H "Authorization: Bearer pk_live_abc123" https://api.example.com/users

# ApiKey format
curl -H "Authorization: ApiKey pk_live_abc123" https://api.example.com/users
```

### 2. X-API-Key Header

```bash
curl -H "X-API-Key: pk_live_abc123" https://api.example.com/users
```

### 3. Query Parameter (Least Secure)

```bash
curl "https://api.example.com/users?api_key=pk_live_abc123"
```

> ⚠️ **Warning**: Query parameters appear in logs and browser history. Use headers for production.

## API Key Patterns

### Tiered Access

```yaml
api_keys:
  # Free tier
  - key: "pk_free_user123"
    name: "free-tier-user"
    requests_per_second: 60
    burst_size: 100
    enabled: true

  # Pro tier
  - key: "pk_pro_user456"
    name: "pro-tier-user"
    requests_per_second: 600
    burst_size: 1000
    enabled: true

  # Enterprise tier
  - key: "pk_ent_user789"
    name: "enterprise-user"
    requests_per_second: 6000
    burst_size: 10000
    enabled: true
```

### Development vs Production

```yaml
api_keys:
  # Production keys - higher limits
  - key: "pk_live_webapp"
    name: "webapp-production"
    requests_per_second: 5000
    burst_size: 10000
    enabled: true

  # Development/staging keys - lower limits
  - key: "pk_test_webapp"
    name: "webapp-development"
    requests_per_second: 100
    burst_size: 200
    enabled: true
```

### Service-to-Service

```yaml
api_keys:
  # Internal service keys - high limits
  - key: "sk_internal_orders"
    name: "orders-service"
    requests_per_second: 50000
    burst_size: 100000
    enabled: true

  - key: "sk_internal_payments"
    name: "payments-service"
    requests_per_second: 50000
    burst_size: 100000
    enabled: true
```

### Partner Integrations

```yaml
api_keys:
  - key: "pk_partner_acme"
    name: "acme-corp-integration"
    requests_per_second: 2000
    burst_size: 4000
    enabled: true

  - key: "pk_partner_bigco"
    name: "bigco-integration"
    requests_per_second: 5000
    burst_size: 10000
    enabled: true
```

## Key Naming Conventions

### Recommended Prefixes

| Prefix     | Use Case                    |
| ---------- | --------------------------- |
| `pk_live_` | Production publishable keys |
| `pk_test_` | Test/development keys       |
| `sk_live_` | Production secret keys      |
| `sk_test_` | Test secret keys            |

### Example

```yaml
api_keys:
  - key: "pk_live_user_abc123def456789"
    name: "acme-corp-production"
    # ...

  - key: "pk_test_user_xyz789abc123456"
    name: "acme-corp-development"
    # ...
```

## Enabling Rate Limiting for API Keys

Enable per-API-key rate limiting globally:

```yaml
rate_limit:
  enabled: true
  per_api_key: true # Required for API key rate limiting

api_keys:
  - key: "pk_live_abc123"
    name: "my-app"
    requests_per_second: 1000
    burst_size: 2000
    enabled: true
```

## Disabling API Keys

Instantly disable a key:

```yaml
api_keys:
  - key: "pk_live_compromised_key"
    name: "compromised-client"
    requests_per_second: 1000
    enabled: false # Key is disabled
```

Disabled keys:

- Are not checked for rate limits
- Do not appear in metrics
- Requests are still processed (unless other limits block them)

> **Note**: Disabling a key doesn't block requests entirely. For blocking, implement authentication middleware or use a firewall.

## Monitoring API Key Usage

### Metrics

```bash
curl http://localhost:9090/metrics | grep api_key
```

Output:

```
gateway_api_key_requests_total{key="production-app_200"} 15234
gateway_api_key_requests_total{key="production-app_429"} 45
gateway_api_key_requests_total{key="development-app_200"} 1234
```

### Stats Endpoint

```bash
curl http://localhost:8080/stats | jq '.[] | select(.key | contains("apikey"))'
```

### Grafana Queries

```promql
# Requests per API key
sum by (key) (rate(gateway_api_key_requests_total[5m]))

# Top 10 API keys by request volume
topk(10, sum by (key) (rate(gateway_api_key_requests_total[5m])))

# Rate limit hits per API key
sum by (key) (rate(gateway_rate_limit_hits_total{key=~".*_apikey"}[5m]))
```

## Security Best Practices

### 1. Keep Keys Secret

```yaml
# Bad: Key in plain text in version control
api_keys:
  - key: "pk_live_abc123"
    name: "my-app"

# Better: Use environment variables
# Process config with envsubst before loading
api_keys:
  - key: "${API_KEY_MYAPP}"
    name: "my-app"
```

### 2. Use Strong Key Values

```python
# Generate secure keys
import secrets
key = f"pk_live_{secrets.token_urlsafe(32)}"
# Result: pk_live_dG9rZW5fdXJsc2FmZV8zMg...
```

```bash
# Or use openssl
openssl rand -base64 32 | tr -d '/+=' | head -c 32
```

### 3. Rotate Keys Regularly

```yaml
# During rotation, both keys are active
api_keys:
  - key: "pk_live_new_key_abc"
    name: "my-app-v2"
    enabled: true

  - key: "pk_live_old_key_xyz"
    name: "my-app-v1-deprecated"
    enabled: true # Disable after migration
```

### 4. Use Different Keys per Environment

```yaml
# Production config
api_keys:
  - key: "pk_live_production"
    name: "webapp-prod"
    enabled: true

# Staging config
api_keys:
  - key: "pk_test_staging"
    name: "webapp-staging"
    enabled: true
```

### 5. Log Key Names, Not Values

Relaypoint logs the key `name`, never the key `value`:

```json
{
  "level": "INFO",
  "msg": "request processed",
  "api_key_name": "production-app"
}
```

## Handling Unknown Keys

Requests with unrecognized API keys:

- Are still processed (not blocked)
- Use default rate limits
- Do not appear in API key metrics

To require API keys, implement authentication middleware in your backend or add a validation route.

## Example: Full Configuration

```yaml
server:
  port: 8080

rate_limit:
  enabled: true
  default_rps: 60 # Anonymous users
  default_burst: 120
  per_ip: true
  per_api_key: true

upstreams:
  - name: api
    targets:
      - url: http://backend:3000
    load_balance: round_robin

routes:
  - name: public-api
    path: /api/**
    upstream: api

api_keys:
  # Tier 1: Free
  - key: "pk_free_abc123"
    name: "free-tier"
    requests_per_second: 60
    burst_size: 100
    enabled: true

  # Tier 2: Starter
  - key: "pk_starter_def456"
    name: "starter-tier"
    requests_per_second: 300
    burst_size: 500
    enabled: true

  # Tier 3: Pro
  - key: "pk_pro_ghi789"
    name: "pro-tier"
    requests_per_second: 1000
    burst_size: 2000
    enabled: true

  # Tier 4: Enterprise
  - key: "pk_ent_jkl012"
    name: "enterprise-tier"
    requests_per_second: 10000
    burst_size: 20000
    enabled: true

  # Internal services
  - key: "sk_internal_service"
    name: "internal-service"
    requests_per_second: 100000
    burst_size: 200000
    enabled: true

  # Disabled key (compromised)
  - key: "pk_old_compromised"
    name: "compromised-key"
    requests_per_second: 0
    enabled: false
```

## Troubleshooting

### API Key Not Recognized

**Symptoms**: Requests not tracked with key name.

**Causes**:

1. Key not in configuration
2. Wrong key format in request
3. Typo in key value

**Solutions**:

1. Verify key is in config
2. Check request header format
3. Compare key values exactly

### Rate Limit Not Applied

**Symptoms**: API key exceeds its limit without 429 errors.

**Causes**:

1. `per_api_key: false` in rate limit config
2. Key has very high limits
3. Rate limiting disabled

**Solutions**:

1. Enable `per_api_key: true`
2. Check key's RPS configuration
3. Verify `rate_limit.enabled: true`

### Metrics Not Showing

**Symptoms**: API key requests not in metrics.

**Causes**:

1. Key not found in configuration
2. Metrics disabled

**Solutions**:

1. Add key to configuration
2. Enable metrics: `metrics.enabled: true`

## Next Steps

- [Rate Limiting](./rate-limiting.md) - How rate limits work
- [Metrics](./metrics.md) - Monitor API key usage
- [Best Practices](./best-practices.md) - Security recommendations
