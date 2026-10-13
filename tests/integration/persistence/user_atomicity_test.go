package persistence

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/fdanielsin/inbackend/internal/infrastructure/pgsetup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN a trigger that refuses the deletion of the user row itself / WHEN Delete
// THEN the tasks are still there too: the cascade and the user row are one statement, not tasks first then the user (USER-R13)
func TestUserRepository_Delete_FailureOnTheUserRowKeepsTheTasks(t *testing.T) {
	t.Run("USER-R13: a refusal on the user row leaves the user and all its tasks (sad path)", func(t *testing.T) {
		db := setupTestDB(t)
		repo := pgstore.NewUserRepository(db)
		owner := createTestUser(t, db, "owner@example.com")
		addTasks(t, db, owner, 3)
		require.NoError(t, db.Exec(`CREATE FUNCTION refuse_user() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'refused'; END $$ LANGUAGE plpgsql`).Error)
		// the database is shared by the whole package: the refusal must not outlive this test
		t.Cleanup(func() { _ = db.Exec("DROP FUNCTION IF EXISTS refuse_user() CASCADE").Error })
		require.NoError(t, db.Exec("CREATE TRIGGER refuse_user BEFORE DELETE ON user_models FOR EACH ROW EXECUTE FUNCTION refuse_user()").Error)

		err := repo.Delete(context.Background(), owner)

		require.Error(t, err)
		assert.Equal(t, int64(1), countWhere(t, db, "user_models", "id = ?", owner))
		assert.Equal(t, int64(3), countWhere(t, db, "task_models", "user_id = ?", owner))
	})
}

// GIVEN version 3 with data / WHEN the migration goes up, down, up / THEN no row changes at any step, the rule follows each step, and the final rule cascades
func TestMigration4_UpDownUpWithData(t *testing.T) {
	t.Run("USER-R13: up, down and up again lose no row and end with the cascade (happy path)", func(t *testing.T) {
		dsn, db := migratedTo(t, 3)
		seedUsersWithTasks(t, db)
		before := fingerprint(t, db)
		for i, step := range []struct {
			to   uint
			kind string
		}{{4, "c"}, {3, "a"}, {4, "c"}} {
			require.NoError(t, pgsetup.MigrateTo(dsn, step.to), "step %d", i)
			assert.Equal(t, before, fingerprint(t, db), "step %d", i)
			assert.Equal(t, step.kind, foreignKeyDeleteType(t, db), "step %d", i)
		}

		require.NoError(t, db.Exec("DELETE FROM user_models WHERE id = 1").Error)

		assert.Equal(t, []string{"task 3 2 B1 d false", "user 2 Bob bob@x.io"}, fingerprint(t, db))
	})
}
