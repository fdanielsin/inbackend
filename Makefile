.PHONY: help build run test test-unit test-integration test-e2e cover mutate migrate-up migrate-down migrate-force migrate-new lint lint-api lint-ci fmt vuln clean docker-build docker-run

# Outils épinglés, lancés avec `go run` pour ne pas modifier go.mod (ADR 0006).
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@v1.8.0
ACTIONLINT := go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
MIGRATE := go run -tags pgx5 github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1
MIGRATIONS_DIR := internal/infrastructure/pgsetup/migrations
MIGRATE_DATABASE_URL = $(patsubst postgres://%,pgx5://%,$(patsubst postgresql://%,pgx5://%,$(DATABASE_URL)))
COVERAGE_MIN ?= 85
GREMLINS := go run github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0
MUTATION_BASE ?= develop
REDOCLY := npx --yes @redocly/cli@2.55.0

help:
	@echo "🔧 Available commands:"
	@echo "  make build           - Build Go binary"
	@echo "  make run             - Run server (requires DATABASE_URL)"
	@echo "  make test            - Run all tests"
	@echo "  make test-unit       - Run unit tests"
	@echo "  make test-integration - Run integration tests with testcontainers"
	@echo "  make test-e2e        - Run E2E tests (requires Docker or Podman)"
	@echo "  make migrate-up      - Apply pending migrations (requires DATABASE_URL)"
	@echo "  make migrate-down    - Revert the last migration (requires DATABASE_URL)"
	@echo "  make migrate-force version=N - Clear the dirty flag after a failed migration (N = last version fully applied)"
	@echo "  make migrate-new name=xxx - Create the next up/down migration files"
	@echo "  make cover           - Run all tests with coverage of internal/ and fail under COVERAGE_MIN (default 85)"
	@echo "  make mutate          - Mutation testing on the diff against MUTATION_BASE (default develop), not run in CI"
	@echo "  make lint            - Run golangci-lint and check formatting (gofumpt, goimports)"
	@echo "  make lint-api        - Lint the OpenAPI contract with Redocly (requires Node)"
	@echo "  make lint-ci         - Lint GitHub Actions workflows with actionlint"
	@echo "  make fmt             - Format code and tidy modules"
	@echo "  make vuln            - Scan dependencies with govulncheck"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make docker-build    - Build Docker image with Podman"
	@echo "  make docker-run      - Run Docker container with Podman"

build:
	@echo "📦 Building..."
	CGO_ENABLED=0 go build -o app .

run:
	@echo "🚀 Starting server..."
	go run main.go

test:
	@echo "🧪 Running all tests..."
	go test -v -race -shuffle=on -cover ./...

test-unit:
	@echo "🧪 Running unit tests..."
	go test -v -race -shuffle=on ./tests/units/...

test-integration:
	@echo "🧪 Running integration tests with Testcontainers..."
	go test -v -race -shuffle=on ./tests/integration/...

test-e2e:
	@echo "🧪 Running E2E tests with Testcontainers..."
	go test -v -shuffle=on ./tests/e2e/...

cover:
	@echo "📊 Running all tests with coverage (requires Docker/Podman)..."
	go test -race -shuffle=on -coverpkg=./internal/... -coverprofile=coverage.out ./...
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {sub("%", "", $$NF); print $$NF}'); \
	echo "Coverage: $$total% (minimum $(COVERAGE_MIN)%)"; \
	awk -v total="$$total" -v min="$(COVERAGE_MIN)" 'BEGIN { exit (total + 0 >= min + 0) ? 0 : 1 }' \
		|| { echo "❌ Coverage $$total% is below the minimum of $(COVERAGE_MIN)%"; exit 1; }

mutate:
	@echo "🧬 Mutation testing on the diff against $(MUTATION_BASE)..."
	$(GREMLINS) unleash --integration --coverpkg=./internal/... -D $(MUTATION_BASE)

migrate-up:
	@echo "⬆️ Applying migrations..."
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DATABASE_URL)" up

migrate-down:
	@echo "⬇️ Reverting the last migration..."
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DATABASE_URL)" down 1

migrate-force:
	@test -n "$(version)" || (echo "usage: make migrate-force version=N" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DATABASE_URL)" force $(version)

migrate-new:
	@test -n "$(name)" || (echo "usage: make migrate-new name=add_something" && exit 1)
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) -seq -digits 6 $(name)

lint:
	@echo "🔍 Linting and checking formatting..."
	$(GOLANGCI_LINT) run ./...
	$(GOLANGCI_LINT) fmt --diff ./...

lint-api:
	@echo "📜 Linting OpenAPI contract..."
	$(REDOCLY) lint --config docs/specs/api/redocly.yaml docs/specs/api/openapi.yaml

lint-ci:
	@echo "⚙️ Linting GitHub Actions workflows..."
	$(ACTIONLINT) .github/workflows/*.yml

fmt:
	@echo "✨ Formatting code..."
	$(GOLANGCI_LINT) fmt ./...
	go mod tidy

vuln:
	@echo "🛡️ Scanning dependencies..."
	$(GOVULNCHECK) ./...

clean:
	@echo "🧹 Cleaning..."
	rm -f app
	rm -f *.db
	rm -rf dist/

docker-build:
	@echo "🐳 Building with Podman..."
	podman build -t inbackend:latest .

docker-run:
	@echo "🐳 Running with Podman..."
	podman-compose -f docker-compose.yml up --build
