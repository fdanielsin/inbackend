package persistence

import (
	"errors"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/database"
	"github.com/fdanielsin/inbackend/tests/testsupport"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// GIVEN an empty database / WHEN InitDB (application startup) / THEN every migration is applied before the connection is returned
func TestInitDB_MigratesAnEmptyDatabase(t *testing.T) {
	dsn := pg.NewEmptyDatabase(t)

	db, err := database.InitDB(dsn, zap.NewNop())
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	for _, table := range []string{"user_models", "task_models"} {
		assert.True(t, db.Migrator().HasTable(table), "table %s must exist after startup", table)
	}
	assert.Equal(t, migrationState{Version: latestMigration}, readMigrationState(t, db))
	assert.True(t, hasIndex(t, db, "idx_user_models_email_lower"), "unique lower(email) index must exist")
	assert.True(t, hasConstraint(t, db, "fk_task_models_user"), "task owner foreign key must exist")
	assert.Equal(t, 25, sqlDB.Stats().MaxOpenConnections, "connection pool size")
}

// GIVEN an unreachable database / WHEN InitDB / THEN an error is returned instead of a usable connection
func TestInitDB_UnreachableDatabase(t *testing.T) {
	db, err := database.InitDB("postgres://testuser:testpass@127.0.0.1:1/testdb?sslmode=disable&connect_timeout=2", zap.NewNop())

	assert.Error(t, err)
	assert.Nil(t, db)
}

// initDBObserved runs InitDB with a logger that records every entry.
func initDBObserved(t *testing.T, dsn string) error {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	db, err := database.InitDB(dsn, zap.New(core))
	assert.Nil(t, db)
	assert.Equal(t, 1, logs.FilterMessage("Initializing PostgreSQL database").Len())
	assert.Zero(t, logs.FilterLevelExact(zapcore.ErrorLevel).Len(), "InitDB returns its failure, the caller logs it")
	return err
}

// GIVEN an unreachable database whose DSN carries a password / WHEN InitDB
// THEN a wrapped "migrate database" error, nothing logged as an error, and no password in the error
func TestInitDB_UnreachableDatabase_ReturnsAWrappedError(t *testing.T) {
	err := initDBObserved(t, "postgres://qauser:qa-s3cr3t-pw@127.0.0.1:1/testdb?sslmode=disable&connect_timeout=2")

	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "migrate database: "), err.Error())
	assert.Error(t, errors.Unwrap(err), "the cause is wrapped, not flattened")
	assert.NotContains(t, err.Error(), "qa-s3cr3t-pw")
}

// GIVEN a database whose data blocks a migration / WHEN InitDB
// THEN the PostgreSQL error is still reachable through the wrapping (errors.As), nothing logged as an error
func TestInitDB_RefusedMigration_KeepsThePostgresCause(t *testing.T) {
	dsn := pg.NewEmptyDatabase(t)
	require.NoError(t, database.MigrateTo(dsn, 1))
	require.NoError(t, testsupport.OpenDSN(t, dsn).Exec(
		"INSERT INTO user_models (name, email) VALUES ('Ada', 'dup.person@x.io'), ('Ada2', 'DUP.PERSON@X.IO')").Error)

	err := initDBObserved(t, dsn)

	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "migrate database: "), err.Error())
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Contains(t, pgErr.Message, "cannot normalize emails")
}
