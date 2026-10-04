package testsupport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// NewDatabaseFromRelease1_0_0 creates an empty database holding the schema of the
// published v1.0.0, which created it with GORM AutoMigrate (internal/db/db.go) on
// the models below, copied from internal/models/models.go of that tag. It returns
// the DSN; the database is dropped at the end of the test.
func (p *Postgres) NewDatabaseFromRelease1_0_0(t testing.TB) string {
	t.Helper()
	type User struct {
		ID    uint `gorm:"primaryKey"`
		Name  string
		Email string `gorm:"uniqueIndex"`
	}
	type Task struct {
		ID          uint `gorm:"primaryKey"`
		Title       string
		Description string
		UserID      uint
		Completed   bool
	}

	dsn := p.NewEmptyDatabase(t)
	require.NoError(t, OpenDSN(t, dsn).AutoMigrate(&User{}, &Task{}))
	return dsn
}
