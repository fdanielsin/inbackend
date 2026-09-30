.PHONY: help build run test test-unit test-integration test-e2e lint lint-api fmt vuln clean docker-build docker-run

# Outils épinglés, lancés avec `go run` pour ne pas modifier go.mod (ADR 0006).
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@v1.8.0
REDOCLY := npx --yes @redocly/cli@2.55.0

help:
	@echo "🔧 Available commands:"
	@echo "  make build           - Build Go binary"
	@echo "  make run             - Run server (requires DATABASE_URL)"
	@echo "  make test            - Run all tests"
	@echo "  make test-unit       - Run unit tests"
	@echo "  make test-integration - Run integration tests with testcontainers"
	@echo "  make test-e2e        - Run E2E tests"
	@echo "  make lint            - Run golangci-lint and check formatting (gofumpt, goimports)"
	@echo "  make lint-api        - Lint the OpenAPI contract with Redocly (requires Node)"
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
	go test -v -race -cover ./...

test-unit:
	@echo "🧪 Running unit tests..."
	go test -v -race ./tests/units/...

test-integration:
	@echo "🧪 Running integration tests with Testcontainers..."
	go test -v -race ./tests/integration/...

test-e2e:
	@echo "🧪 Running E2E tests..."
	go test -v ./tests/e2e/...

lint:
	@echo "🔍 Linting and checking formatting..."
	$(GOLANGCI_LINT) run ./...
	$(GOLANGCI_LINT) fmt --diff ./...

lint-api:
	@echo "📜 Linting OpenAPI contract..."
	$(REDOCLY) lint --config docs/specs/api/redocly.yaml docs/specs/api/openapi.yaml

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
