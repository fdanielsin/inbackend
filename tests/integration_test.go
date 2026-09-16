package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/fdanielsin/inbackend/internal/db"
	"github.com/fdanielsin/inbackend/internal/logger"
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

	tables := database.Migrator().GetTables()
	require.Contains(t, fmt.Sprintf("%v", tables), "users", "users table not found")
	require.Contains(t, fmt.Sprintf("%v", tables), "tasks", "tasks table not found")
}
