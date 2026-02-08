# Outbox Relay

Background service that polls the outbox table and enqueues events for processing by the Search Indexer.

## Tech Stack

- **PGX** - PostgreSQL driver
- **Asynq** - Redis-based job queue

## How It Works

The Outbox Relay implements a polling mechanism to reliably deliver events:

```mermaid
flowchart LR
    Relay[Outbox Relay] -->|Poll Every N Seconds| DB[(PostgreSQL)]
    DB -->|1 Unenqueued Events| Relay
    Relay -->|2 Enqueue| Redis[(Redis Queue)]
    Relay -->|3 Mark Enqueued| DB
```

### Polling Mechanism

1. Polls database every `OUTBOX_RELAY_INTERVAL` (default: 5 seconds)
2. Queries for unenqueued outbox events
3. Enqueues events to Redis using Asynq
4. Marks events as enqueued in database (only if successfuly enqueued)
5. Continues polling until stopped

### Event Types

- `program.upsert` - Program created or updated
- `program.delete` - Program deleted

## Quick Start

```bash
task up:deps
docker compose up outbox-relay
```

## Environment Variables

| Variable                | Description                     | Default                                                                |
| ----------------------- | ------------------------------- | ---------------------------------------------------------------------- |
| `PORT`                  | Server port (for health checks) | `8082`                                                                 |
| `DATABASE_URL`          | PostgreSQL connection string    | `postgres://postgres:postgres@localhost:5432/mediacms?sslmode=disable` |
| `REDIS_ADDR`            | Redis address                   | `localhost:6379`                                                       |
| `OUTBOX_RELAY_INTERVAL` | Polling interval                | `5s`                                                                   |

## Project Structure

```
cmd/outbox-relay/
└── main.go              # Service entry point

internal/outboxrelay/
├── repository/          # Database and queue access
│   ├── outbox.go       # Outbox queries
│   └── queue.go        # Asynq client
├── port/                # Interfaces (contracts)
├── relay.go             # Polling logic
└── tests/               # Integration tests
```

### Event Payload

Each enqueued event contains:

```json
{
  "id": "event-uuid",
  "type": "program.upsert",
  "payload": {
    "program_id": "program-uuid",
    "slug": "example",
    "title": "Example",
    ...
  }
}
```

## Reliability Features

### Idempotency

- Events are marked as enqueued after successful queue insertion
- If relay crashes, unenqueued events will be picked up on restart
- Duplicate events are idempotent in the search indexer

### Error Handling

- Database connection errors cause relay retry
- Queue errors prevent events from being marked enqueued
- Failed events remain in queue for retry

### Ordering

- Events are processed in creation order
- Bulk operations create multiple events
- Search indexer handles events sequentially per program

## Monitoring

The service logs polling activity:

```
Starting Outbox Relay...
Polled 5 unenqueued events
Enqueued event: program.upsert
Marked event as enqueued: uuid
```

## Testing

Run tests:

```bash
# Integration tests (requires infra running)
go test ./internal/outboxrelay/... -v -tags=integration -parallel=1
```
