# trap-webhook

[![Build](https://github.com/probitas-test/state-servers/actions/workflows/build.trap-webhook.yml/badge.svg)](https://github.com/probitas-test/state-servers/actions/workflows/build.trap-webhook.yml)
[![Docker](https://github.com/probitas-test/state-servers/actions/workflows/docker.trap-webhook.yml/badge.svg)](https://github.com/probitas-test/state-servers/actions/workflows/docker.trap-webhook.yml)

Webhook trap server that captures and stores received HTTP requests in-memory,
providing a web interface for viewing them.

## Image

```
ghcr.io/probitas-test/trap-webhook:latest
```

## URLs

| Environment    | URL                      |
| -------------- | ------------------------ |
| Container      | `http://localhost:8080`  |
| Docker Compose | `http://localhost:18090` |

## Features

- **Webhook Receiver** - Accepts requests on any `/webhook/*` path
- **Web UI** - Built-in web interface for viewing captured requests
- **REST API** - JSON API for programmatic access
- **Server-Sent Events** - Real-time updates when new requests arrive
- **Request Details** - Captures headers, body, query params, and path
- **Configurable Storage** - TTL and max entries limit

## Quick Start

```bash
# Run with Docker
docker run -p 8080:8080 ghcr.io/probitas-test/trap-webhook

# Or run locally
just run
```

## API Endpoints

| Endpoint            | Method | Description                      |
| ------------------- | ------ | -------------------------------- |
| `/`                 | GET    | Web UI                           |
| `/doc`              | GET    | API documentation (Markdown)     |
| `/webhook/*`        | ANY    | Accept request on any sub-path   |
| `/api/entries`      | GET    | List all requests (newest first) |
| `/api/entries/{id}` | GET    | Get specific request             |
| `/api/entries/{id}` | DELETE | Delete specific request          |
| `/api/entries`      | DELETE | Clear all requests               |
| `/api/stats`        | GET    | Store statistics                 |
| `/api/events`       | GET    | Server-Sent Events stream        |
| `/health`           | GET    | Health check                     |

### Filtering

`GET /api/entries` supports query parameters for filtering:

| Parameter        | Description                       |
| ---------------- | --------------------------------- |
| `method`         | Exact match on HTTP method        |
| `path`           | Contains match on request path    |
| `query`          | Contains match on query string    |
| `body`           | Contains match on request body    |
| `jsonpath`       | JSONPath expression for JSON body |
| `jsonpath_value` | Expected value at JSONPath        |
| `content_type`   | Contains match on Content-Type    |
| `header`         | Header name to check              |
| `header_value`   | Header value contains match       |
| `host`           | Contains match on Host header     |
| `since`          | ReceivedAt after (RFC3339)        |
| `until`          | ReceivedAt before (RFC3339)       |
| `limit`          | Maximum number of results         |
| `offset`         | Skip first N results              |

See [API Documentation](/doc) for detailed examples.

## Configuration

| Variable      | Description             | Default   |
| ------------- | ----------------------- | --------- |
| `HOST`        | Bind address            | `0.0.0.0` |
| `PORT`        | HTTP API port           | `8080`    |
| `MAX_ENTRIES` | Maximum entries to keep | `1000`    |
| `ENTRY_TTL`   | Entry TTL in seconds    | `3600`    |

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
