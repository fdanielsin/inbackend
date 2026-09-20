package persistence

import (
	"context"
	"fmt"
	"testing"

	"github.com/fdanielsin/inbackend/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgreSQL container for integration tests
var pgContainer testcontainers.Container

// setupTestDB creates a PostgreSQL database for integration tests via Testcontainers
// GIVEN: Testing database needed via Testcontainers
// WHEN: setupTestDB() called in test
// THEN: PostgreSQL container starts and returns configured GORM DB instance
func setupTestDB(t *testing.T) *gorm.DB {
	ctx := context.Background()

	// Start PostgreSQL container
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
	require.NoError(t, err, "failed to create PostgreSQL container")

	// Store container reference for cleanup
	pgContainer = container

	// Get container port
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err, "failed to get container port")

	// Build DSN
	dsn := fmt.Sprintf("host=localhost port=%s user=testuser password=testpass dbname=testdb sslmode=disable", port.Port())

	// Connect to PostgreSQL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "failed to connect to PostgreSQL")

	// Run migrations
	err = db.AutoMigrate(&models.User{}, &models.Task{})
	require.NoError(t, err, "failed to migrate database")

	// Cleanup: deferred container termination
	t.Cleanup(func() {
		if pgContainer != nil {
			_ = pgContainer.Terminate(ctx)
		}
	})

	return db
}
