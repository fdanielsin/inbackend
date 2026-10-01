# InDrive Backend

Professional Go + Echo + PostgreSQL REST API with comprehensive testing.

## 🏗️ Architecture

```
inbackend/
├── main.go                 # Entry point: config, database, server start
├── docs/                   # Project architecture and workflow documentation
├── CHANGELOG.md            # Concise project history
├── internal/
│   ├── app/                # Composition root: app.NewRouter wires everything
│   ├── domain/             # Business domain entities and interfaces
│   │   └── user/           # User domain (entity, errors, repository)
│   ├── application/        # Use cases (CQRS commands and queries) [Phase 2]
│   ├── controllers/        # HTTP controllers [Phase 4]
│   ├── infrastructure/     # Database, persistence, logger, and technical details
│   └── shared/              # HTTP responses and cross-cutting errors
├── tests/                  # Unit, integration, E2E tests
├── Makefile                # Development automation
└── .github/workflows/      # CI/CD pipeline (ci.yml)
```

## 🔧 Stack

- **Framework**: Echo (high-performance HTTP)
- **Database**: PostgreSQL with GORM ORM
- **Logging**: slog structured logging delegated to zap
- **Testing**: Testcontainers, stdlib testing
- **Security**: govulncheck dependency scanning in the backend CI/CD workflow
- **Container**: Docker + Podman support

## ✨ Features

- ✅ Full REST API (Users, Tasks CRUD)
- ✅ PostgreSQL with versioned migrations (golang-migrate, ADR 0007)
- ✅ Structured logging with slog delegated to Zap
- ✅ CORS middleware configured
- ✅ Comprehensive test suite (unit, integration, E2E)
- ✅ Testcontainers for isolated database tests
- ✅ Multi-stage Docker build (image ~36 MB, non-root user, static binary)
- ✅ GitHub Actions CI/CD pipelines
- ✅ Code quality & security scanning

## 📋 Requirements

- Go 1.26.8 (`go.mod` pins the toolchain; Go downloads it automatically)
- PostgreSQL 14+ (or Docker/Podman)
- Make
- **Docker or Podman** (required for the Testcontainers integration and E2E tests)
  - Podman is preferred (open-source, rootless-by-default, better compatibility)

## 🚀 Quick Start

### Setup Database

**Option A: Docker (Recommended)**
```bash
docker run --name inbackend-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=testdb \
  -p 5432:5432 \
  -d postgres:15-alpine
```

**Option B: Podman**
```bash
podman machine start
podman run --name inbackend-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=testdb \
  -p 5432:5432 \
  -d postgres:15-alpine
```

**Option C: Local PostgreSQL**
```bash
createdb testdb
```

### Run Server

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable"
export PORT=8080
export ENV=development

go run main.go
```

Server runs at `http://localhost:8080`

## 📚 Makefile Commands

| Command | Purpose |
|---------|---------|
| `make build` | Compile binary |
| `make run` | Run with `go run` |
| `make test` | All tests (unit + integration + E2E) |
| `make test-unit` | Unit tests only |
| `make test-integration` | Integration tests with Testcontainers |
| `make cover` | All tests with coverage of `internal/`, fails under 85 % (requires Docker/Podman; same command as CI) |
| `make test-e2e` | End-to-end tests (PostgreSQL via Testcontainers, requires Docker/Podman) |
| `make lint` | golangci-lint and formatting check (gofumpt, goimports) |
| `make lint-api` | Redocly lint of the OpenAPI contract (requires Node) |
| `make lint-ci` | actionlint on the GitHub Actions workflows |
| `make vuln` | govulncheck dependency scan |
| `make fmt` | Auto-format code and tidy modules |
| `make docker-build` | Build Docker image |
| `make docker-run` | Run container |

## 🔌 API Endpoints

### Users
- `GET /api/v1/users` - List users
- `POST /api/v1/users` - Create user
  ```json
  {"name": "John", "email": "john@example.com"}
  ```
- `GET /api/v1/users/:id` - Get user
- `GET /api/v1/users/:id/tasks` - List tasks for a user

### Tasks
- `POST /api/v1/tasks` - Create task
  ```json
  {"user_id": 1, "title": "Task", "description": "...", "completed": false}
  ```
- `PUT /api/v1/tasks/:id` - Update task
- `DELETE /api/v1/tasks/:id` - Delete task
- `GET /api/v1/stats` - Get aggregate statistics

### Health
- `GET /health` - Liveness (no database check)
- `GET /ready` - Readiness (database reachable within 2 s, otherwise 503 `DATABASE_UNAVAILABLE`)

