// Package testsupport starts the PostgreSQL container shared by the integration
// and E2E test packages.
package testsupport

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fdanielsin/inbackend/internal/infrastructure/database"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver used by the container readiness check
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
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
	return OpenDSN(t, p.DSN)
}

// OpenDSN opens a connection to the given database, closed at the end of the test.
func OpenDSN(t testing.TB, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err, "failed to connect to PostgreSQL")
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// NewDB opens a connection to the schema of the production migrations, with
// empty tables and ids restarting at 1.
func (p *Postgres) NewDB(t testing.TB) *gorm.DB {
	t.Helper()
	p.Require(t)
	require.NoError(t, database.Migrate(p.DSN), "failed to migrate database")
	db := OpenDSN(t, p.DSN)
	require.NoError(t, db.Exec("TRUNCATE TABLE task_models, user_models RESTART IDENTITY").Error, "failed to reset tables")
	return db
}

var emptyDatabases atomic.Int64

// NewEmptyDatabase creates a database without any table in the container and
// returns its DSN; it is dropped at the end of the test. Each migration test
// gets its own, since a refused migration leaves its database dirty.
func (p *Postgres) NewEmptyDatabase(t testing.TB) string {
	t.Helper()
	admin := p.Open(t)
	name := fmt.Sprintf("qa_empty_%d", emptyDatabases.Add(1))
	require.NoError(t, admin.Exec("DROP DATABASE IF EXISTS "+name+" WITH (FORCE)").Error)
	require.NoError(t, admin.Exec("CREATE DATABASE "+name).Error)
	t.Cleanup(func() { _ = admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)").Error })

	dsn := strings.Replace(p.DSN, "/testdb?", "/"+name+"?", 1)
	require.NotEqual(t, p.DSN, dsn, "unexpected DSN format")
	return dsn
}
