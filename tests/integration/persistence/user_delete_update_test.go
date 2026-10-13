package persistence

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fdanielsin/inbackend/internal/domain/tasks"
	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func countWhere(t *testing.T, db *gorm.DB, table, where string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw("SELECT count(*) FROM "+table+" WHERE "+where, args...).Scan(&n).Error)
	return n
}

func addTasks(t *testing.T, db *gorm.DB, userID uint, n int) {
	t.Helper()
	repo := pgstore.NewTaskRepository(db)
	for i := 1; i <= n; i++ {
		createTestTask(t, repo, userID, fmt.Sprintf("Task %d of %d", i, userID))
	}
}

// GIVEN users with 0, 1 and 3 tasks, and a neighbour / WHEN Delete / THEN the user and exactly its tasks are gone from the table
func TestUserRepository_Delete(t *testing.T) {
	ctx := context.Background()
	for _, tasksOfDeleted := range []int{0, 1, 3} {
		t.Run(fmt.Sprintf("USER-R13: a user with %d tasks is deleted with them, the neighbour is untouched (happy path)", tasksOfDeleted), func(t *testing.T) {
			db := setupTestDB(t)
			repo := pgstore.NewUserRepository(db)
			deleted := createTestUser(t, db, "deleted@example.com")
			neighbour := createTestUser(t, db, "neighbour@example.com")
			addTasks(t, db, deleted, tasksOfDeleted)
			addTasks(t, db, neighbour, 3)
			neighbourBefore := fingerprintOf(t, db, neighbour)

			require.NoError(t, repo.Delete(ctx, deleted))

			assert.Zero(t, countWhere(t, db, "user_models", "id = ?", deleted))
			assert.Zero(t, countWhere(t, db, "task_models", "user_id = ?", deleted))
			assert.Equal(t, neighbourBefore, fingerprintOf(t, db, neighbour))
		})
	}
}

func fingerprintOf(t *testing.T, db *gorm.DB, userID uint) []string {
	t.Helper()
	var rows []string
	require.NoError(t, db.Raw(`
		SELECT 'user ' || id || ' ' || name || ' ' || email FROM user_models WHERE id = ?
		UNION ALL
		SELECT 'task ' || id || ' ' || title || ' ' || description || ' ' || completed FROM task_models WHERE user_id = ?
		ORDER BY 1`, userID, userID).Scan(&rows).Error)
	return rows
}

// GIVEN no such user, or one already deleted / WHEN Delete / THEN ErrUserNotFound and the number of users is unchanged
func TestUserRepository_Delete_AbsentUser(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)
	kept := createTestUser(t, db, "kept@example.com")
	gone := createTestUser(t, db, "gone@example.com")
	require.NoError(t, repo.Delete(ctx, gone))

	for name, id := range map[string]uint{"never existed": 999, "already deleted": gone, "zero": 0, "largest uint32": maxContractID} {
		t.Run("USER-R14: "+name+" is not found (sad path)", func(t *testing.T) {
			err := repo.Delete(ctx, id)

			require.ErrorIs(t, err, users.ErrUserNotFound)
			count, err := repo.Count(ctx)
			require.NoError(t, err)
			assert.Equal(t, int64(1), count)
			_, err = repo.GetByID(ctx, kept)
			assert.NoError(t, err)
		})
	}
}

// GIVEN a user with 3 tasks and a trigger that makes the deletion of the second task fail / WHEN Delete
// THEN it fails as a whole: the user and its 3 tasks are still there, read from another connection;
// without the trigger, the same deletion succeeds (USER-R13, criterion 2)
func TestUserRepository_Delete_IsAllOrNothing(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)
	owner := createTestUser(t, db, "owner@example.com")
	addTasks(t, db, owner, 3)
	other := pg.Open(t) // another connection to the same database
	require.NoError(t, db.Exec(`
		CREATE FUNCTION fail_on_second_task() RETURNS trigger AS $$
		BEGIN
			IF OLD.title LIKE 'Task 2 %' THEN RAISE EXCEPTION 'simulated failure'; END IF;
			RETURN OLD;
		END $$ LANGUAGE plpgsql`).Error)
	// the database is shared by the whole package: the failure must not outlive this test
	t.Cleanup(func() { _ = db.Exec("DROP FUNCTION IF EXISTS fail_on_second_task() CASCADE").Error })
	require.NoError(t, db.Exec("CREATE TRIGGER fail_second BEFORE DELETE ON task_models FOR EACH ROW EXECUTE FUNCTION fail_on_second_task()").Error)

	t.Run("USER-R13: a failure while deleting a task leaves the user and all its tasks (exception)", func(t *testing.T) {
		err := repo.Delete(ctx, owner)

		require.Error(t, err)
		require.NotErrorIs(t, err, users.ErrUserNotFound)
		assert.Equal(t, int64(1), countWhere(t, other, "user_models", "id = ?", owner))
		assert.Equal(t, int64(3), countWhere(t, other, "task_models", "user_id = ?", owner))
	})

	t.Run("USER-R13: without the trigger the same deletion succeeds, so the trigger is what made it fail (happy path)", func(t *testing.T) {
		require.NoError(t, db.Exec("DROP TRIGGER fail_second ON task_models").Error)

		require.NoError(t, repo.Delete(ctx, owner))

		assert.Zero(t, countWhere(t, other, "user_models", "id = ?", owner))
		assert.Zero(t, countWhere(t, other, "task_models", "user_id = ?", owner))
	})
}

