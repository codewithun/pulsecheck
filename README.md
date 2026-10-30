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
