# Database Migrations

PostgreSQL schema migrations using golang-migrate/migrate.

## Running Migrations

### Using Task

```bash
task migrate
```

This applies all pending migrations.

### Manual Migrate Command

```bash
migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/mediacms?sslmode=disable" up/down
```

## SQLC Integration

SQLC generates type-safe Go code from SQL queries.

### Generate Code

```bash
task sqlc
```

### SQLC Queries

Place queries in `sqlc.yaml` configured directories. Run `sqlc generate` after modifying queries.

### Files

- `internal/cms/query.sql` - CMS queries
- `internal/outboxrelay/query.sql` - Outbox relay queries
- `sqlc.yaml` - SQLC configuration
