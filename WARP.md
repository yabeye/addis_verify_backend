# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview
AddisVerify Backend is a Go-based REST API for identity verification services. The application uses:
- **Framework**: chi router for HTTP routing
- **Database**: PostgreSQL (via pgx/v5) with connection pooling
- **Cache**: Redis for session/rate limiting data
- **Code Generation**: sqlc for type-safe SQL queries
- **Migrations**: Goose for database schema management
- **API Documentation**: Swagger/OpenAPI via swaggo
- **Development**: Air for hot-reloading during development

## Common Commands

### Development
```bash
# Start development server with hot-reload (builds on file changes)
air

# Run without hot-reload
go run ./cmd/api

# Build binary
go build -o bin/api ./cmd/api
```

### Database Operations
```bash
# Run migrations up
goose -dir sql/migrations postgres "$GOOSE_DBSTRING" up

# Rollback last migration
goose -dir sql/migrations postgres "$GOOSE_DBSTRING" down

# Check migration status
goose -dir sql/migrations postgres "$GOOSE_DBSTRING" status

# Create new migration
goose -dir sql/migrations create <migration_name> sql

# Generate Go code from SQL (after modifying sql/query.sql or migrations)
sqlc generate
```

### API Documentation
```bash
# Generate/update Swagger docs (run after changing handler annotations)
swag init -g cmd/api/main.go --parseDependency --parseInternal --dir ./

# View docs: http://localhost:8080/swagger/index.html (when server running)
```

### Docker
```bash
# Start PostgreSQL and Redis for local development
docker compose up -d

# Stop services
docker compose down

# Stop and remove volumes (clean slate)
docker compose down -v
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/account/...
```

## Architecture

### Project Structure
```
cmd/api/          - Application entry point and HTTP server setup
  main.go         - Initializes app, DB, Redis, starts server
  api.go          - Defines application struct, server config, mount middleware
  routes.go       - Feature-specific route mounting (currently stubbed)

internal/         - Private application code
  account/        - Account management domain (handlers, service, requests)
  verify/         - Verification domain (planned/WIP)
  database/       - Generated sqlc code (do not edit manually)
  store/          - Database and cache connection factories
  middlewares/    - Custom HTTP middlewares (rate limiting, size limits)
  env/            - Environment variable helpers

pkg/              - Shared utilities (can be imported by external projects)
  json/           - JSON encoding/error response utilities

sql/
  migrations/     - Goose migration files (versioned schema changes)
  query.sql       - SQL queries for sqlc to generate Go code from

docs/             - Auto-generated Swagger documentation (do not edit manually)
```

### Application Flow
1. **Initialization** (`main.go`):
   - Load `.env` file (uses system env vars if not present)
   - Setup structured logging (slog)
   - Parse configuration from environment variables
   - Establish PostgreSQL connection pool (10 max connections)
   - Establish Redis client connection
   - Initialize application struct with config, db, cache, logger

2. **HTTP Server** (`api.go`):
   - Creates chi router with middleware stack applied globally:
     - Request ID, Real IP, Logger, Recoverer (from chi/middleware)
     - CORS (allows localhost:3000 origin)
     - Request size limit (1MB max payload)
     - Global rate limit (100 req/min)
     - Request timeout (30s max)
     - Response compression (level 5)
   - Mounts Swagger UI at `/swagger/*`
   - Mounts health check at `/health`
   - Returns configured handler to main

3. **Database Layer**:
   - Uses sqlc to generate type-safe Go code from SQL
   - SQL queries defined in `sql/query.sql`
   - Schema managed via Goose migrations in `sql/migrations/`
   - Generated code outputs to `internal/database/` with package name `repo`
   - Connection pooling handled by pgxpool

4. **Feature Domains** (Handler → Service pattern):
   - Each domain (e.g., account, verify) follows a layered structure:
     - `handler.go`: HTTP request/response handling
     - `service.go`: Business logic
     - `requests.go`: Request validation structs
   - Handlers receive application dependencies via the app struct
   - Services interact with database via generated sqlc queries

### Key Development Patterns

**Environment Configuration**:
- All configuration comes from environment variables
- Use `internal/env.GetString(key, fallback)` for retrieving values
- Never hardcode secrets or connection strings

**Database Workflow**:
1. Create migration: `goose -dir sql/migrations create <name> sql`
2. Edit migration file with `-- +goose Up` and `-- +goose Down` sections
3. Run migration: `goose -dir sql/migrations postgres "$GOOSE_DBSTRING" up`
4. Add queries to `sql/query.sql`
5. Generate code: `sqlc generate`
6. Use generated types/functions from `internal/database/repo`

**API Documentation**:
- Add godoc comments above handlers with swag annotations
- Format: `// HandlerName godoc` followed by `@Summary`, `@Description`, `@Tags`, etc.
- Regenerate docs after changes: `swag init -g cmd/api/main.go --parseDependency --parseInternal --dir ./`
- Air config auto-runs `swag init` before each build

**Middleware Usage**:
- Global middleware in `api.go:mount()`
- Route-specific middleware can be applied in route groups
- Custom middlewares in `internal/middlewares/` follow chi's `func(next http.Handler) http.Handler` signature

**Error Responses**:
- Use `pkg/json.WriteError(w, statusCode, message)` for consistent error formatting
- Custom middlewares already use this pattern (see `rate_limit.go`)

### Dependencies
Core dependencies:
- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/joho/godotenv` - .env file loading
- `github.com/swaggo/http-swagger` - Swagger UI middleware
- `github.com/go-chi/httprate` - Rate limiting

Dev tools (install via go install):
- `github.com/air-verse/air` - Hot reload
- `github.com/pressly/goose/v3/cmd/goose` - Migrations
- `github.com/sqlc-dev/sqlc/cmd/sqlc` - SQL code generation
- `github.com/swaggo/swag/cmd/swag` - Swagger doc generation

## Environment Variables
Required variables (see `.env.example`):
- `GOOSE_DBSTRING` - PostgreSQL connection string (used by goose CLI and app)
- `GOOSE_DRIVER` - Database driver (always "postgres")
- `GOOSE_MIGRATION_DIR` - Migration directory path
- `JWT_SECRET` - Secret key for JWT signing (generate with `openssl rand -base64 32`)
- `ADDR` - Server listen address (default: ":8080")
- `REDIS_ADDR` - Redis server address (default: "localhost:6379")
- `DEFAULT_FALLBACK_URL` - Fallback URL for redirects

## Code Generation
The following directories contain auto-generated code and should not be edited manually:
- `internal/database/` - Generated by sqlc from SQL queries
- `docs/` - Generated by swag from handler annotations

To regenerate:
- **sqlc**: Run `sqlc generate` after modifying `sql/query.sql` or `sql/migrations/`
- **swagger**: Run `swag init -g cmd/api/main.go --parseDependency --parseInternal --dir ./` or let Air do it automatically

## Docker Compose Services
The `docker-compose.yaml` provides local development services:
- **postgres**: PostgreSQL 16 (port 5432)
  - Database: `addis_verify_test`
  - User/Password: `postgres/postgres`
  - Health checks enabled
- **redis**: Redis 7 (port 6379)
  - Data persisted to volume

Connection string example: `postgresql://postgres:postgres@localhost:5432/addis_verify_test?sslmode=disable`
