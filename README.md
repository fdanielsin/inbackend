# InDrive Backend

Go REST API for users and tasks, built with Echo, GORM and PostgreSQL.

## Requirements

- Go 1.26.8 (`go.mod` pins the toolchain; Go downloads it when needed)
- PostgreSQL 14+, or Docker
- Make
- Docker or Podman for the integration and end-to-end tests (Testcontainers)

## Quick start

Start a PostgreSQL database, point the server at it, and run it. Migrations are
applied on startup.

```bash
docker run --name inbackend-postgres -d -p 5432:5432 \
  -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=inbackend \
  postgres:16-alpine

export DATABASE_URL="postgres://postgres:postgres@localhost:5432/inbackend?sslmode=disable"
make run
```

The API listens on `http://localhost:8080`; check it with
`curl http://localhost:8080/ready`. In development, the contract is browsable at
`http://localhost:8080/docs`.

To run PostgreSQL and the backend together in containers:

```bash
docker compose up --build --wait
docker compose down -v
```

## Docker image

Every push to `develop` and every `v*` tag publishes a private image to
`ghcr.io/fdanielsin/inbackend`, tagged with the full commit sha, `develop` (branch
pushes) and the tag name (tag pushes). It is built for the future deployment of
the backend; no other project pulls it today. To run it against PostgreSQL:

```bash
echo "$GITHUB_TOKEN" | docker login ghcr.io -u <user> --password-stdin   # token with read:packages
docker run -p 8080:8080 -e DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=disable" \
  ghcr.io/fdanielsin/inbackend:<sha>
curl http://localhost:8080/openapi.yaml   # the contract of that commit
```

Use a sha tag to pin a contract version: `develop` moves with every push. Details
in [`docs/deployment.md`](docs/deployment.md).

## Configuration

The server reads these environment variables (see `.env.example`):

| Variable | Default | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | none (required) | PostgreSQL URL, `postgres://` or `postgresql://` |
| `PORT` | `8080` | HTTP port |
| `ENV` | `development` | `production` logs JSON at Info level; anything else logs text at Debug level |
| `CORS_ALLOWED_ORIGINS` | `*` | Allowed origins, comma-separated. Each entry must be `http://` or `https://` + lowercase host (+ `:port`), with no path, trailing slash, credentials, query, fragment, default port (80 or 443) or `*.` pattern; `*` is accepted only alone. A bad entry stops the server at startup. Required when `ENV=production` (exactly that value): absent, empty or containing `*` also stops it |
| `API_DOCS_ENABLED` | `true`, `false` when `ENV=production` | Serves the contract on `/openapi.yaml` and Swagger UI on `/docs` (loaded from a CDN) |

## Commands

| Command | Purpose |
| --- | --- |
| `make build` | Compile the binary |
| `make run` | Run the server (requires `DATABASE_URL`) |
| `make test` | All tests (needs Docker or Podman) |
| `make test-unit` | Unit tests only |
| `make test-integration` | Integration tests |
| `make test-e2e` | End-to-end tests |
| `make cover` | All tests, fails under 85 % coverage of `internal/` |
| `make lint` | golangci-lint and formatting check |
| `make lint-packages` | Refuse package names that clash with the standard library or a dependency (also part of `make lint`) |
| `make lint-api` | Lint the OpenAPI contract (requires Node) |
| `make lint-ci` | Lint the GitHub Actions workflows |
| `make vuln` | Scan dependencies with govulncheck |
| `make fmt` | Format the code and tidy modules |
| `make migrate-up`, `make migrate-down` | Apply or revert migrations (requires `DATABASE_URL`) |
| `make migrate-new name=<name>` | Create the next migration files |

## Documentation

- [Backend documentation](./docs/README.md): architecture, decisions, specs
- [OpenAPI contract](./docs/specs/api/openapi.yaml): the API reference
- [Adding an endpoint](./docs/adding-an-endpoint.md)
- [Deployment](./docs/deployment.md) and [troubleshooting](./docs/troubleshooting.md)
- [Changelog](./CHANGELOG.md)
- [Frontend repository](https://github.com/fdanielsin/infrontend)
- Workspace docs, outside this repository: `../docs/README.md`
