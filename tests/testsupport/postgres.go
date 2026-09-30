// Package testsupport starts the PostgreSQL container shared by the integration
// and E2E test packages.
package testsupport

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	pgimpl "github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver used by the container readiness check
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Postgres is one PostgreSQL container, started once per test package from TestMain.
type Postgres struct {
	// DSN is empty when the container could not be started.
	DSN       string
	container testcontainers.Container
	startErr  error
}

// StartPostgres starts the container. A failure is kept and reported by each test
// through Require, so that TestMain itself never aborts the package.
func StartPostgres() *Postgres {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForSQL("5432/tcp", "pgx", func(host string, port network.Port) string {
			return fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())
		}).WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	pg := &Postgres{container: container, startErr: err}
	if err != nil {
		return pg
	}

	host, err := container.Host(ctx)
	if err != nil {
		pg.startErr = err
		return pg
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		pg.startErr = err
		return pg
	}
	pg.DSN = fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())
	return pg
}

// Available reports whether the container started.
func (p *Postgres) Available() bool {
	return p.startErr == nil
}

// Terminate stops the container, if it was created.
func (p *Postgres) Terminate() {
	if p.container != nil {
		_ = p.container.Terminate(context.Background())
	}
}

// Require skips the test locally when no container provider is available, but
// fails it in CI (CI=true, set by GitHub Actions): an unavailable PostgreSQL must
// never turn a test suite green (Q-09).
func (p *Postgres) Require(t testing.TB) {
	t.Helper()
	if p.startErr == nil {
		return
	}
	if strings.Contains(p.startErr.Error(), "provider") && os.Getenv("CI") != "true" {
		t.Skipf("PostgreSQL tests require a container provider: %v", p.startErr)
	}
	t.Fatalf("PostgreSQL container unavailable: %v", p.startErr)
}

// Open opens a new connection, closed at the end of the test.
func (p *Postgres) Open(t testing.TB) *gorm.DB {
	t.Helper()
	p.Require(t)

	db, err := gorm.Open(postgres.Open(p.DSN), &gorm.Config{})
	require.NoError(t, err, "failed to connect to PostgreSQL")
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// NewDB opens a connection to a migrated schema with empty tables and ids restarting at 1.
func (p *Postgres) NewDB(t testing.TB) *gorm.DB {
	t.Helper()
	db := p.Open(t)

	require.NoError(t, db.AutoMigrate(&pgimpl.UserModel{}, &pgimpl.TaskModel{}), "failed to migrate database")
	require.NoError(t, db.Exec("TRUNCATE TABLE task_models, user_models RESTART IDENTITY").Error, "failed to reset tables")
	return db
}
