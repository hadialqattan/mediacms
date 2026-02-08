# Search Indexer

Background worker that consumes outbox events and keeps the Typesense search index in sync with PostgreSQL. Handles program upserts and deletions.

## Tech Stack

- **Asynq** - Redis-based job queue
- **Typesense Go Client** - Search engine client
- **PGX** - PostgreSQL driver

## What It Does

The Search Indexer:

1. Consumes events from Redis queue
2. Fetches program data from PostgreSQL
3. Updates Typesense collection
4. Handles upsert and delete operations
5. Ensures collection exists on startup

```mermaid
flowchart LR
    Queue[(Redis Queue)] -->|Consume Events| Worker[Search Indexer Worker]
    Worker -->|Fetch Program| DB[(PostgreSQL)]
    DB -->|Program Data| Worker
    Worker -->|Upsert/Delete| Typesense[(Typesense)]
```

## Quick Start

### Local Development

```bash
cd cmd/search-indexer
go run main.go
```

### Docker Compose

```bash
docker compose up search-indexer
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8083` |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/mediacms?sslmode=disable` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `TYPESENSE_ADDRESS` | Typesense address | `http://localhost:8108` |
| `TYPESENSE_API_KEY` | Typesense API key | `xyz` |

## Project Structure

```
cmd/search-indexer/
└── main.go              # Service entry point with retry logic

internal/searchindexer/
├── handler.go           # Event handlers
├── mux.go               # Asynq task mux
└── worker.go            # Worker setup
```

## Data Flow

```mermaid
sequenceDiagram
    participant Queue as Redis Queue
    participant Worker as Search Indexer
    participant DB as PostgreSQL
    participant TS as Typesense

    Queue->>Worker: Deliver task
    Worker->>DB: Fetch program by ID
    DB-->>Worker: Return program

    alt Upsert Event
        Worker->>Worker: Check if published
        alt Published
            Worker->>TS: Upsert document
            TS-->>Worker: Ack
        else Unpublished/Deleted
            Worker->>TS: Do nothing
        end
    else Delete Event
        Worker->>TS: Delete document
        TS-->>Worker: Ack
    end

    Worker-->>Queue: Task complete
```

## Event Handling

### Program Upsert

1. Fetch program from PostgreSQL by ID
2. Check if program is published (`published_at` not null, `deleted_at` is null)
3. If published: upsert to Typesense
4. If not published: skip (document won't appear in search)
5. Acknowledge task

### Program Delete

1. Delete document from Typesense by ID
2. Acknowledge task

### Idempotency

- Upsert operations are idempotent (same result if run multiple times)
- Delete operations are idempotent (no error if document doesn't exist)
- Failed tasks are retried by Asynq

## Typesense Index Schema

### Collection: `programs`

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Program UUID |
| `slug` | string | URL-friendly identifier |
| `title` | string | Program title |
| `description` | string | Program description |
| `type` | string | `podcast` or `documentary` (faceted) |
| `language` | string | `ar` or `en` (faceted) |
| `duration_ms` | int32 | Duration in milliseconds |
| `tags` | string[] | Tags array (faceted) |
| `published_at` | int64 | Unix timestamp |
| `created_at` | int64 | Unix timestamp |

### Default Sorting

- Default sort field: `published_at`
- Ensures consistent relevance scoring

## Collection Initialization

On startup, the service:

1. Retries up to 10 times with exponential backoff
2. Checks if collection exists
3. Creates collection with schema if missing
4. Starts worker after collection is ready

## Reliability Features

### Retry Logic

- Asynq automatically retries failed tasks
- Configurable retry count and backoff
- Collection creation retries with exponential backoff

### Error Handling

- Database errors: task is retried
- Typesense errors: task is retried
- Malformed events: logged and acknowledged (no retry)

### Concurrency

- Asynq handles concurrent task processing
- Multiple workers can run in parallel
- Each worker processes tasks independently

## Testing

Run tests:

```bash
# Unit tests
go test ./internal/searchindexer/... -v -short

# Integration tests
go test ./internal/searchindexer/... -v -tags=integration -parallel=1
```

## Monitoring

The service logs indexing activity:

```
Ensuring Typesense collection exists...
Typesense collection ready
Starting Search Indexer worker...
Processing task: program.upsert
Upserted program: uuid
Task complete
```

## Configuration

### Worker Concurrency

Adjust worker pool size in Asynq server configuration.

### Task Priority

All events use default priority. Customize by setting priority in enqueue.

### Retry Policy

Configure Asynq retry options in worker setup.

## Troubleshooting

### Collection Not Created

Check:
1. Typesense is accessible at configured address
2. API key is correct
3. Logs show "Typesense collection ready"

### Events Not Processing

Check:
1. Redis queue has tasks: `redis-cli LRANGE asynq:queues:default 0 -1`
2. Worker is running and connected to Redis
3. Database contains programs being indexed

### Out of Sync

If Typesense is out of sync with PostgreSQL:
1. Delete the Typesense collection
2. Restart the worker (will recreate collection)
3. Re-publish all programs to trigger re-indexing
