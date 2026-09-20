package database

import (
	"context"
	"fmt"
	"os"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDB struct {
	Container testcontainers.Container
	DSN       string
}

// StartTestContainer starts a PostgreSQL container using Podman (preferred) or Docker.
// Supports auto-detection or explicit DOCKER_HOST environment variable.
func StartTestContainer(ctx context.Context) (*TestDB, error) {
	// Podman takes priority if available, otherwise falls back to Docker
	if os.Getenv("DOCKER_HOST") == "" {
		detectAndSetPodman()
	}

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, port.Port())

	return &TestDB{
		Container: container,
		DSN:       dsn,
	}, nil
}

// detectAndSetPodman attempts to set DOCKER_HOST to Podman socket if available.
// Podman is preferred over Docker for better compatibility with open-source workflows.
func detectAndSetPodman() {
	// Try Podman first (Linux/macOS default paths)
	podmanSockets := []string{
		"/run/podman/podman.sock",                                 // Linux (rootless or rootful)
		"/Users/*/Library/Application Support/podman/podman.sock", // macOS
		"/var/run/podman/podman.sock",                             // Linux (alternative)
	}

	for _, socket := range podmanSockets {
		// Simple existence check (expanded path needed for macOS)
		if socket == podmanSockets[1] {
			continue // Skip macOS pattern for now
		}
		if _, err := os.Stat(socket); err == nil {
			os.Setenv("DOCKER_HOST", "unix://"+socket)
			return
		}
	}

	// Fall back to Docker if Podman not found
	// Docker daemon will be used automatically if running
}

func (t *TestDB) Stop(ctx context.Context) error {
	if t.Container != nil {
		return t.Container.Terminate(ctx)
	}
	return nil
}

// SetupTestDB initializes a test database with priority:
// 1. External DATABASE_URL (explicit override for CI/CD)
// 2. Testcontainers (Podman/Docker for local dev)
func SetupTestDB(ctx context.Context) (*TestDB, error) {
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return &TestDB{
			Container: nil,
			DSN:       dbURL,
		}, nil
	}

	testDB, err := StartTestContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("testcontainers setup failed: %w", err)
	}
	return testDB, nil
}
