# InDrive Backend

Professional Go + Echo + PostgreSQL REST API with comprehensive testing.

## 🏗️ Architecture

```
inbackend/
├── main.go                 # Entry point, server setup
├── internal/
│   ├── db/                 # GORM migrations, DB initialization
│   ├── handlers/           # HTTP handlers (CRUD operations)
│   ├── models/             # Data structures (User, Task)
│   └── logger/             # Zap logger setup
├── tests/                  # Unit, integration, E2E tests
├── Makefile                # Development automation
└── .github/workflows/      # CI/CD pipelines (go.yml, quality.yml)
```

## 🔧 Stack

- **Framework**: Echo (high-performance HTTP)
- **Database**: PostgreSQL with GORM ORM
- **Logging**: Zap structured logging
- **Testing**: Testcontainers, stdlib testing
- **Security**: Trivy scanning in CI/CD
- **Container**: Docker + Podman support

## ✨ Features

- ✅ Full REST API (Users, Tasks CRUD)
- ✅ PostgreSQL with GORM auto-migrations
- ✅ Structured logging with Zap
- ✅ CORS middleware configured
- ✅ Comprehensive test suite (unit, integration, E2E)
- ✅ Testcontainers for isolated database tests
- ✅ Multi-stage Docker builds (24MB)
- ✅ GitHub Actions CI/CD pipelines
- ✅ Code quality & security scanning

## 📋 Requirements

- Go 1.21+
- PostgreSQL 14+ (or Docker)
- Make
- Optional: Docker or Podman for containerized tests

## 🚀 Quick Start

### Setup Database

**Option A: Docker (Recommended)**
```bash
docker-compose up -d postgres
```

**Option B: Podman**
```bash
podman machine start
podman-compose up -d postgres
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
| `make test-e2e` | End-to-end tests |
| `make lint` | Format & vet checks |
| `make fmt` | Auto-format code |
| `make docker-build` | Build Docker image |
| `make docker-run` | Run container |

## 🔌 API Endpoints

### Users
- `POST /users` - Create user
  ```json
  {"name": "John", "email": "john@example.com"}
  ```
- `GET /users` - List users
- `GET /users/:id` - Get user
- `PUT /users/:id` - Update user
- `DELETE /users/:id` - Delete user

### Tasks
- `POST /tasks` - Create task
  ```json
  {"user_id": 1, "title": "Task", "description": "...", "completed": false}
  ```
- `GET /tasks?user_id=1` - List tasks for user
- `GET /tasks/:id` - Get task
- `PUT /tasks/:id` - Update task
- `DELETE /tasks/:id` - Delete task

### Health
- `GET /health` - Server status

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
  -e DATABASE_URL="postgres://db:5432/testdb" \
  -e PORT=8080 \
  -p 8080:8080 \
  inbackend:latest
```

### With Docker Compose
```bash
docker-compose up --build
```

## 📦 Deployment

### Standalone Binary
```bash
CGO_ENABLED=1 go build -o app .
DATABASE_URL=... ./app
```

### Production Environment
```bash
export ENV=production
export DATABASE_URL="postgres://user:pass@prod-db.example.com/appdb?sslmode=require"
./app
```

## 🔄 GitHub Actions CI/CD

### go.yml - Full Pipeline
Runs on push to `main`/`develop`:
- ✅ Build verification
- ✅ Unit tests
- ✅ Integration tests with Testcontainers
- ✅ E2E tests
- ✅ Coverage report (upload to Codecov)
- ✅ Docker image build

### quality.yml - Code Quality
Runs on every push:
- ✅ Format checks (gofmt)
- ✅ Static analysis (go vet, go mod tidy)
- ✅ Security scanning (Trivy)
- ✅ Uploads results to GitHub Security tab

## 🔗 Frontend Integration

Frontend (Svelte) API calls:
```typescript
const response = await fetch('http://localhost:8080/users');
```

Frontend Repo: https://github.com/fdanielsin/infrontend

## 🛠️ Development Notes

- **GORM Auto-Migration**: Runs on startup, see `internal/db/db.go`
- **Error Handling**: Unified `ApiResponse<T>` in `internal/models/models.go`
- **CORS**: Configured in `main.go`, adjust origins as needed
- **Logging**: Development (pretty) vs Production (structured JSON)
- **Testing**: Testcontainers spin up isolated PostgreSQL per test

## ❓ Troubleshooting

### "connection refused"
```bash
# Verify database
docker ps | grep postgres
# Or start it
docker-compose up -d postgres
```

### "SSL error"
Add `?sslmode=disable` to DATABASE_URL for local dev.

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
