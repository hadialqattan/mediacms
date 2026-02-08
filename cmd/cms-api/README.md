# CMS API

Content Management System API for managing media programs. Provides CRUD operations for programs, user management, and authentication.

## Tech Stack

- **Chi** - HTTP router
- **PGX** - PostgreSQL driver
- **JWT** - Authentication tokens
- **Swagger** - API documentation
- **Redis** - Session storage

## API Endpoints

### Authentication

- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - User logout

### Users

- `POST /api/v1/users` - Create new user (admin only)

### Programs

- `GET /api/v1/programs` - List all programs
- `POST /api/v1/programs` - Create program
- `POST /api/v1/programs/bulk` - Bulk create programs
- `DELETE /api/v1/programs/bulk` - Bulk delete programs
- `GET /api/v1/programs/{id}` - Get program by ID
- `PUT /api/v1/programs/{id}` - Update program
- `POST /api/v1/programs/{id}/publish` - Publish program
- `DELETE /api/v1/programs/{id}` - Delete program

### Health

- `GET /health` - Health check

## Quick Start

```bash
task up:deps
docker compose up cms-api
```

The API will start on port 8080.

## Environment Variables

| Variable                 | Description                   | Default                                                                |
| ------------------------ | ----------------------------- | ---------------------------------------------------------------------- |
| `PORT`                   | Server port                   | `8080`                                                                 |
| `DATABASE_URL`           | PostgreSQL connection string  | `postgres://postgres:postgres@localhost:5432/mediacms?sslmode=disable` |
| `DB_MAX_CONNECTIONS`     | Database connection pool size | `25`                                                                   |
| `REDIS_ADDR`             | Redis address                 | `localhost:6379`                                                       |
| `REDIS_MAX_RETRIES`      | Redis max retry attempts      | `3`                                                                    |
| `JWT_SECRET`             | JWT signing secret            | `secret-key-change-in-production`                                      |
| `JWT_ACCESS_TOKEN_TTL`   | Access token lifetime         | `15m`                                                                  |
| `JWT_REFRESH_TOKEN_TTL`  | Refresh token lifetime        | `720h`                                                                 |
| `DEFAULT_ADMIN_EMAIL`    | Default admin email           | `admin@mediacms.local`                                                 |
| `DEFAULT_ADMIN_PASSWORD` | Default admin password        | `changeme`                                                             |

## Project Structure

```

cmd/cms-api/
└── main.go # Service entry point

internal/cms/
├── auth/ # JWT authentication
├── handler/
├── middleware/
├── port/ # Interfaces (contracts)
├── repository/ # Database access
├── router/ # Route definition
├── service/ # Business logic
└── tests/ # Integration tests

```

## Authentication

The API uses JWT-based authentication. Include the token in the Authorization header:

```

Authorization: Bearer <token>

```

### Token Lifecycle

1. Login returns access and refresh tokens
2. Access tokens expire after 15 minutes (default)
3. Use refresh token to get new access token
4. Refresh tokens expire after 720 hours (30 days)

### User Roles

- **admin** - Full access including user management
- **editor** - Program CRUD operations only

## Testing

Run tests:

```bash
# Unit tests
go test ./internal/cms/... -v -short

# Integration tests (requires infra running)
go test ./internal/cms/... -v -tags=integration -parallel=1
```

## API Documentation

Access Swagger documentation at:

```
http://localhost:8080/swagger/index.html
```
