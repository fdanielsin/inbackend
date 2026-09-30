package persistence

import (
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// GIVEN an empty database / WHEN InitDB (application startup) / THEN both tables are created
func TestInitDB_CreatesSchema(t *testing.T) {
	admin := pg.Open(t)
	require.NoError(t, admin.Exec("DROP TABLE IF EXISTS task_models, user_models").Error)

	db, err := database.InitDB(pg.DSN, zap.NewNop())
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	for _, table := range []string{"user_models", "task_models"} {
		assert.True(t, db.Migrator().HasTable(table), "table %s must exist after startup", table)
	}
	assert.True(t, db.Migrator().HasIndex("user_models", "idx_user_models_email"), "unique email index must exist")
}

// GIVEN an unreachable database / WHEN InitDB / THEN an error is returned instead of a usable connection
func TestInitDB_UnreachableDatabase(t *testing.T) {
	db, err := database.InitDB("postgres://testuser:testpass@127.0.0.1:1/testdb?sslmode=disable&connect_timeout=2", zap.NewNop())

	assert.Error(t, err)
	assert.Nil(t, db)
}
