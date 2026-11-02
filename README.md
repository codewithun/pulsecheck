## Centralized Monitoring and Discord Alerting

PulseCheck also provides a centralized monitoring system that receives health-check results from probe agents, stores monitoring data in PostgreSQL, and sends Discord notifications when target availability changes.

### Architecture

- **PulseCheck CLI** — Performs HTTP and TCP health checks.
- **Probe Agent** — Sends monitoring results to the Central API.
- **Central API** — Receives and stores check results.
- **PostgreSQL** — Stores check history, monitoring states, and pending alerts.
- **Discord Alert Worker** — Processes alert events and sends notifications through a Discord webhook.

### Alert Features

- Send a Discord notification when a target transitions from `UP` to `DOWN`.
- Send a recovery notification when a target transitions from `DOWN` to `UP`.
- Avoid duplicate notifications when a target remains in the same state.
- Ignore `INVALID` results when updating operational monitoring state.
- Store alert events in a PostgreSQL outbox before delivery.
- Retry failed Discord deliveries using exponential backoff, capped at 15 minutes.
- Keep monitoring operational when Discord delivery fails.

The first valid `UP` or `DOWN` result establishes the monitoring baseline and does not trigger an alert.

Alert delivery uses an **at-least-once** approach. In rare cases, a notification may be delivered more than once if Discord accepts it but the database cannot record the successful delivery.

### Configuration

Configure the following environment variables:

| Variable              | Description                              |
| --------------------- | ---------------------------------------- |
| `POSTGRES_PASSWORD`   | PostgreSQL password                      |
| `DATABASE_URL`        | PostgreSQL connection string             |
| `API_TOKEN`           | Authentication token for the Central API |
| `DISCORD_WEBHOOK_URL` | Discord webhook URL for notifications    |

Create a local `.env` file based on `.env.example`. Never commit `.env` or expose your Discord webhook URL and API token.

### Running the Central API

Start PostgreSQL and ensure migrations have been applied. Then run:

```bash
set -a
source .env
set +a

API_ADDR=:8081 go run ./cmd/central-api
```

Health check:

```bash
curl http://localhost:8081/healthz
```

The API starts the Discord alert worker when `DISCORD_WEBHOOK_URL` is configured.

### Probe Agent Configuration

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

#### Configuration Fields

| Field      | Description                             |
| ---------- | --------------------------------------- |
| `name`     | Human-readable target name used in logs |
| `target`   | URL or host and port to monitor         |
| `protocol` | Monitoring protocol: `http` or `tcp`    |
| `interval` | Time between consecutive checks         |
| `timeout`  | Maximum duration for each check         |

All fields are required. Interval and timeout must be positive duration values.

#### Running the Probe Agent

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

#### Legacy Configuration

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

### Agent Heartbeat

PulseCheck includes an agent heartbeat mechanism to track the availability of registered Probe Agents independently from their monitoring targets.

Each Probe Agent automatically sends a heartbeat to the Central API every 30 seconds through a dedicated background worker.

**Heartbeat Features**

- Automatically registers agents using their hostname as the probe ID.
- Sends heartbeat updates every 30 seconds.
- Updates the agent's `last_seen` timestamp in PostgreSQL.
- Determines agent availability based on the last heartbeat received.
- Marks agents as `ONLINE` when the last heartbeat was received within 90 seconds.
- Marks agents as `OFFLINE` when no heartbeat has been received for more than 90 seconds.
- Runs independently from target monitoring workers.
- Stops gracefully when the Probe Agent receives a shutdown signal.

**Agent API Endpoints**

| Method | Endpoint                   | Description                                   |
| ------ | -------------------------- | --------------------------------------------- |
| `POST` | `/api/v1/agents/heartbeat` | Register an agent or update its heartbeat     |
| `GET`  | `/api/v1/agents`           | Retrieve registered agents and their statuses |

Both endpoints require Bearer token authentication.

**Example: Heartbeat Request**

```json
{
  "probe_id": "192.168.1.8",
  "hostname": "macbook"
}
```

**Example: Agent List Response**

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

### API Endpoints

| Method | Endpoint          | Description                   |
| ------ | ----------------- | ----------------------------- |
| `GET`  | `/healthz`        | Check API and database health |
| `POST` | `/api/v1/results` | Submit a probe result         |
| `GET`  | `/api/v1/results` | Retrieve check results        |
| `GET`  | `/api/v1/summary` | Retrieve monitoring summary   |

The `/healthz` endpoint does not require authentication. The `/api/v1/*` endpoints require a Bearer token.

### Database Migrations

Database migration files are located in `migrations/`:

- `001` — Initial check results schema.
- `002` — Monitoring state.
- `003` — Alert outbox.
- `004` — Alert outbox claim support.
- `005` — Registered agents and heartbeat tracking.

Apply migrations in numerical order.

### Alerting Tests

Run all tests:

```bash
go test -count=1 ./...
```

Run static analysis:

```bash
go vet ./...
```

Integration tests use a dedicated PostgreSQL test database configured through `PULSECHECK_TEST_DATABASE_URL`. The test database must be named `pulsecheck_test`.

Discord sender tests use an HTTP test server. They do not send actual Discord messages.
