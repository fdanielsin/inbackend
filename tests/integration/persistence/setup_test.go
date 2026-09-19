package persistence

import (
	"testing"

	"github.com/fdanielsin/inbackend/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for integration tests
// GIVEN: Testing database needed / WHEN: setupTestDB() / THEN: SQLite in-memory DB with migrations
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to setup test database")

	err = db.AutoMigrate(&models.User{}, &models.Task{})
	require.NoError(t, err, "failed to migrate test database")

	return db
}
