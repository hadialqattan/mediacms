# Discovery API

Public discovery (search) API for media programs. Provides full-text search over programs titles & descriptions with filtering and faceted navigation.

## Tech Stack

- **Chi** - HTTP router
- **Typesense** - Search engine client
- **Swagger** - API documentation

## Search Capabilities

- **Full-text search** across title and description
- **Filtering** by type, language, and tags
- **Sorting** by relevance, recency, or oldest first
- **Pagination** with configurable page size
- **Faceted counts** for type, language, and tags

## API Endpoints

### Programs

- `GET /api/v1/programs` - Search programs with filters
- `GET /api/v1/programs/{id}` - Get program by ID
- `GET /api/v1/programs/recent` - Get recently published programs
- `GET /api/v1/programs/facets` - Get all available facets

### Health

- `GET /health` - Health check

## Quick Start

```bash
task up:deps
docker compose up discovery-api
```

## Environment Variables

| Variable            | Description       | Default                 |
| ------------------- | ----------------- | ----------------------- |
| `PORT`              | Server port       | `8081`                  |
| `TYPESENSE_ADDRESS` | Typesense address | `http://localhost:8108` |
| `TYPESENSE_API_KEY` | Typesense API key | `xyz`                   |

## Project Structure

```
cmd/discovery-api/
└── main.go              # Service entry point

internal/discovery/
├── handler/
├── port/                # Interfaces (contracts)
├── repository/          # Typesense access
├── router/              # Route definitions
├── service/             # Business logic
└── tests/               # Integration tests
```

## Testing

Run tests:

```bash
# Unit tests
go test ./internal/discovery/... -v -short

# Integration tests
go test ./internal/discovery/... -v -tags=integration -parallel=1
```

## API Documentation

Access Swagger documentation at:

```
http://localhost:8081/swagger/index.html
```
