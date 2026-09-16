package db

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

func StartTestContainer(ctx context.Context) (*TestDB, error) {
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

func (t *TestDB) Stop(ctx context.Context) error {
	if t.Container != nil {
		return t.Container.Terminate(ctx)
	}
	return nil
}

func SetupTestDB(ctx context.Context) (*TestDB, error) {
	// Check if DATABASE_URL is set (used by GitHub Actions service)
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return &TestDB{
			Container: nil, // No container to stop
			DSN:       dbURL,
		}, nil
	}

	// Otherwise, use Testcontainers
	testDB, err := StartTestContainer(ctx)
	if err != nil {
		return nil, err
	}
	return testDB, nil
}
