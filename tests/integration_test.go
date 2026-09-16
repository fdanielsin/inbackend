package tests

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/db"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var testLogger *zap.Logger

func init() {
	testLogger = zap.NewNop()
}

func setupTestDB(t *testing.T, ctx context.Context) (*gorm.DB, *db.TestDB) {
	testDB, err := db.SetupTestDB(ctx)
	require.NoError(t, err, "failed to setup test database")

	database, err := db.InitDB(testDB.DSN, testLogger)
	require.NoError(t, err, "failed to initialize database")

	return database, testDB
}

func TestDatabaseConnection(t *testing.T) {
	ctx := context.Background()
	database, testDB := setupTestDB(t, ctx)
	defer testDB.Stop(ctx)

	sqlDB, _ := database.DB()
	err := sqlDB.Ping()
	require.NoError(t, err, "failed to ping database")
}

func TestMigrations(t *testing.T) {
	ctx := context.Background()
	database, testDB := setupTestDB(t, ctx)
	defer testDB.Stop(ctx)

	tables, _ := database.Migrator().GetTables()
	require.Greater(t, len(tables), 0, "no tables found after migration")
}
