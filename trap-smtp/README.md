# trap-smtp

[![Build](https://github.com/probitas-test/state-servers/actions/workflows/build.trap-smtp.yml/badge.svg)](https://github.com/probitas-test/state-servers/actions/workflows/build.trap-smtp.yml)
[![Docker](https://github.com/probitas-test/state-servers/actions/workflows/docker.trap-smtp.yml/badge.svg)](https://github.com/probitas-test/state-servers/actions/workflows/docker.trap-smtp.yml)

SMTP trap server that captures and stores received emails in-memory, providing a
web interface for viewing them.

## Image

```
ghcr.io/probitas-test/trap-smtp:latest
```

## URLs

| Environment    | Web UI                   | SMTP              |
| -------------- | ------------------------ | ----------------- |
| Container      | `http://localhost:8080`  | `localhost:2525`  |
| Docker Compose | `http://localhost:18091` | `localhost:12525` |

## Features

- **SMTP Server** - Receives emails on configurable port (default: 2525)
- **SMTP Authentication** - Optional PLAIN/LOGIN authentication support
- **Web UI** - Built-in web interface for viewing captured emails
- **REST API** - JSON API for programmatic access
- **Server-Sent Events** - Real-time updates when new emails arrive
- **Raw Email Access** - View original RFC 822 format
- **Configurable Storage** - TTL and max entries limit

## Quick Start

```bash
# Run with Docker
docker run -p 8080:8080 -p 2525:2525 ghcr.io/probitas-test/trap-smtp

# Or run locally
just run
```

## API Endpoints

| Endpoint                | Method | Description                    |
| ----------------------- | ------ | ------------------------------ |
| `/`                     | GET    | Web UI                         |
| `/doc`                  | GET    | API documentation (Markdown)   |
| `/api/entries`          | GET    | List all emails (newest first) |
| `/api/entries/{id}`     | GET    | Get specific email             |
| `/api/entries/{id}/raw` | GET    | Get raw email (RFC 822 format) |
| `/api/entries/{id}`     | DELETE | Delete specific email          |
| `/api/entries`          | DELETE | Clear all emails               |
| `/api/stats`            | GET    | Store statistics               |
| `/api/events`           | GET    | Server-Sent Events stream      |
| `/health`               | GET    | Health check                   |

### Filtering

`GET /api/entries` supports query parameters for filtering:

| Parameter        | Description                     |
| ---------------- | ------------------------------- |
| `from`           | Contains match on sender        |
| `to`             | Contains match on any recipient |
| `subject`        | Contains match on subject       |
| `body`           | Contains match on body          |
| `jsonpath`       | JSONPath expression for body    |
| `jsonpath_value` | Expected value at JSONPath      |
| `header`         | Header name to check            |
| `header_value`   | Header value contains match     |
| `since`          | ReceivedAt after (RFC3339)      |
| `until`          | ReceivedAt before (RFC3339)     |
| `limit`          | Maximum number of results       |
| `offset`         | Skip first N results            |

See [API Documentation](/doc) for detailed examples.

## Configuration

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
| `USERNAME`       | SMTP authentication username    | (empty)     |
| `PASSWORD`       | SMTP authentication password    | (empty)     |

## Authentication

The server supports SMTP authentication with **PLAIN** and **LOGIN** mechanisms.

### Open Relay Mode (Default)

When `USERNAME` and `PASSWORD` are both empty, the server runs in open relay
mode:

- Authentication mechanisms are advertised (PLAIN, LOGIN)
- Any credentials sent by clients are accepted
- Clients that don't authenticate can also send emails

This is the default behavior, suitable for testing environments where you want
to capture all emails regardless of authentication.

### Authentication Required Mode

When both `USERNAME` and `PASSWORD` are set, the server requires valid
credentials:

```bash
# Run with authentication
docker run -p 8080:8080 -p 2525:2525 \
  -e USERNAME=testuser \
  -e PASSWORD=testpass \
  ghcr.io/probitas-test/trap-smtp
```

Clients must authenticate with the configured credentials to send emails.

## Development

```bash
# Run linter
just lint

# Run tests
just test

# Build binary
just build

# Run locally
just run
```
