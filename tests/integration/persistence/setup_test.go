package persistence

import (
	"testing"

	"github.com/fdanielsin/inbackend/tests/testsupport"
	"gorm.io/gorm"
)

// pg is the PostgreSQL container shared by every test of the package.
var pg *testsupport.Postgres

func TestMain(m *testing.M) {
	pg = testsupport.StartPostgres()
	m.Run()
	pg.Terminate()
}

// setupTestDB returns a connection to a migrated schema with empty tables.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return pg.NewDB(t)
}
