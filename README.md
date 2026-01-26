# Trap Servers

[![Build trap-smtp](https://github.com/probitas-test/trap-servers/actions/workflows/build.trap-smtp.yml/badge.svg)](https://github.com/probitas-test/trap-servers/actions/workflows/build.trap-smtp.yml)
[![Build trap-webhook](https://github.com/probitas-test/trap-servers/actions/workflows/build.trap-webhook.yml/badge.svg)](https://github.com/probitas-test/trap-servers/actions/workflows/build.trap-webhook.yml)

Trap servers for capturing and inspecting SMTP emails and HTTP requests. Built
for testing [Probitas](https://github.com/probitas-test/probitas) and other
client implementations.

## Images

| Image                                | Protocol | Default Port             | Status                                                                                                                                                                                                  |
| ------------------------------------ | -------- | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ghcr.io/probitas-test/trap-smtp`    | SMTP     | 8080 (Web) / 2525 (SMTP) | [![Docker](https://github.com/probitas-test/trap-servers/actions/workflows/docker.trap-smtp.yml/badge.svg)](https://github.com/probitas-test/trap-servers/actions/workflows/docker.trap-smtp.yml)       |
| `ghcr.io/probitas-test/trap-webhook` | HTTP     | 8080                     | [![Docker](https://github.com/probitas-test/trap-servers/actions/workflows/docker.trap-webhook.yml/badge.svg)](https://github.com/probitas-test/trap-servers/actions/workflows/docker.trap-webhook.yml) |

## URLs (Docker Compose)

| Server       | Web UI / API             | Protocol Endpoint |
| ------------ | ------------------------ | ----------------- |
| trap-smtp    | `http://localhost:18091` | `localhost:12525` |
| trap-webhook | `http://localhost:18090` | (same as Web UI)  |

## Quick Start

```bash
# Start all servers
docker compose up -d

# Test trap-webhook: Send a request
curl -X POST http://localhost:18090/webhook/test \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'

# View captured requests
curl http://localhost:18090/api/entries

# Test trap-smtp: Send an email
echo -e "Subject: Test\n\nHello World" | \
  curl smtp://localhost:12525 \
    --mail-from sender@example.com \
    --mail-rcpt recipient@example.com \
    --upload-file -

# View captured emails
curl http://localhost:18091/api/entries

# Stop all servers
docker compose down
```

## Environment Variables

### trap-smtp

| Variable         | Description                     | Default     |
| ---------------- | ------------------------------- | ----------- |
| `HOST`           | Bind address                    | `0.0.0.0`   |
| `WEB_PORT`       | HTTP API port                   | `8080`      |
| `SMTP_PORT`      | SMTP server port                | `2525`      |
| `MAX_ENTRIES`    | Maximum entries to keep         | `1000`      |
| `ENTRY_TTL`      | Entry TTL in seconds            | `3600`      |
| `SMTP_DOMAIN`    | SMTP domain name                | `localhost` |
| `SMTP_MAX_SIZE`  | Maximum message size in bytes   | `10485760`  |
| `ALLOW_INSECURE` | Allow insecure SMTP connections | `true`      |

### trap-webhook

| Variable      | Description             | Default   |
| ------------- | ----------------------- | --------- |
| `HOST`        | Bind address            | `0.0.0.0` |
| `PORT`        | HTTP API port           | `8080`    |
| `MAX_ENTRIES` | Maximum entries to keep | `1000`    |
| `ENTRY_TTL`   | Entry TTL in seconds    | `3600`    |

Servers also support `.env` file for configuration.

## Features

All servers are designed for testing purposes:

- **Stateful storage** - Captured entries are stored in-memory with configurable TTL
- **Web UI** - Built-in web interface for viewing captured entries
- **REST API** - JSON API for programmatic access
- **Server-Sent Events** - Real-time updates via SSE
- **No rate limits** - Test high-throughput scenarios
- **Minimal images** - Built on scratch, ~10-20MB each

## API Endpoints

Both servers provide the following endpoints:

| Endpoint            | Method | Description                     |
| ------------------- | ------ | ------------------------------- |
| `/`                 | GET    | Web UI                          |
| `/doc`              | GET    | API documentation (Markdown)    |
| `/api/entries`      | GET    | List all entries (newest first) |
| `/api/entries/{id}` | GET    | Get specific entry              |
| `/api/entries/{id}` | DELETE | Delete specific entry           |
| `/api/entries`      | DELETE | Clear all entries               |
| `/api/stats`        | GET    | Store statistics                |
| `/api/events`       | GET    | Server-Sent Events stream       |
| `/health`           | GET    | Health check                    |

### Filtering

`GET /api/entries` supports query parameters for filtering:

| Parameter        | Description                     | trap-smtp | trap-webhook |
| ---------------- | ------------------------------- | --------- | ------------ |
| `from`           | Contains match on sender        | ✓         |              |
| `to`             | Contains match on any recipient | ✓         |              |
| `subject`        | Contains match on subject       | ✓         |              |
| `method`         | Exact match on HTTP method      |           | ✓            |
| `path`           | Contains match on request path  |           | ✓            |
| `query`          | Contains match on query string  |           | ✓            |
| `body`           | Contains match on body          | ✓         | ✓            |
| `jsonpath`       | JSONPath expression for body    | ✓         | ✓            |
| `jsonpath_value` | Expected value at JSONPath      | ✓         | ✓            |
| `content_type`   | Contains match on Content-Type  |           | ✓            |
| `header`         | Header name to check            | ✓         | ✓            |
| `header_value`   | Header value contains match     | ✓         | ✓            |
| `host`           | Contains match on Host header   |           | ✓            |
| `since`          | ReceivedAt after (RFC3339)      | ✓         | ✓            |
| `until`          | ReceivedAt before (RFC3339)     | ✓         | ✓            |
| `limit`          | Maximum number of results       | ✓         | ✓            |
| `offset`         | Skip first N results            | ✓         | ✓            |

### trap-smtp specific

| Endpoint                | Method | Description             |
| ----------------------- | ------ | ----------------------- |
| `/api/entries/{id}/raw` | GET    | Get raw email (RFC 822) |

### trap-webhook specific

| Endpoint     | Method | Description                    |
| ------------ | ------ | ------------------------------ |
| `/webhook/*` | ANY    | Accept request on any sub-path |

## Documentation

- [trap-smtp](./trap-smtp/README.md) - SMTP trap server
- [trap-webhook](./trap-webhook/README.md) - Webhook trap server

## Development

### Prerequisites

Uses [Nix Flakes](https://nixos.wiki/wiki/Flakes) for development environment
management.

```bash
# Enter development environment
nix develop

# Or use direnv for automatic activation
echo "use flake" > .envrc
direnv allow
```

### Commands

Requires [just](https://github.com/casey/just) command runner (included in Nix
environment).

```bash
# Show available commands
just

# Run linter on all packages
just lint

# Run tests on all packages
just test

# Build all packages
just build

# Format all code (Go + Markdown/JSON/YAML)
just fmt

# Build and run Docker images
docker compose up --build
```
