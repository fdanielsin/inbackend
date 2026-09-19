# 🚀 CI/CD Pipeline - Backend (Inbackend)

## Overview
**7-stage automated CI/CD pipeline** for Go backend with comprehensive testing, security checks, and coverage validation.

```
FEATURE BRANCH
    ↓
┌─────────────────────────────────────┐
│ 1️⃣  LINT & FORMAT (gofmt + vet)    │ ✅ 5s
├─────────────────────────────────────┤
│ ✓ Code format validation            │
│ ✓ Go vet analysis                   │
│ ✓ Module tidiness check             │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│ 2️⃣  BUILD                           │ ✅ 10s
├─────────────────────────────────────┤
│ ✓ Compile Go binary                 │
│ ✓ Dependency download               │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│ 3️⃣  UNIT TESTS                      │ ✅ 30s
├─────────────────────────────────────┤
│ ✓ ./internal/... tests              │
│ ✓ No external dependencies          │
│ ✓ Race condition detection          │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│ 4️⃣  SECURITY CHECK                  │ ✅ 20s
├─────────────────────────────────────┤
│ ✓ Vulnerable dependencies scan      │
│ ✓ Nancy security audit              │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│ 5️⃣  INTEGRATION TESTS               │ ✅ 60s
├─────────────────────────────────────┤
│ ✓ PostgreSQL tests                  │
│ ✓ Database migrations               │
│ ✓ Handler integration               │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│ 6️⃣  E2E TESTS                       │ ✅ 30s
├─────────────────────────────────────┤
│ ✓ Full workflow tests               │
│ ✓ User + Task CRUD operations       │
│ ✓ API endpoint validation           │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│ 7️⃣  COVERAGE CHECK                  │ ✅ 20s
├─────────────────────────────────────┤
│ ✓ Generate coverage report          │
│ ✓ Threshold: 80% (configurable)     │
│ ✓ Upload to Codecov                 │
└─────────────────────────────────────┘
    ↓
    🎉 READY TO MERGE
```

## Configuration

| Parameter | Value | Notes |
|-----------|-------|-------|
| Go Version | 1.25.0 | Latest stable |
| Coverage Threshold | 80% | Fail if below |
| Test Timeout | 30s-60s | Per stage |
| Database | PostgreSQL 16 | Alpine image |

## Stages Detail

### 1️⃣ Lint & Format
```bash
gofmt -s -l .              # Format check
go vet ./...               # Vet analysis
go mod tidy                # Module sync
```

### 2️⃣ Build
```bash
go mod download            # Download deps
go build -v -o app .       # Compile
```

### 3️⃣ Unit Tests
```bash
go test -v -race -short ./internal/... -timeout 30s
```

### 4️⃣ Security Check
```bash
nancy sleuth               # Vulnerable deps
```

### 5️⃣ Integration Tests
```bash
go test -v -race ./tests/... -timeout 60s -run '!E2E'
# Runs with PostgreSQL service
```

### 6️⃣ E2E Tests
```bash
go test -v -race ./tests/... -timeout 60s -run 'E2E'
# Full stack workflow tests
```

### 7️⃣ Coverage Check
```bash
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | tail -1
# Check threshold: COVERAGE >= 80%
```

## Git Integration

### Triggered On:
- ✅ Push to `main` or `develop`
- ✅ Push to `feature/**` branches
- ✅ Push to `bugfix/**` branches
- ✅ Pull requests to `main` or `develop`

### Branch Workflow:
```
feature/new-feature
    ↓
    Run CI/CD (all 7 stages)
    ↓
    Pull Request → Code Review
    ↓
    Merge to develop (if approved)
    ↓
    Run CI/CD again
    ↓
    Manual QA in staging
    ↓
    Merge to main (production)
```

## Local Development

Run locally before pushing:

```bash
# Format & lint
go fmt ./...
go vet ./...

# Unit tests only
go test -v -short ./internal/...

# With integration (requires Docker)
docker-compose up -d postgres
export DATABASE_URL="postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
go test -v ./tests/... -run '!E2E'

# Full E2E
go test -v ./tests/... -run 'E2E'

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Performance

| Stage | Time | Parallel |
|-------|------|----------|
| Lint | 5s | - |
| Build | 10s | - |
| Unit Tests | 30s | - |
| Security | 20s | ✅ Parallel with Build |
| Integration | 60s | - |
| E2E | 30s | - |
| Coverage | 20s | - |
| **TOTAL** | **~3-4 min** | - |

## Troubleshooting

### Coverage below threshold?
- Add test cases to `tests/`
- Run `go test -coverprofile=coverage.out ./...` locally
- Check `go tool cover -html=coverage.out`

### Security scan failing?
- Run `nancy sleuth` locally
- Check for outdated dependencies: `go list -u -m all`
- Update: `go get -u all`

### E2E tests failing?
- Verify PostgreSQL is running
- Check `DATABASE_URL` is set correctly
- Review test data cleanup (DELETE statements)

## Merge Requirements

✅ **All 7 stages MUST pass** before merging:
1. ✅ Lint & Format
2. ✅ Build
3. ✅ Unit Tests
4. ✅ Security
5. ✅ Integration
6. ✅ E2E
7. ✅ Coverage (≥80%)

## Future Enhancements

- [ ] Docker artifact build & push (Docker registry)
- [ ] Automated deployment to staging
- [ ] Performance benchmark tracking
- [ ] Code quality metrics (SonarQube)
- [ ] Automated rollback on deployment failure
- [ ] Slack/Discord notifications on failure

---

**Last Updated:** 2026-09-16
**Go Version:** 1.25.0
**Status:** ✅ Fully Operational
