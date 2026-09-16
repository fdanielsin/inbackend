.PHONY: help build run test test-unit test-integration test-e2e lint fmt clean docker-build docker-run

help:
	@echo "🔧 Available commands:"
	@echo "  make build           - Build Go binary"
	@echo "  make run             - Run server (requires DATABASE_URL)"
	@echo "  make test            - Run all tests"
	@echo "  make test-unit       - Run unit tests"
	@echo "  make test-integration - Run integration tests with testcontainers"
	@echo "  make test-e2e        - Run E2E tests"
	@echo "  make lint            - Run golangci-lint"
	@echo "  make fmt             - Format code"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make docker-build    - Build Docker image with Podman"
	@echo "  make docker-run      - Run Docker container with Podman"

build:
	@echo "📦 Building..."
	CGO_ENABLED=1 go build -o app .

run:
	@echo "🚀 Starting server..."
	go run main.go

test:
	@echo "🧪 Running all tests..."
	go test -v -race -cover ./...

test-unit:
	@echo "🧪 Running unit tests..."
	go test -v -race -short ./internal/...

test-integration:
	@echo "🧪 Running integration tests with testcontainers..."
	go test -v -race -run Integration ./tests/...

test-e2e:
	@echo "🧪 Running E2E tests..."
	go test -v -run E2E ./tests/...

lint:
	@echo "🔍 Linting with golangci-lint..."
	golangci-lint run ./...

fmt:
	@echo "✨ Formatting code..."
	gofmt -w .
	go mod tidy

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
