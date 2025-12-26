# Relaypoint {.hidden}

<p align="center">
    <img src="./assets/relaypoint-wordmark-light.svg#only-light">
    <img src="./assets/relaypoint-wordmark_dark.svg#only-dark">
</p>

Welcome to the Relaypoint documentation. Relaypoint is a lightweight, high-performance API gateway designed to help startups secure and manage their APIs with minimal configuration.

## Table of Contents

- [Getting Started](./getting-started.md)
- [Installation](./installation.md)
- [Configuration Reference](./configuration.md)
- [Features](./features/index.md)
  - [Routing](./features/routing.md)
  - [Load Balancing](./features/load-balancing.md)
  - [Rate Limiting](./features/rate-limiting.md)
  - [Health Checks](./features/health-checks.md)
  - [Metrics & Monitoring](./features/metrics.md)
  - [API Keys](./features/api-keys.md)
- [Examples](./examples.md)
- [Troubleshooting](./troubleshooting.md)
- [Best Practices](./best-practices.md)

## Why Relaypoint?

Relaypoint provides enterprise-grade API gateway features without the complexity:

- **Simple YAML Configuration** - Get started in minutes, not days
- **Intelligent Load Balancing** - Round-robin, least connections, weighted, and random strategies
- **Flexible Rate Limiting** - Per-route, per-IP, and per-API-key rate limiting
- **Health Checks** - Automatic backend health monitoring with circuit breaking
- **Prometheus Metrics** - Built-in observability with detailed request metrics
- **Zero Dependencies** - Single binary deployment with no external dependencies

## Quick Example

```yaml
# relaypoint.yml
server:
  port: 8080

upstreams:
  - name: api-backend
    targets:
      - url: http://localhost:3001
      - url: http://localhost:3002
    load_balance: round_robin

routes:
  - name: api
    path: /api/**
    upstream: api-backend
```

```bash
./relaypoint -config relaypoint.yml
```

That's it! Relaypoint is now load balancing requests across your backend servers.

## Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/relaypoint/relaypoint/issues)
- **Discussions**: [Community Q&A](https://github.com/relaypoint/relaypoint/discussions)

## License

Relaypoint is licensed under the [Apache License 2.0](../LICENSE).