## Documentation

- [Backend documentation](./docs/README.md)
- [API documentation (OpenAPI 3.1)](./docs/specs/api/openapi.yaml)
- [Global workspace context](../docs/README.md)

## 🔐 Environment Variables

Create `.env.local`:
```bash
DATABASE_URL="postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable"
PORT=8080
ENV=development
```

See `.env.example` for all options.

## 🧪 Testing

### Run Tests Locally

```bash
# Requires Docker/Podman running
make test
```

Tests will:
1. Spin up PostgreSQL container via Testcontainers
2. Run migrations
3. Test CRUD operations
4. Validate error handling
5. Clean up containers

### View Coverage

```bash
go test -cover ./...
```

## 🐳 Docker

### Build Image
```bash
make docker-build
# Or manually:
docker build -t inbackend:latest .
```

### Run Container
```bash
docker run \
  -e DATABASE_URL="postgres://user:password@host:5432/dbname?sslmode=disable" \
  -e PORT=8080 \
  -p 8080:8080 \
  inbackend:latest
```

### With Docker Compose
```bash
docker compose up --build --wait   # PostgreSQL 16 + backend, waits for healthchecks
curl http://localhost:8080/health
curl http://localhost:8080/ready
docker compose down -v             # stops and deletes the database volume
```

Compose uses development-only credentials (`inbackend`/`inbackend`), overridable
with `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` and `PORT`. The CI
`docker` job runs this same sequence. Compose is tested with Docker only; the
Podman path (`make docker-run`) is not tested.

## 📦 Deployment

### Standalone Binary
```bash
CGO_ENABLED=0 go build -o app .
DATABASE_URL=... ./app
```

### Production Environment
```bash
export ENV=production
export DATABASE_URL="postgres://user:pass@prod-db.example.com/appdb?sslmode=require"
./app
```

## 🔄 GitHub Actions CI/CD

### ci.yml - Full Pipeline
Runs on push to `main`/`develop`:
- ✅ Build verification
- ✅ Unit tests
- ✅ Integration tests with Testcontainers
- ✅ E2E tests
- ✅ Coverage report (upload to Codecov)
- ✅ Docker image build

The same `ci.yml` workflow also runs code quality checks:
Runs on every push:
- ✅ golangci-lint and formatting check (`make lint`)
- ✅ OpenAPI contract lint with Redocly (`make lint-api`)
- ✅ `go mod tidy` consistency
- ✅ Dependency vulnerability scan with govulncheck (`make vuln`, blocking)

## 🔗 Frontend Integration

Frontend (Svelte) API calls:
```typescript
const response = await fetch('http://localhost:8080/api/v1/users');
```

Frontend Repo: https://github.com/fdanielsin/infrontend

## 🛠️ Development Notes

- **Migrations**: versioned SQL in `internal/infrastructure/database/migrations/`, applied on startup (ADR 0007); `make migrate-new name=...` creates the next one
- **Error Handling**: Unified `APIResponse` in `internal/shared/response.go`; every error carries a stable `error_code`, all mapped in one table in `internal/shared/errors.go` (codes listed in `docs/specs/api/openapi.yaml`)
- **CORS**: `CORS_ALLOWED_ORIGINS` (comma-separated, `*` by default, restrict it in production), see `internal/app/router.go`
- **Requests**: each response carries `X-Request-Id` (the client value is kept if it is 1-64 chars of `A-Za-z0-9._-`); bodies above 1 MiB get 413; unknown routes and methods use the error envelope (`ROUTE_NOT_FOUND`, `METHOD_NOT_ALLOWED`)
- **Shutdown**: SIGINT/SIGTERM stop the server gracefully (10 s for in-flight requests)
- **Git hook**: `git config core.hooksPath .githooks` enables a blocking pre-push (lint, build, unit tests, no Docker needed)
- **Logging**: Development (pretty) vs Production (structured JSON)
- **Testing**: Testcontainers start one PostgreSQL container per test package; tables are truncated before each test

## ❓ Troubleshooting

### "connection refused"
```bash
# Verify database
docker ps | grep postgres
# Or start PostgreSQL using the database setup instructions above.
```

### "SSL error"
Add `?sslmode=disable` to DATABASE_URL for local dev. `DATABASE_URL` must be a `postgres://` or `postgresql://` URL (the key=value format is refused at startup).

### Tests timeout
Increase timeout:
```bash
go test -timeout 120s ./tests/...
```

### Testcontainers not working
Ensure Docker daemon is running:
```bash
docker ps
# Or with Podman:
podman machine start
```

## 📖 License

MIT
