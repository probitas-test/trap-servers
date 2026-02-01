# trap-smtp API Reference

## Base URL

| Environment    | Web/API URL              | SMTP              |
| -------------- | ------------------------ | ----------------- |
| Container      | `http://localhost:8080`  | `localhost:2525`  |
| Docker Compose | `http://localhost:18091` | `localhost:12525` |

> **Note:** The container listens on port 8080 for the Web/API interface and port
> 2525 for SMTP. When using `docker compose up`, ports are mapped to 18091 (Web)
> and 12525 (SMTP) on the host.

## Environment Variables

### Server Configuration

| Variable    | Default   | Description             |
| ----------- | --------- | ----------------------- |
| `HOST`      | `0.0.0.0` | Bind address            |
| `WEB_PORT`  | `8080`    | HTTP API listen port    |
| `SMTP_PORT` | `2525`    | SMTP server listen port |

### Storage Configuration

| Variable      | Default | Description                             |
| ------------- | ------- | --------------------------------------- |
| `MAX_ENTRIES` | `1000`  | Maximum entries to keep (0 = unlimited) |
| `ENTRY_TTL`   | `3600`  | Entry TTL in seconds (0 = never expire) |

### SMTP Configuration

| Variable         | Default     | Description                          |
| ---------------- | ----------- | ------------------------------------ |
| `SMTP_DOMAIN`    | `localhost` | SMTP server domain                   |
| `SMTP_MAX_SIZE`  | `10485760`  | Maximum message size in bytes (10MB) |
| `ALLOW_INSECURE` | `true`      | Allow insecure authentication        |
| `USERNAME`       | (empty)     | SMTP authentication username         |
| `PASSWORD`       | (empty)     | SMTP authentication password         |

---

## Endpoints

### GET /api/entries

List all stored email entries with optional filtering.

**Query Parameters:**

| Parameter        | Type   | Description                        |
| ---------------- | ------ | ---------------------------------- |
| `from`           | string | Contains match on sender address   |
| `to`             | string | Contains match on any recipient    |
| `subject`        | string | Contains match on subject          |
| `body`           | string | Contains match on body             |
| `jsonpath`       | string | JSONPath expression for body       |
| `jsonpath_value` | string | Expected value at JSONPath         |
| `header`         | string | Header name to check               |
| `header_value`   | string | Header value contains match        |
| `since`          | string | ReceivedAt after (RFC3339 format)  |
| `until`          | string | ReceivedAt before (RFC3339 format) |
| `limit`          | int    | Maximum number of results          |
| `offset`         | int    | Skip first N results               |

> **Note:** All filters use AND logic. Empty filters return all entries
> (backward compatible).

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
    "from": "sender@example.com",
    "to": ["recipient@example.com"],
    "subject": "Test Email",
    "date": "Mon, 27 Jan 2025 10:30:00 +0000",
    "message_id": "<abc123@example.com>",
    "content_type": "text/plain",
    "headers": {
      "From": ["sender@example.com"],
      "To": ["recipient@example.com"],
      "Subject": ["Test Email"]
    },
    "body": "Hello, this is a test email.",
    "raw_email": "From: sender@example.com\r\nTo: recipient@example.com\r\n..."
  }
]
```

**Filter Examples:**

```bash
# Filter by sender
curl "http://localhost:8080/api/entries?from=test@example.com"

# Filter by subject containing "welcome"
curl "http://localhost:8080/api/entries?subject=welcome"

# Filter by JSONPath (for JSON email bodies)
curl "http://localhost:8080/api/entries?jsonpath=$.user.id&jsonpath_value=123"

# Filter by custom header
curl "http://localhost:8080/api/entries?header=X-Priority&header_value=high"

# Filter by time range
curl "http://localhost:8080/api/entries?since=2025-01-01T00:00:00Z&until=2025-01-31T23:59:59Z"

# Pagination
curl "http://localhost:8080/api/entries?limit=10&offset=0"

# Combined filters (AND logic)
curl "http://localhost:8080/api/entries?from=example.com&subject=hello&limit=5"
```

### GET /api/entries/{id}

Get a specific email entry by ID.

**Request:**

```bash
curl "http://localhost:8080/api/entries/550e8400-e29b-41d4-a716-446655440000"
```

**Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "received_at": "2025-01-27T10:30:00Z",
  "from": "sender@example.com",
  "to": ["recipient@example.com"],
  "subject": "Test Email",
  "date": "Mon, 27 Jan 2025 10:30:00 +0000",
  "message_id": "<abc123@example.com>",
  "content_type": "text/plain",
  "headers": {
    "From": ["sender@example.com"],
    "To": ["recipient@example.com"],
    "Subject": ["Test Email"]
  },
  "body": "Hello, this is a test email.",
  "raw_email": "From: sender@example.com\r\nTo: recipient@example.com\r\n..."
}
```

**Error Response (404):**