// GIVEN a user / WHEN Update / THEN only the name changes: the email, the id and the tasks are read back unchanged from SQL
func TestUserRepository_Update(t *testing.T) {
	ctx := context.Background()
	t.Run("USER-R17: the name is replaced and nothing else changes (happy path)", func(t *testing.T) {
		db := setupTestDB(t)
		repo := pgstore.NewUserRepository(db)
		id := createTestUser(t, db, "ada@example.com")
		addTasks(t, db, id, 2)
		before := fingerprintOf(t, db, id)

		require.NoError(t, repo.Update(ctx, &users.User{ID: id, Name: "Grace", Email: "other@example.com"}))

		after := fingerprintOf(t, db, id)
		assert.Equal(t, strings.Replace(before[len(before)-1], "Owner", "Grace", 1), after[len(after)-1])
		assert.Equal(t, before[:len(before)-1], after[:len(after)-1], "tasks unchanged")
		assert.Contains(t, after[len(after)-1], "ada@example.com", "the email passed in the entity is never written")
		assert.Zero(t, countWhere(t, db, "user_models", "email = 'other@example.com'"))
	})
	t.Run("USER-R17: an absent user is not found and no row is created (sad path)", func(t *testing.T) {
		db := setupTestDB(t)
		repo := pgstore.NewUserRepository(db)
		createTestUser(t, db, "ada@example.com")

		err := repo.Update(ctx, &users.User{ID: 999, Name: "Ghost", Email: "ghost@example.com"})

		require.ErrorIs(t, err, users.ErrUserNotFound)
		assert.Equal(t, int64(1), countWhere(t, db, "user_models", "true"))
		assert.Zero(t, countWhere(t, db, "user_models", "name = 'Ghost' OR email = 'ghost@example.com'"))
	})
	t.Run("USER-R17: a deleted user is not recreated (sad path)", func(t *testing.T) {
		db := setupTestDB(t)
		repo := pgstore.NewUserRepository(db)
		id := createTestUser(t, db, "ada@example.com")
		require.NoError(t, repo.Delete(ctx, id))

		err := repo.Update(ctx, &users.User{ID: id, Name: "Back", Email: "ada@example.com"})

		require.ErrorIs(t, err, users.ErrUserNotFound)
		assert.Zero(t, countWhere(t, db, "user_models", "true"))
	})
}

// GIVEN a user with a task, and a task inserted in a transaction still open / WHEN the user is deleted from another connection
// THEN the deletion waits for the insertion; once committed, neither the user nor the task is left (USER-R13: no orphan)
func TestUserRepository_Delete_WaitsForAConcurrentTaskInsertion(t *testing.T) {
	t.Run("USER-R13: a deletion waits for a task inserted meanwhile, and takes it along (exception)", func(t *testing.T) {
		db := setupTestDB(t)
		repo := pgstore.NewUserRepository(db)
		owner := createTestUser(t, db, "owner@example.com")
		addTasks(t, db, owner, 1)
		inserter := pg.Open(t)
		tx := inserter.Begin()
		require.NoError(t, tx.Error)
		t.Cleanup(func() { tx.Rollback() }) // a failed assertion must not leave the locks held
		require.NoError(t, tx.Exec("INSERT INTO task_models (title, description, user_id, completed) VALUES ('In flight', '', ?, false)", owner).Error)

		deletion := make(chan error, 1)
		go func() { deletion <- repo.Delete(context.Background(), owner) }()
		waitForLockWait(t, db, "DELETE FROM")
		select {
		case err := <-deletion:
			t.Fatalf("the deletion must wait for the open insertion, it returned: %v", err)
		default:
		}
		require.NoError(t, tx.Commit().Error)

		require.NoError(t, <-deletion)
		assert.Zero(t, countWhere(t, db, "user_models", "true"))
		assert.Zero(t, countWhere(t, db, "task_models", "true"), "the task inserted meanwhile went with its user")
		assert.Zero(t, countWhere(t, db, "task_models t LEFT JOIN user_models u ON u.id = t.user_id", "u.id IS NULL"))
	})
}

