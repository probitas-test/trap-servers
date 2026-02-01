# trap-webhook API Reference

## Base URL

| Environment    | URL                      |
| -------------- | ------------------------ |
| Container      | `http://localhost:8080`  |
| Docker Compose | `http://localhost:18090` |

> **Note:** The container listens on port 8080. When using `docker compose up`,
> the port is mapped to 18090 on the host.

## Environment Variables

### Server Configuration

| Variable      | Default   | Description                             |
| ------------- | --------- | --------------------------------------- |
| `HOST`        | `0.0.0.0` | Bind address                            |
| `PORT`        | `8080`    | Listen port                             |
| `MAX_ENTRIES` | `1000`    | Maximum entries to keep (0 = unlimited) |
| `ENTRY_TTL`   | `3600`    | Entry TTL in seconds (0 = never expire) |

---

## Endpoints

### Webhook Receiver

#### ANY /webhook

#### ANY /webhook/*

Accept any HTTP request and store it. Supports all HTTP methods (GET, POST, PUT,
PATCH, DELETE, etc.) and any path under `/webhook/`.

**Request:**

```bash
curl -X POST "http://localhost:8080/webhook/my-endpoint" \
  -H "Content-Type: application/json" \
  -d '{"event": "user.created", "data": {"id": 123}}'
```

**Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "received"
}
```

**Response Headers:**

| Header         | Description                   |
| -------------- | ----------------------------- |
| `X-Webhook-ID` | Unique ID of the stored entry |

---

### API Endpoints

### GET /api/entries

List all stored webhook entries with optional filtering.

**Query Parameters:**

| Parameter            | Type   | Description                                   |
| -------------------- | ------ | --------------------------------------------- |
| `method`             | string | Exact match on HTTP method (case-insensitive) |
| `path`               | string | Contains match on request path                |
| `query`              | string | Contains match on query string                |
| `body`               | string | Contains match on request body                |
| `jsonpath`           | string | JSONPath expression for JSON body             |
| `jsonpath_value`     | string | Expected value at JSONPath                    |
| `content_type`       | string | Contains match on Content-Type                |
| `header`             | string | Header name to check                          |
| `header_value`       | string | Header value contains match                   |
| `host`               | string | Contains match on Host header                 |
| `path_regex`         | string | Regex match on request path                   |
| `query_regex`        | string | Regex match on query string                   |
| `body_regex`         | string | Regex match on request body                   |
| `content_type_regex` | string | Regex match on Content-Type                   |
| `host_regex`         | string | Regex match on Host header                    |
| `since`              | string | ReceivedAt after (RFC3339 format)             |
| `until`              | string | ReceivedAt before (RFC3339 format)            |
| `limit`              | int    | Maximum number of results                     |
| `offset`             | int    | Skip first N results                          |

> **Note:** All filters use AND logic. Empty filters return all entries
> (backward compatible). Regex filters use Go's `regexp` syntax. Invalid regex
> returns 400 Bad Request.

**Request:**

```bash
curl "http://localhost:8080/api/entries"
```

**Response:**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "received_at": "2025-01-27T10:30:00Z",
    "method": "POST",
    "path": "/webhook/my-endpoint",
    "query_string": "token=abc123",
    "headers": {
      "Content-Type": ["application/json"],
      "User-Agent": ["curl/8.0.0"]
    },
    "body": "{\"event\": \"user.created\", \"data\": {\"id\": 123}}",
    "content_type": "application/json",
    "remote_addr": "172.17.0.1:54321",
    "host": "localhost:8080"
  }
]
```

**Filter Examples:**

