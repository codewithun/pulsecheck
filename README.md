# Centralized Monitoring and Discord Alerting

PulseCheck provides a centralized monitoring system that receives health-check results from Probe Agents, stores monitoring data in PostgreSQL, tracks agent availability, and sends Discord notifications when target availability changes.

## Architecture

- **PulseCheck CLI** — Performs HTTP and TCP health checks.
- **Probe Agent** — Runs scheduled health checks and sends monitoring results to the Central API.
- **Central API** — Receives monitoring results, manages agent heartbeats, exposes monitoring data, and processes alert events.
- **PostgreSQL** — Stores check history, monitoring states, registered agents, and pending alerts.
- **Discord Alert Worker** — Processes alert events and sends notifications through a Discord webhook.

## Alert Features

- Sends a Discord notification when a target transitions from `UP` to `DOWN`.
- Sends a recovery notification when a target transitions from `DOWN` to `UP`.
- Prevents repeated notifications when a target remains in the same operational state.
- Ignores `INVALID` results when updating operational monitoring state.
- Stores alert events in a PostgreSQL outbox before delivery.
- Retries failed Discord deliveries using exponential backoff, capped at 15 minutes.
- Keeps monitoring operational when Discord delivery fails.

The first valid `UP` or `DOWN` result establishes the monitoring baseline and does not trigger an alert.

Alert delivery uses an **at-least-once** approach. In rare cases, a notification may be delivered more than once if Discord accepts it but the database cannot record the successful delivery.

## Configuration

Configure the following environment variables:

| Variable              | Description                              |
| --------------------- | ---------------------------------------- |
| `POSTGRES_PASSWORD`   | PostgreSQL password                      |
| `DATABASE_URL`        | PostgreSQL connection string             |
| `API_TOKEN`           | Authentication token for the Central API |
| `DISCORD_WEBHOOK_URL` | Discord webhook URL for notifications    |

Create a local `.env` file based on `.env.example`.

Never commit `.env` or expose your Discord webhook URL and API token.

## Running the Central API

Start PostgreSQL and ensure all database migrations have been applied.

Then run:

```bash
set -a
source .env
set +a

API_ADDR=:8081 go run ./cmd/central-api
```

### Health Check

```bash
curl http://localhost:8081/healthz
```

The `/healthz` endpoint checks API and database health without requiring authentication.

The API starts the Discord alert worker when `DISCORD_WEBHOOK_URL` is configured.

## Probe Agent Configuration

The Probe Agent supports JSON-based per-target configuration. Each target can define its own name, protocol, monitoring interval, and timeout.

Create a `targets.example.json` file:

```json
{
  "targets": [
    {
      "name": "Google",
      "target": "https://google.com",
      "protocol": "http",
      "interval": "30s",
      "timeout": "5s"
    },
    {
      "name": "GitHub",
      "target": "https://github.com",
      "protocol": "http",
      "interval": "60s",
      "timeout": "10s"
    },
    {
      "name": "PostgreSQL",
      "target": "localhost:5433",
      "protocol": "tcp",
      "interval": "10s",
      "timeout": "3s"
    }
  ]
}
```

### Configuration Fields

| Field      | Description                             |
| ---------- | --------------------------------------- |
| `name`     | Human-readable target name used in logs |
| `target`   | URL or host and port to monitor         |
| `protocol` | Monitoring protocol: `http` or `tcp`    |
| `interval` | Time between consecutive checks         |
| `timeout`  | Maximum duration for each check         |

All fields are required. Interval and timeout must be positive duration values.

### Running the Probe Agent

Ensure the Central API is running and `API_TOKEN` is configured in `.env`.

```bash
set -a
source .env
set +a

go run ./cmd/probe-agent \
  -api http://localhost:8081 \
  -config targets.example.json
```

Each target runs independently with its configured interval and timeout. The Probe Agent supports concurrent monitoring of multiple targets.

### Legacy Configuration

For backward compatibility, the Probe Agent also supports command-line target configuration:

```bash
go run ./cmd/probe-agent \
  -api http://localhost:8081 \
  -protocol http \
  -targets https://google.com,https://github.com \
  -interval 30s \
  -timeout 5s
```

