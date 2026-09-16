# inBackend - Go + Echo API

Senior-grade Go backend with Echo framework, Docker containerization, and PostgreSQL integration.

## Stack

- **Framework**: Echo (lightweight, high-performance)
- **Container**: Docker + Docker Compose
- **Database**: PostgreSQL (optional, configured in docker-compose)
- **Tools**: Go 1.23+

## Features

- REST API with `/health` and `/api/v1/status` endpoints
- CORS middleware configured
- Structured logging with Echo middleware
- Multi-stage Docker builds for optimized images
- Docker Compose setup with optional PostgreSQL service

## Quick Start

### Local Development

```bash
# Install dependencies
go mod download

# Run server
go run main.go
```

Server runs on `http://localhost:8080`

### Docker

```bash
# Build and run
docker-compose up --build

# With PostgreSQL
docker-compose --profile with-db up --build
```

## Environment Variables

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

## API Endpoints

- `GET /health` - Health check
- `GET /api/v1/status` - Service status

## Next Steps

- Add database models & migrations
- Implement auth (JWT/OAuth2)
- Add OpenAPI/Swagger documentation
- Set up structured logging (e.g., zap, slog)
- Add unit & integration tests
- Configure CI/CD pipeline