```bash
# Filter by HTTP method
curl "http://localhost:8080/api/entries?method=POST"

# Filter by path
curl "http://localhost:8080/api/entries?path=/webhook/users"

# Filter by query string
curl "http://localhost:8080/api/entries?query=token"

# Filter by JSONPath (for JSON bodies)
curl "http://localhost:8080/api/entries?jsonpath=$.event&jsonpath_value=user.created"

# Filter by nested JSONPath
curl "http://localhost:8080/api/entries?jsonpath=$.data.id&jsonpath_value=123"

# Filter by Content-Type
curl "http://localhost:8080/api/entries?content_type=json"

# Filter by custom header
curl "http://localhost:8080/api/entries?header=Authorization&header_value=Bearer"

# Filter by host
curl "http://localhost:8080/api/entries?host=api.example.com"

# Filter by time range
curl "http://localhost:8080/api/entries?since=2025-01-01T00:00:00Z&until=2025-01-31T23:59:59Z"

# Pagination
curl "http://localhost:8080/api/entries?limit=10&offset=0"

# Combined filters (AND logic)
curl "http://localhost:8080/api/entries?method=POST&path=/webhook&jsonpath=$.event&limit=5"
```

### GET /api/entries/{id}

Get a specific webhook entry by ID.

**Request:**

```bash
curl "http://localhost:8080/api/entries/550e8400-e29b-41d4-a716-446655440000"
```

**Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "received_at": "2025-01-27T10:30:00Z",
  "method": "POST",
  "path": "/webhook/my-endpoint",
  "query_string": "token=abc123",
  "headers": {
    "Content-Type": ["application/json"],
    "User-Agent": ["curl/8.0.0"]
  },
  "body": "{\"event\": \"user.created\", \"data\": {\"id\": 123}}",
  "content_type": "application/json",
  "remote_addr": "172.17.0.1:54321",
  "host": "localhost:8080"
}
```

**Error Response (404):**

```
Entry not found
```

### DELETE /api/entries/{id}

Delete a specific webhook entry.

**Request:**

```bash
curl -X DELETE "http://localhost:8080/api/entries/550e8400-e29b-41d4-a716-446655440000"
```

**Response:**

```json
{
  "status": "deleted"
}
```

### DELETE /api/entries

Clear all stored webhook entries.

**Request:**

```bash
curl -X DELETE "http://localhost:8080/api/entries"
```

**Response:**

```json
{
  "status": "cleared",
  "deleted": 42
}
```

### GET /api/stats

Get storage statistics.

**Request:**

```bash
curl "http://localhost:8080/api/stats"
```

**Response:**

```json
{
  "count": 42
}
```

### GET /api/count

Get the number of entries matching the filter criteria. Useful for quick
assertions without fetching full entry data.

**Query Parameters:**

Same filter parameters as `GET /api/entries` (`method`, `path`, `query`, `body`,
`jsonpath`, `jsonpath_value`, `content_type`, `header`, `header_value`, `host`,
`path_regex`, `query_regex`, `body_regex`, `content_type_regex`, `host_regex`,
`since`, `until`). Pagination parameters (`limit`, `offset`) are ignored.

**Request:**

```bash
# Count all entries
curl "http://localhost:8080/api/count"

# Count POST requests
curl "http://localhost:8080/api/count?method=POST"

# Count with multiple filters
curl "http://localhost:8080/api/count?method=POST&path=payment"
```

**Response:**

```json
{
  "count": 3
}
```

### GET /api/await

Block until the specified number of entries match the filter criteria, or the
timeout is reached. This eliminates the need for polling or `sleep` in tests.

**Query Parameters:**

| Parameter | Type   | Default | Description                                  |
| --------- | ------ | ------- | -------------------------------------------- |
| `count`   | int    | `1`     | Minimum number of matching entries to return |
| `timeout` | string | `10s`   | Maximum wait duration (Go duration format)   |

All webhook filter parameters (`method`, `path`, `query`, `body`, `jsonpath`,
`jsonpath_value`, `content_type`, `header`, `header_value`, `host`, `since`,
`until`, `path_regex`, `query_regex`, `body_regex`, `content_type_regex`,
`host_regex`) are also supported. Pagination parameters (`limit`, `offset`) are
not supported.

**Request:**

```bash
# Wait for 1 POST webhook (up to 5 seconds)
curl "http://localhost:8080/api/await?method=POST&timeout=5s"