// GIVEN a user being deleted in a transaction still open / WHEN a task is created for it from another connection
// THEN the creation waits, then is refused as ErrUserNotFound: no orphan either way (USER-R13)
func TestTaskRepository_Create_WaitsForAConcurrentUserDeletion(t *testing.T) {
	t.Run("USER-R13: a task created while its user is being deleted is refused as unknown user (exception)", func(t *testing.T) {
		db := setupTestDB(t)
		owner := createTestUser(t, db, "owner@example.com")
		deleter := pg.Open(t)
		tx := deleter.Begin()
		require.NoError(t, tx.Error)
		t.Cleanup(func() { tx.Rollback() })
		require.NoError(t, tx.Exec("DELETE FROM user_models WHERE id = ?", owner).Error)

		creation := make(chan error, 1)
		go func() {
			_, err := pgstore.NewTaskRepository(db).Create(context.Background(), &tasks.Task{Title: "Late", UserID: owner})
			creation <- err
		}()
		waitForLockWait(t, db, "INSERT INTO")
		require.NoError(t, tx.Commit().Error)

		require.ErrorIs(t, <-creation, users.ErrUserNotFound)
		assert.Zero(t, countWhere(t, db, "task_models", "true"))
	})
}

// waitForLockWait blocks until a query starting with prefix waits for a lock held by another transaction.
func waitForLockWait(t *testing.T, db *gorm.DB, prefix string) {
	t.Helper()
	require.Eventually(t, func() bool {
		return countWhere(t, db, "pg_stat_activity", "wait_event_type = 'Lock' AND query LIKE ? AND datname = current_database()", prefix+"%") > 0
	}, 10*time.Second, 20*time.Millisecond, "no query %q is waiting for a lock", prefix)
}

// GIVEN the user with the largest id / WHEN it is deleted and another one created in the same test, with no reset between
// THEN the new id is greater than every id ever given (USER-R16); after a RESTART IDENTITY reset the id is reused,
// which shows the assertion can fail
func TestUserRepository_Create_NeverReusesADeletedID(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)
	first := createTestUser(t, db, "first@example.com")
	last := createTestUser(t, db, "last@example.com")
	addTasks(t, db, last, 2)
	require.NoError(t, repo.Delete(ctx, last))

	t.Run("USER-R16: a new user gets an id greater than the deleted largest one (happy path)", func(t *testing.T) {
		created, err := repo.Create(ctx, &users.User{Name: "Newcomer", Email: "last@example.com"})

		require.NoError(t, err)
		assert.Greater(t, created, last)
		assert.Greater(t, created, first)
		assert.Zero(t, countWhere(t, db, "task_models", "user_id = ?", created), "it inherits no task")
	})

	t.Run("USER-R16: after the test tooling reset the id goes back, so the assertion above can fail (limit)", func(t *testing.T) {
		a := createTestUser(t, db, "a@example.com")
		require.NoError(t, repo.Delete(ctx, a))
		require.NoError(t, db.Exec("TRUNCATE TABLE task_models, user_models RESTART IDENTITY").Error)

		reused, err := repo.Create(ctx, &users.User{Name: "Again", Email: "b@example.com"})

		require.NoError(t, err)
		assert.LessOrEqual(t, reused, a, "RESTART IDENTITY restarts the sequence: only test tooling does this")
	})
}

// GIVEN a canary user with tasks / WHEN it is deleted / THEN no table of the public schema holds its name or email any more;
// before the deletion the same search finds them, so the search can fail (USER-R15)
func TestUserRepository_Delete_LeavesNoPersonalDataInAnyTable(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := pgstore.NewUserRepository(db)
	const name, email = "Canary Zq-7731 Name", "canary.zq7731@example.org"
	id, err := repo.Create(ctx, &users.User{Name: name, Email: email})
	require.NoError(t, err)
	addTasks(t, db, id, 2)
	require.NoError(t, repo.Update(ctx, &users.User{ID: id, Name: "Canary Zq-7731 Renamed"}))

	before := exportEveryTable(t, db)
	require.Contains(t, before, "Canary Zq-7731 Renamed", "control: the export sees the data before the deletion")
	require.Contains(t, before, email)

	require.NoError(t, repo.Delete(ctx, id))

	t.Run("USER-R15: neither the name nor the email is in any table after the deletion (happy path)", func(t *testing.T) {
		after := exportEveryTable(t, db)

		assert.NotContains(t, after, "Zq-7731")
		assert.NotContains(t, strings.ToLower(after), "zq7731")
	})
}

// exportEveryTable returns every row of every table of the public schema as JSON text.
func exportEveryTable(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var tables []string
	require.NoError(t, db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY 1").Scan(&tables).Error)
	require.NotEmpty(t, tables)
	var out strings.Builder
	for _, table := range tables {
		var rows []string
		require.NoError(t, db.Raw(fmt.Sprintf("SELECT row_to_json(t)::text FROM %q t", table)).Scan(&rows).Error)
		out.WriteString(strings.Join(rows, "\n"))
		out.WriteString("\n")
	}
	return out.String()
}
