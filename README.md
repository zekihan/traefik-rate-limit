# Traefik Rate Limit Plugin

A Traefik middleware plugin that rate limits incoming requests using the GCRA algorithm and Redis as a storage backend.

[![Traefik Plugin](https://img.shields.io/badge/Traefik%20Plugin-Traefik%20Rate%20Limit-blue)](https://plugins.traefik.io/plugins/67f2cf4768f0062a5d501e61/traefik-rate-limit)
[![Version](https://img.shields.io/badge/version-0.1.4-green)](https://github.com/zekihan/traefik-rate-limit/releases/tag/v0.1.4)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/zekihan/traefik-rate-limit/blob/main/LICENSE)

## Overview

Traefik Rate Limit is a middleware plugin for Traefik that limits the number of requests from a specific IP address within a defined time period. It utilizes the GCRA (Generic Cell Rate Algorithm) algorithm and Redis for storing rate limiting data, ensuring consistency across multiple Traefik instances.

## Features

- Rate limiting based on IP address
- Uses GCRA algorithm for precise rate limiting
- Redis backend for distributed rate limiting
- Configurable rate, burst, and period
- IP Whitelisting
- Support for resolving IP from headers (e.g., `X-Forwarded-For`)
- Local IP Whitelisting
- Configurable logging level

## Installation

### From Traefik Pilot

The easiest way to install this plugin is through the [Traefik Plugin Catalog](https://plugins.traefik.io/plugins/67f2cf4768f0062a5d501e61/traefik-rate-limit).

### Manual Installation

Add the plugin to your Traefik static configuration:

```yaml
experimental:
  plugins:
    traefik-rate-limit:
      moduleName: github.com/zekihan/traefik-rate-limit
      version: v0.1.4
```

## Configuration

### Static Configuration Example

```yaml
# Static configuration
experimental:
  plugins:
    traefik-rate-limit:
      moduleName: github.com/zekihan/traefik-rate-limit
      version: v0.1.4
```

### Middleware Configuration

```yaml
# Dynamic configuration
http:
  middlewares:
    rate-limit:
      plugin:
        traefik-rate-limit:
          redis:
            host: redis
            port: 6379
          rateLimit:
            rate: 100
            burst: 200
            period: 1m
          ipResolver:
            header: X-Real-IP
            useSrcIP: false
          whitelistedIPNets:
            - "127.0.0.1/32"
          whitelistLocalIPs: true
          logLevel: info
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `redis.host` | string | `localhost` | The hostname or IP address of your Redis server. |
| `redis.port` | int | `6379` | The port number of your Redis server. |
| `redis.prefix` | string | `traefik` | The prefix for the Redis keys. |
| `rateLimit.rate` | int | `100` | The number of requests allowed per `period`. |
| `rateLimit.burst` | int | `200` | The maximum number of requests that can be made in a short period of time. |
| `rateLimit.period` | string | `1m` | The time interval for the rate limit (e.g., `1s`, `1m`, `1h`). |
| `ipResolver.header` | string | `""` | The header to use to resolve the client IP address. If empty, the source IP is used. |
| `ipResolver.useSrcIP` | boolean | `true` | Whether to use the source IP address of the request. |
| `whitelistedIPNets` | array of strings | `[]` | A list of IP addresses or CIDR ranges that are not rate limited. |
| `whitelistLocalIPs` | boolean | `true` | Whether to whitelist local IP ranges. |
| `logLevel` | string | `info` | Log level (debug, info, warn, error) |

## How It Works

1.  The plugin resolves the client IP address using the configured `ipResolver`.
2.  It checks if the IP address is whitelisted.
3.  If not whitelisted, it uses the GCRA algorithm and Redis to check if the request is allowed.
4.  If the request is allowed, it is passed to the next middleware.
5.  If the request is not allowed, a `429 Too Many Requests` error is returned.

## Development

### Testing Locally

A Docker Compose setup is provided in the `testing` folder to test the plugin locally:

```bash
cd testing
docker-compose up -d
```

### Running Tests

```bash
go test ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/zekihan/traefik-rate-limit/blob/main/LICENSE) file for details.
