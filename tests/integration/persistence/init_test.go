package persistence

import (
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
