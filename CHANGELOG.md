# Changelog

## Unreleased

### Phase 6 — Architecture Restructuring & Logging slog [2026-09-20]

- **infrastructure**: Moved `internal/db` to `internal/infrastructure/database/` for proper Clean Architecture alignment.
- **logger**: Added slog integration with zap delegation via `internal/logger/slog_handler.go`.
- **logger**: InitLogger() now initializes both zap and slog, with slog as default.
- **controllers**: Added structured slog logging to all 4 HTTP controllers (UserController, TaskController, StatsController, HealthController).
- **infrastructure**: Added structured slog logging to UserRepository for persistence operations.
- **tests**: Centralized all integration/E2E tests in `tests/` folder, removed obsolete `e2e_test.go`.
- **tests**: Unit tests remain in `internal/` packages following Go conventions.
- All 38 tests passing (domain 9 + application 13 + infrastructure 6 + integration 9 + E2E 1).
- Code formatting validated and applied.

### Phase 5 — Validation & End-to-End Tests [2026-09-20]

- **tests**: Added comprehensive controller integration tests with SQLite in-memory.
- **tests**: UserController tests (Create, Get, List, validation error handling).
- **tests**: TaskController tests (Create, Update, Delete, ListByUser).
- **tests**: StatsController and HealthController tests.
- **tests**: Complete E2E workflow test covering full user and task lifecycle.
- All 38 tests passing (28 unit + 9 integration + 1 E2E).
- Architecture validated: controllers → application → domain → infrastructure.
- Code formatting verified and applied.
- Build and vet checks passing.

### Phase 4 — Controllers HTTP & Dependency Injection [2026-09-20]

- **controllers/http**: Created UserController with routes for Create, Get, List users.
- **controllers/http**: Created TaskController with routes for Create, Update, Delete, List by user.
- **controllers/http**: Created HealthController for service health checks.
- **controllers/http**: Created StatsController for aggregated statistics.
- **shared**: Added error mapping from domain errors to HTTP responses.
- **shared**: Added APIResponse helpers for consistent response formatting.
- **main.go**: Refactored to implement dependency injection pattern.
- Repositories (PostgreSQL) injected into commands/queries at startup.
- All routes preserved (9 endpoints), same response formats, same HTTP codes.

### Phase 3 — Infrastructure & PostgreSQL/GORM [2026-09-20]

- **infrastructure/persistence/postgres**: Created UserRepository implementation with context support.
- **infrastructure/persistence/postgres**: Created TaskRepository implementation with full CRUD operations.
- **infrastructure**: Repositories map domain entities to GORM models with explicit conversion.
- **infrastructure**: Error handling maps gorm.ErrRecordNotFound to domain errors.
- Tests for repositories using SQLite in-memory database (no external dependencies).
- All 28 tests passing (domain 9 + application 13 + infrastructure 6).

### Phase 2 — Application Layer & CQRS [2026-09-20]

- **application/user**: Added command CreateUser with validation and repository integration.
- **application/user**: Added queries GetUser and ListUsers with mock testing.
- **application/task**: Added commands CreateTask, UpdateTask, DeleteTask with repository integration.
- **application/task**: Added queries ListUserTasks and GetStats with repository aggregation.
- **domain/task**: Created Task domain entity, repository interface, and error definitions.
- **domain/task**: Added unit tests for Task entity validation.
- CQRS applicatif structure with 16+ unit tests passing.

### Phase 1 — Domain Foundations [2026-09-20]

- **domain/user**: Created User domain entity, repository interface, and error definitions.
- **domain/user**: Added unit tests for User entity validation.
- Documentation updated to reflect architecture progression.
- Handoff document updated with Phase 1 completion status.

### Previous

- Documentation structure established for architecture, specifications,
  decisions, roadmap and agent handoff.
- Backend architecture migration planned around domain, application,
  controllers and infrastructure.

## 1.0.0

- Added the stats endpoint.
