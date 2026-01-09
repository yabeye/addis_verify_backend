# AddisVerify Backend

> **Note:** This documentation may not always be fully up to date.
AddisVerify Backend is a **production-grade Go REST API** that powers identity verification services.

---

## Tech Stack

* **Language**: Go (1.22+ recommended)
* **HTTP Router**: chi
* **Database**: PostgreSQL (pgx v5 + pgxpool)
* **Cache / Rate Limiting**: Redis
* **SQL Code Generation**: sqlc
* **Database Migrations**: goose
* **API Documentation**: Swagger / OpenAPI (swaggo)
* **Development**: air (hot reload)
* **Containerization**: Docker & Docker Compose

---

## Getting Started

### Prerequisites

* Go
* Docker & Docker Compose
* PostgreSQL client (optional)
* Redis CLI (optional)

Install development tools:

```bash
go install github.com/air-verse/air@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/swaggo/swag/cmd/swag@latest
```

---

## Common Commands

### Development

```bash
# Start API with hot reload
air

# Run API without hot reload
go run ./cmd/api

# Build production binary
go build -o bin/api ./cmd/api
```

---

### Database Operations

```bash
# Run all migrations
goose -dir sql/migrations postgres "$GOOSE_DBSTRING" up

# Roll back last migration
goose -dir sql/migrations postgres "$GOOSE_DBSTRING" down

# Check migration status
goose -dir sql/migrations postgres "$GOOSE_DBSTRING" status

# Create a new migration
goose -dir sql/migrations create <migration_name> sql

# Generate Go code from SQL
sqlc generate
```

---

### API Documentation (Swagger)

```bash
# Generate or update Swagger docs
swag init -g cmd/api/main.go --parseDependency --parseInternal --dir ./
```

When the server is running:

```
http://localhost:8080/swagger/index.html
```

---

### Docker (Local Infrastructure)

```bash
# Start PostgreSQL and Redis
docker compose up -d

# Stop services
docker compose down

# Stop services and remove volumes (clean reset)
docker compose down -v
```

---

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific domain
go test ./internal/account/...
```

---

## Project Structure

```
cmd/api/                Application entry point
  main.go               Bootstraps config, DB, Redis, server
  api.go                HTTP server & global middleware
  routes.go             Feature route registration

internal/               Private application code
  account/              Account domain (handlers, service, requests)
  verify/               Verification domain (WIP)
  database/             sqlc-generated code (DO NOT EDIT)
  store/                DB & Redis connection factories
  middlewares/          Custom HTTP middlewares
  env/                  Environment variable helpers

pkg/                    Shared reusable utilities
  json/                 JSON helpers and error responses

sql/
  migrations/           Goose migration files
  query.sql             SQL queries for sqlc

docs/                   Swagger docs (auto-generated)
```

---

## Architecture Overview

### Application Flow

1. **Startup (`main.go`)**

   * Load environment variables (`.env` optional)
   * Initialize structured logging (`slog`)
   * Parse configuration
   * Create PostgreSQL connection pool
   * Create Redis client
   * Initialize application container
   * Start HTTP server

2. **HTTP Server (`api.go`)**

   * chi router with global middleware:

     * Request ID
     * Real IP
     * Structured logging
     * Panic recovery
     * CORS (dev-safe defaults)
     * Request size limit (1MB)
     * Global rate limiting (100 req/min)
     * Request timeout (30s)
     * Response compression
   * Routes:

     * `/health`
     * `/swagger/*`
     * Feature-specific endpoints

3. **Domain Design**

   * Handler → Service → Repository pattern
   * No business logic in handlers
   * All DB access via sqlc-generated code

---

## Database & SQL Workflow

1. Create a migration:

```bash
goose -dir sql/migrations create <migration_name> sql
```

2. Edit migration file:

```sql
-- +goose Up
CREATE TABLE example (...);

-- +goose Down
DROP TABLE example;
```

3. Apply migrations:

```bash
goose -dir sql/migrations postgres "$GOOSE_DBSTRING" up
```

4. Add queries to `sql/query.sql`
5. Generate Go code:

```bash
sqlc generate
```

Generated code lives in `internal/database/` and must not be edited manually.

---

## API Documentation Standards

* All handlers must include Swagger annotations
* Example:

```go
// CreateAccount godoc
// @Summary Create account
// @Description Registers a new account
// @Tags account
// @Accept json
// @Produce json
// @Success 201 {object} AccountResponse
// @Failure 400 {object} ErrorResponse
// @Router /accounts [post]
```

* Regenerate docs after changes
* Air automatically regenerates Swagger docs before rebuild

---

## Middleware & Error Handling

* Global middleware is mounted in `api.go`
* Route-specific middleware via chi route groups
* Custom middlewares live in `internal/middlewares/`
* Consistent error responses via:

```go
pkg/json.WriteError(w, statusCode, message)
```

---

## Environment Variables

See `.env.example`.

Required:

* `GOOSE_DBSTRING` – PostgreSQL connection string
* `GOOSE_DRIVER` – `postgres`
* `GOOSE_MIGRATION_DIR` – `sql/migrations`
* `JWT_SECRET` – JWT signing key
* `ADDR` – Server address (default `:8080`)
* `REDIS_ADDR` – Redis address (`localhost:6379`)
* `DEFAULT_FALLBACK_URL` – Redirect fallback

Example connection string:

```
postgresql://postgres:postgres@localhost:5432/addis_verify_test?sslmode=disable
```

---

## Code Generation (Important)

The following directories are auto-generated and **must not be edited manually**:

* `internal/database/` (sqlc)
* `docs/` (swag)

Regenerate with:

```bash
sqlc generate
swag init -g cmd/api/main.go --parseDependency --parseInternal --dir ./
```

---

## Docker Compose Services

Local development provides:

* **PostgreSQL 16**

  * Port: 5432
  * Database: `addis_verify_test`
  * User/Password: `postgres / postgres`
  * Health checks enabled

* **Redis 7**

  * Port: 6379
  * Persistent volume

---

## Internal Conventions

* No raw SQL outside `sql/query.sql`
* No environment access outside `internal/env`
* No business logic in HTTP handlers
* All schema changes require migrations
* All endpoints require Swagger documentation