When `-config` is provided, JSON configuration takes precedence over the legacy target flags.

## Agent Heartbeat

PulseCheck includes an agent heartbeat mechanism to track the availability of registered Probe Agents independently from their monitoring targets.

Each Probe Agent automatically sends a heartbeat to the Central API every 30 seconds through a dedicated background worker.

### Heartbeat Features

- Automatically registers agents using their hostname as the probe ID.
- Sends heartbeat updates every 30 seconds.
- Updates the agent's `last_seen` timestamp in PostgreSQL.
- Determines agent availability based on the last heartbeat received.
- Marks agents as `ONLINE` when the last heartbeat was received within 90 seconds.
- Marks agents as `OFFLINE` when no heartbeat has been received for more than 90 seconds.
- Runs independently from target monitoring workers.
- Stops gracefully when the Probe Agent receives a shutdown signal.

### Agent API Endpoints

| Method | Endpoint                   | Description                                   |
| ------ | -------------------------- | --------------------------------------------- |
| `POST` | `/api/v1/agents/heartbeat` | Register an agent or update its heartbeat     |
| `GET`  | `/api/v1/agents`           | Retrieve registered agents and their statuses |

Both endpoints require Bearer token authentication.

### Example: Heartbeat Request

```json
{
  "probe_id": "192.168.1.8",
  "hostname": "macbook"
}
```

### Example: Agent List Response

```json
{
  "data": [
    {
      "probe_id": "192.168.1.8",
      "hostname": "macbook",
      "last_seen": "2026-11-02T19:51:43+07:00",
      "status": "ONLINE"
    }
  ]
}
```

The heartbeat worker does not interrupt target monitoring when the Central API is temporarily unavailable. Failed heartbeat requests are logged, and the worker retries on its next scheduled interval.

## Monitoring API Endpoints

All `/api/v1/*` endpoints require Bearer token authentication.

| Method | Endpoint                   | Description                                                                     |
| ------ | -------------------------- | ------------------------------------------------------------------------------- |
| `POST` | `/api/v1/results`          | Submit a probe result                                                           |
| `GET`  | `/api/v1/results`          | Retrieve check results                                                          |
| `GET`  | `/api/v1/summary`          | Retrieve the latest monitoring result per probe and target                      |
| `GET`  | `/api/v1/overview`         | Retrieve aggregated monitoring statistics                                       |
| `GET`  | `/api/v1/targets`          | Retrieve monitored targets with operational status and latest check information |
| `POST` | `/api/v1/agents/heartbeat` | Register an agent or update its heartbeat                                       |
| `GET`  | `/api/v1/agents`           | Retrieve registered agents and their statuses                                   |

The `/healthz` endpoint does not require authentication.

## Targets API

The Targets API provides a consolidated view of monitored targets across all registered probes.

Each target is identified by the combination of `probe_id` and `target`. This allows the same URL or host to be monitored independently by multiple agents.

### Endpoint

```http
GET /api/v1/targets
```

### Query Parameters

| Parameter  | Required | Default     | Description                                                          |
| ---------- | -------- | ----------- | -------------------------------------------------------------------- |
| `limit`    | No       | `50`        | Number of records to return. Minimum 1, maximum 100.                 |
| `offset`   | No       | `0`         | Number of records to skip. Must be non-negative.                     |
| `probe_id` | No       | —           | Filter targets by exact probe ID.                                    |
| `status`   | No       | —           | Filter by operational status: `UP`, `DOWN`, or `UNKNOWN`.            |
| `target`   | No       | —           | Search targets using a case-insensitive partial match.               |
| `period`   | No       | All history | Include target pairs with checks within `1h`, `24h`, `7d`, or `30d`. |

When `period` is omitted, all historically observed target pairs are included, including stale targets.

When `period` is specified, it determines which target pairs are included. The latest check information for each included target is still taken from its full history.

### Example: Retrieve All Targets

```bash
curl -sS \
  -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/v1/targets?limit=10"
```

### Example: Filter by Operational Status

```bash
curl -sS \
  -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/v1/targets?status=UP"
```