# Wait for 2 webhooks to a specific path
curl "http://localhost:8080/api/await?path=/webhook/payment&count=2&timeout=10s"
```

**Success Response (200):**

Returns all matching entries when the count threshold is met.

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "received_at": "2025-01-27T10:30:00Z",
    "method": "POST",
    "path": "/webhook/payment",
    "body": "{\"event\": \"payment.completed\"}"
  }
]
```

**Timeout Response (408):**

```json
{
  "error": "timeout",
  "matched": 0,
  "expected": 1
}
```

**Test Example (using curl in a CI pipeline):**

```bash
# 1. Trigger your app (which sends a webhook to trap-webhook)
curl -X POST http://my-app/process-order

# 2. Wait for the webhook to arrive and verify
curl -sf "http://localhost:8080/api/await?path=/webhook/payment&timeout=5s" | \
  jq '.[0].body | fromjson | .event'
# Output: "payment.completed"
```

### GET /api/events

Server-Sent Events (SSE) stream for real-time webhook notifications.

**Request:**

```bash
curl -N "http://localhost:8080/api/events"
```

**Response (SSE stream):**

```
data: {"id":"550e8400-e29b-41d4-a716-446655440000","received_at":"2025-01-27T10:30:00Z",...}

data: {"id":"660e8400-e29b-41d4-a716-446655440001","received_at":"2025-01-27T10:31:00Z",...}
```

### GET /health

Health check endpoint.

**Request:**

```bash
curl "http://localhost:8080/health"
```

**Response:**

```json
{
  "status": "ok"
}
```

---

## Web UI

### GET /

Access the web-based webhook viewer.

**URL:** `http://localhost:8080/`

The web UI provides:

- Real-time webhook list with auto-refresh via SSE
- Request detail view with headers and body
- JSON body formatting
- Delete individual entries or clear all

---

## Data Types

### WebhookEntry

| Field          | Type                | Description                     |
| -------------- | ------------------- | ------------------------------- |
| `id`           | string              | Unique identifier (UUID)        |
| `seq`          | int64               | Monotonically increasing seq#   |
| `received_at`  | string (RFC3339)    | Timestamp when request received |
| `method`       | string              | HTTP method (GET, POST, etc.)   |
| `path`         | string              | Request path                    |
| `query_string` | string              | Query string (without ?)        |
| `headers`      | map[string][]string | Request headers                 |
| `body`         | string              | Request body content            |
| `content_type` | string              | Content-Type header value       |
| `remote_addr`  | string              | Client IP address and port      |
| `host`         | string              | Host header value               |

---

## Usage Examples

### Testing Webhook Integration

```bash
# Send a simple POST request
curl -X POST "http://localhost:8080/webhook/test" \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'

# Send with custom headers
curl -X POST "http://localhost:8080/webhook/github" \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=abc123" \
  -d '{"ref": "refs/heads/main", "commits": []}'

# Send form data
curl -X POST "http://localhost:8080/webhook/form" \
  -d "name=test" \
  -d "email=test@example.com"

# Query string parameters
curl "http://localhost:8080/webhook/callback?code=abc123&state=xyz"
```

### Verifying Received Webhooks

```bash
# List all webhooks
curl "http://localhost:8080/api/entries"

# Find specific webhook by JSONPath
curl "http://localhost:8080/api/entries?jsonpath=$.ref&jsonpath_value=refs/heads/main"

# Find webhooks from GitHub
curl "http://localhost:8080/api/entries?header=X-GitHub-Event"

# Find POST requests to specific path
curl "http://localhost:8080/api/entries?method=POST&path=/webhook/github"
```

### Real-time Monitoring

```bash
# Watch for new webhooks in terminal
curl -N "http://localhost:8080/api/events" | while read line; do
  echo "$line" | jq -r 'select(.id) | "\(.method) \(.path) - \(.id)"'
done
```