```
Entry not found
```

### GET /api/entries/{id}/raw

Download the raw email content as an .eml file.

**Request:**

```bash
curl "http://localhost:8080/api/entries/550e8400-e29b-41d4-a716-446655440000/raw"
```

**Response:**

```
Content-Type: message/rfc822
Content-Disposition: attachment; filename="email.eml"

From: sender@example.com
To: recipient@example.com
Subject: Test Email
Date: Mon, 27 Jan 2025 10:30:00 +0000

Hello, this is a test email.
```

### DELETE /api/entries/{id}

Delete a specific email entry.

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

Clear all stored email entries.

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

### GET /api/events

Server-Sent Events (SSE) stream for real-time email notifications.

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

Access the web-based email viewer.

**URL:** `http://localhost:8080/`

The web UI provides:

- Real-time email list with auto-refresh via SSE
- Email detail view with headers and body
- Raw email download
- Delete individual emails or clear all

---

## SMTP Server

The SMTP server accepts emails on port 2525 (default).

**SMTP Configuration:**

| Setting          | Value                     |
| ---------------- | ------------------------- |
| Host             | `localhost`               |
| Port             | `2525` (12525 in Compose) |
| Authentication   | Optional (PLAIN, LOGIN)   |
| TLS              | Not required              |
| Max Message Size | 10MB (configurable)       |
| Max Recipients   | 50                        |

### Authentication

The server supports two authentication modes:

#### Open Relay Mode (Default)

When `USERNAME` and `PASSWORD` are both empty:

- AUTH mechanisms (PLAIN, LOGIN) are advertised
- Any credentials sent by clients are accepted
- Unauthenticated clients can also send emails

This is suitable for testing environments where you want to capture all emails.

#### Authentication Required Mode

When both `USERNAME` and `PASSWORD` are set:

- Clients must authenticate with the configured credentials
- Invalid credentials are rejected

**Example:**

```bash
# Set credentials via environment variables
export USERNAME=testuser
export PASSWORD=testpass

# Or via Docker
docker run -e USERNAME=testuser -e PASSWORD=testpass \
  -p 8080:8080 -p 2525:2525 \
  ghcr.io/probitas-test/trap-smtp
```

**Send Test Email:**

```bash
# Using swaks
swaks --to test@example.com \
      --from sender@example.com \
      --server localhost:2525 \
      --header "Subject: Test Email" \
      --body "Hello from swaks!"

# Using curl (SMTP)
curl smtp://localhost:2525 \
  --mail-from sender@example.com \
  --mail-rcpt test@example.com \
  --upload-file email.txt

# Using Python (no auth)
python3 -c "
import smtplib
from email.message import EmailMessage

msg = EmailMessage()
msg['From'] = 'sender@example.com'
msg['To'] = 'test@example.com'
msg['Subject'] = 'Test Email'
msg.set_content('Hello from Python!')

with smtplib.SMTP('localhost', 2525) as smtp:
    smtp.send_message(msg)
"

# Using Python (with auth)
python3 -c "
import smtplib
from email.message import EmailMessage

msg = EmailMessage()
msg['From'] = 'sender@example.com'
msg['To'] = 'test@example.com'
msg['Subject'] = 'Test Email'
msg.set_content('Hello from Python with auth!')

with smtplib.SMTP('localhost', 2525) as smtp:
    smtp.login('testuser', 'testpass')  # Use configured credentials
    smtp.send_message(msg)
"
```

---

## Data Types

### EmailEntry

| Field          | Type                | Description                    |
| -------------- | ------------------- | ------------------------------ |
| `id`           | string              | Unique identifier (UUID)       |
| `seq`          | int64               | Monotonically increasing seq#  |
| `received_at`  | string (RFC3339)    | Timestamp when email received  |
| `from`         | string              | Sender address (envelope)      |
| `to`           | []string            | Recipient addresses (envelope) |
| `subject`      | string              | Email subject                  |
| `date`         | string              | Date header value              |
| `message_id`   | string              | Message-ID header              |
| `content_type` | string              | Content-Type header            |
| `headers`      | map[string][]string | All email headers              |
| `body`         | string              | Email body content             |
| `html_body`    | string              | HTML version (if available)    |
| `text_body`    | string              | Plain text version             |
| `raw_email`    | string              | Complete raw email             |
| `attachments`  | []Attachment        | File attachments               |

### Attachment

| Field          | Type   | Description                                 |
| -------------- | ------ | ------------------------------------------- |
| `id`           | string | Unique identifier for download              |
| `filename`     | string | Original filename                           |
| `content_type` | string | MIME type                                   |
| `size`         | int    | Size in bytes                               |
| `sha256`       | string | SHA-256 hex digest of the binary data       |
| `content_id`   | string | Content-ID for inline images (cid:xxx)      |
| `is_inline`    | bool   | True if inline attachment (e.g., cid image) |