### Example: Filter by Probe ID

```bash
curl -sS \
  -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/v1/targets?probe_id=192.168.1.8"
```

### Example: Search by Target

```bash
curl -sS \
  -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/v1/targets?target=google"
```

### Example: Filter by Period

```bash
curl -sS \
  -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/v1/targets?period=24h"
```

### Example: Pagination

```bash
curl -sS \
  -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/v1/targets?limit=10&offset=10"
```

### Example: Combined Filters

```bash
curl -sS \
  -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/v1/targets?probe_id=192.168.1.8&status=UP&period=24h&limit=10&offset=0"
```

### Example Response

```json
{
  "data": [
    {
      "probe_id": "192.168.1.8",
      "target": "https://google.com",
      "status": "UP",
      "last_check_status": "UP",
      "last_checked_at": "2026-11-02T19:51:43.302929+07:00",
      "latency_ms": 36.524,
      "protocol": "HTTP"
    },
    {
      "probe_id": "192.168.1.8",
      "target": "localhost:5433",
      "status": "UP",
      "last_check_status": "UP",
      "last_checked_at": "2026-11-02T19:51:43.234556+07:00",
      "latency_ms": 1.523,
      "protocol": "TCP"
    }
  ],
  "limit": 10,
  "offset": 0,
  "has_more": true
}
```

### Response Fields

| Field               | Description                                              |
| ------------------- | -------------------------------------------------------- |
| `probe_id`          | ID of the monitoring agent                               |
| `target`            | Monitored URL or host and port                           |
| `status`            | Current operational state from monitoring state records  |
| `last_check_status` | Status of the latest recorded check, including `INVALID` |
| `last_checked_at`   | Timestamp of the latest recorded check                   |
| `latency_ms`        | Latency of the latest check in milliseconds              |
| `protocol`          | Check protocol: `HTTP` or `TCP`                          |
| `error`             | Error message when available                             |
| `limit`             | Maximum number of records requested                      |
| `offset`            | Number of records skipped                                |
| `has_more`          | Indicates whether additional records are available       |

**Operational status and last check status are different concepts.**

- `status` represents the operational state maintained by the monitoring state mechanism.
- `last_check_status` represents the most recent check result.
- An `INVALID` result does not change the operational state.
- A target without an established operational state is reported as `UNKNOWN`.

### Error Responses

Invalid query parameters return HTTP `400 Bad Request`.

Examples include:

- `limit` outside the range of 1–100.
- Negative or invalid `offset`.
- Unsupported `status` values.
- Unsupported `period` values.

## Overview API

The Overview API provides aggregated monitoring statistics for the dashboard.

### Endpoint

```http
GET /api/v1/overview
```

### Query Parameters

| Parameter | Default | Description                               |
| --------- | ------- | ----------------------------------------- |
| `period`  | `24h`   | Time window: `1h`, `24h`, `7d`, or `30d`. |

### Overview Metrics

- Total registered agents.
- Online and offline agent counts.
- Total monitored probe-target pairs.
- UP, DOWN, and UNKNOWN target counts.
- Check counts grouped by UP, DOWN, and INVALID.
- Observed availability percentage.
- Pending and delivered alert counts.

Availability is calculated as:

`UP checks / (UP checks + DOWN checks) × 100`

`INVALID` checks are excluded. Availability is returned as `null` when there are no valid UP or DOWN checks in the selected period.

Target counts represent distinct `(probe_id, target)` pairs. Check statistics are filtered by the selected period.

## Database Migrations

Database migration files are located in `migrations/`:

- `001` — Initial check results schema.
- `002` — Monitoring state.
- `003` — Alert outbox.
- `004` — Alert outbox claim support.
- `005` — Registered agents and heartbeat tracking.

Apply migrations in numerical order.

## Testing

Run all tests:

```bash
go test -count=1 ./...
```

Run static analysis:

```bash
go vet ./...
```

Integration tests use a dedicated PostgreSQL test database configured through `PULSECHECK_TEST_DATABASE_URL`.

The test database must be named `pulsecheck_test`.

Discord sender tests use an HTTP test server. They do not send actual Discord messages.
