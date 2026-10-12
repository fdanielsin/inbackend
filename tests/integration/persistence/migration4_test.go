package persistence

import (
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/pgsetup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// foreignKeyDeleteType returns pg_constraint.confdeltype of the task -> user foreign key:
// 'a' no action, 'c' cascade.
func foreignKeyDeleteType(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var kind string
	require.NoError(t, db.Raw("SELECT confdeltype::text FROM pg_constraint WHERE conname = 'fk_task_models_user'").Scan(&kind).Error)
	return kind
}

// fingerprint is every user and task row, as stored, in a stable order.
func fingerprint(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var rows []string
	require.NoError(t, db.Raw(`
		SELECT 'user ' || id || ' ' || name || ' ' || email FROM user_models
		UNION ALL
		SELECT 'task ' || id || ' ' || user_id || ' ' || title || ' ' || description || ' ' || completed FROM task_models
		ORDER BY 1`).Scan(&rows).Error)
	return rows
}

func seedUsersWithTasks(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec("INSERT INTO user_models (name, email) VALUES ('Ada', 'ada@x.io'), ('Bob', 'bob@x.io')").Error)
	require.NoError(t, db.Exec(`INSERT INTO task_models (title, description, user_id, completed)
		VALUES ('A1', 'd', 1, false), ('A2', '', 1, true), ('B1', 'd', 2, false)`).Error)
}

// GIVEN version 3 with users and tasks / WHEN migration 4 runs
// THEN no row changes, the constraint keeps its name, and it now cascades (USER-R13)
func TestMigration4_CascadesWithoutChangingAnyRow(t *testing.T) {
	t.Run("USER-R13: the migration changes the delete rule and no row (happy path)", func(t *testing.T) {
		dsn, db := migratedTo(t, 3)
		seedUsersWithTasks(t, db)
		before := fingerprint(t, db)
		require.Equal(t, "a", foreignKeyDeleteType(t, db), "version 3 refuses the deletion")

		require.NoError(t, pgsetup.MigrateTo(dsn, 4))

		assert.Equal(t, before, fingerprint(t, db))
		assert.True(t, hasConstraint(t, db, "fk_task_models_user"), "the name is used by the code to recognize the violation")
		assert.Equal(t, "c", foreignKeyDeleteType(t, db))
		assert.True(t, hasIndex(t, db, "idx_task_models_user_id"))
	})
	t.Run("USER-R13: after migration 4 the user and only its tasks are deleted (happy path)", func(t *testing.T) {
		dsn, db := migratedTo(t, 3)
		seedUsersWithTasks(t, db)
		require.NoError(t, pgsetup.MigrateTo(dsn, 4))

		require.NoError(t, db.Exec("DELETE FROM user_models WHERE id = 1").Error)

		assert.Equal(t, []string{"task 3 2 B1 d false", "user 2 Bob bob@x.io"}, fingerprint(t, db))
	})
}

// GIVEN version 4 with users and tasks / WHEN migration 4 is rolled back
// THEN no row is lost and the old constraint refuses the deletion of a user with tasks again (USER-R13)
func TestMigration4_DownRestoresTheRefusal(t *testing.T) {
	t.Run("USER-R13: the down migration restores the refusal and keeps every row (happy path)", func(t *testing.T) {
		dsn, db := migratedTo(t, 4)
		seedUsersWithTasks(t, db)
		before := fingerprint(t, db)

		require.NoError(t, pgsetup.MigrateTo(dsn, 3))

		assert.Equal(t, before, fingerprint(t, db))
		assert.Equal(t, "a", foreignKeyDeleteType(t, db))
		err := db.Exec("DELETE FROM user_models WHERE id = 1").Error
		require.Error(t, err)
		assert.Equal(t, "23503", sqlState(err))
		assert.Equal(t, before, fingerprint(t, db), "the refused deletion changed nothing")
	})
}
