# Configuration Package

Centralized configuration loading from environment variables for all services.

## Environment Variables

### Service Ports

| Variable | Default          | Description                                  |
| -------- | ---------------- | -------------------------------------------- |
| `PORT`   | Service-specific | Server port (8080-8083 depending on service) |

### Database

| Variable             | Default                                                                | Description                  |
| -------------------- | ---------------------------------------------------------------------- | ---------------------------- |
| `DATABASE_URL`       | `postgres://postgres:postgres@localhost:5432/mediacms?sslmode=disable` | PostgreSQL connection string |
| `DB_MAX_CONNECTIONS` | `25`                                                                   | Maximum connection pool size |

### Redis

| Variable                  | Default          | Description            |
| ------------------------- | ---------------- | ---------------------- |
| `REDIS_ADDR`              | `localhost:6379` | Redis server address   |
| `REDIS_MAX_RETRIES`       | `3`              | Maximum retry attempts |
| `REDIS_MIN_RETRY_BACKOFF` | `500ms`          | Minimum retry backoff  |
| `REDIS_MAX_RETRY_BACKOFF` | `1s`             | Maximum retry backoff  |

### Typesense

| Variable            | Default                 | Description          |
| ------------------- | ----------------------- | -------------------- |
| `TYPESENSE_ADDRESS` | `http://localhost:8108` | Typesense server URL |
| `TYPESENSE_API_KEY` | `xyz`                   | Typesense API key    |

### JWT (CMS API only)

| Variable                | Default                           | Description            |
| ----------------------- | --------------------------------- | ---------------------- |
| `JWT_SECRET`            | `secret-key-change-in-production` | JWT signing secret     |
| `JWT_ACCESS_TOKEN_TTL`  | `15m`                             | Access token lifetime  |
| `JWT_REFRESH_TOKEN_TTL` | `720h` (30 days)                  | Refresh token lifetime |

### Default Admin (CMS API only)

| Variable                 | Default                | Description            |
| ------------------------ | ---------------------- | ---------------------- |
| `DEFAULT_ADMIN_EMAIL`    | `admin@mediacms.local` | Default admin email    |
| `DEFAULT_ADMIN_PASSWORD` | `changeme`             | Default admin password |

### Outbox Relay (Outbox Relay only)

| Variable                | Default | Description                        |
| ----------------------- | ------- | ---------------------------------- |
| `OUTBOX_RELAY_INTERVAL` | `5s`    | Polling interval for outbox events |
