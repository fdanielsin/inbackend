package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/tasks"
	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/fdanielsin/inbackend/internal/infrastructure/pgsetup"
	"github.com/fdanielsin/inbackend/tests/testsupport"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// latestMigration is the number of migrations shipped in internal/infrastructure/pgsetup/migrations.
const latestMigration = 4

type migrationState struct {
	Version int64
	Dirty   bool
}

func readMigrationState(t *testing.T, db *gorm.DB) migrationState {
	t.Helper()
	var state migrationState
	require.NoError(t, db.Raw("SELECT version, dirty FROM schema_migrations").Row().Scan(&state.Version, &state.Dirty))
	return state
}

// schemaSnapshot describes the application tables: columns, indexes and constraints.
type schemaSnapshot struct {
	Columns     []string
	Indexes     []string
	Constraints []string
}

func snapshotSchema(t *testing.T, db *gorm.DB) schemaSnapshot {
	t.Helper()
	var s schemaSnapshot
	require.NoError(t, db.Raw(`
		SELECT table_name || '.' || column_name || ' ' || data_type || ' nullable=' || is_nullable || ' default=' || coalesce(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name IN ('user_models', 'task_models')
		ORDER BY 1`).Scan(&s.Columns).Error)
	require.NoError(t, db.Raw(`
		SELECT indexdef FROM pg_indexes
		WHERE schemaname = 'public' AND tablename IN ('user_models', 'task_models')
		ORDER BY 1`).Scan(&s.Indexes).Error)
	require.NoError(t, db.Raw(`
		SELECT c.conrelid::regclass::text || ' ' || c.conname || ' ' || pg_get_constraintdef(c.oid)
		FROM pg_constraint c
		WHERE c.conrelid IN ('user_models'::regclass, 'task_models'::regclass)
		ORDER BY 1`).Scan(&s.Constraints).Error)
	return s
}

func hasIndex(t *testing.T, db *gorm.DB, name string) bool {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw("SELECT count(*) FROM pg_indexes WHERE schemaname = 'public' AND indexname = ?", name).Scan(&n).Error)
	return n == 1
}

func hasConstraint(t *testing.T, db *gorm.DB, name string) bool {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw("SELECT count(*) FROM pg_constraint WHERE conname = ?", name).Scan(&n).Error)
	return n == 1
}

func emails(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var list []string
	require.NoError(t, db.Raw("SELECT email FROM user_models ORDER BY id").Scan(&list).Error)
	return list
}

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// migratedTo returns an empty database brought to the given version, and a connection to it.
func migratedTo(t *testing.T, version uint) (string, *gorm.DB) {
	t.Helper()
	dsn := pg.NewEmptyDatabase(t)
	require.NoError(t, pgsetup.MigrateTo(dsn, version))
	return dsn, testsupport.OpenDSN(t, dsn)
}

// GIVEN an empty database / WHEN Migrate / THEN every version is applied, clean, with the final constraints
func TestMigrate_EmptyDatabase_AppliesEveryVersion(t *testing.T) {
	dsn := pg.NewEmptyDatabase(t)

	require.NoError(t, pgsetup.Migrate(dsn))

	db := testsupport.OpenDSN(t, dsn)
	assert.Equal(t, migrationState{Version: latestMigration}, readMigrationState(t, db))
	assert.True(t, hasIndex(t, db, "idx_user_models_email_lower"))
	assert.False(t, hasIndex(t, db, "idx_user_models_email"), "the case-sensitive index is replaced")
	assert.True(t, hasIndex(t, db, "idx_task_models_user_id"))
	assert.True(t, hasConstraint(t, db, "fk_task_models_user"))
}

// GIVEN an up-to-date database / WHEN Migrate runs again / THEN nothing changes and no error is returned
func TestMigrate_UpToDate_IsANoOp(t *testing.T) {
	dsn, db := migratedTo(t, latestMigration)
	before := snapshotSchema(t, db)

	require.NoError(t, pgsetup.Migrate(dsn))

	assert.Equal(t, before, snapshotSchema(t, db))
	assert.Equal(t, migrationState{Version: latestMigration}, readMigrationState(t, db))
}

// GIVEN each version / WHEN migrating down one step / THEN the schema is exactly the one of the previous version
func TestMigrateTo_EachDownRestoresThePreviousSchema(t *testing.T) {
	dsn := pg.NewEmptyDatabase(t)
	db := testsupport.OpenDSN(t, dsn)
	snapshots := map[uint]schemaSnapshot{}
	for version := uint(1); version <= latestMigration; version++ {
		require.NoError(t, pgsetup.MigrateTo(dsn, version))
		snapshots[version] = snapshotSchema(t, db)
	}

	for version := uint(latestMigration - 1); version >= 1; version-- {
		require.NoError(t, pgsetup.MigrateTo(dsn, version))
		assert.Equal(t, snapshots[version], snapshotSchema(t, db), "down to version %d", version)
	}
	require.NoError(t, pgsetup.MigrateTo(dsn, 0))
	var tables int64
	require.NoError(t, db.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('user_models', 'task_models')").Scan(&tables).Error)
	assert.Zero(t, tables, "version 0 has no application table")

	require.NoError(t, pgsetup.Migrate(dsn))
	assert.Equal(t, snapshots[latestMigration], snapshotSchema(t, db), "up again after a full down")
}

// GIVEN two empty databases / WHEN one is created by GORM AutoMigrate on the pgstore models (user_models, task_models)
// and the other by migration 1 / THEN both schemas are identical, and migrating the AutoMigrate one keeps its data
// (the 1.0.0 tables users and tasks are a different case: see legacy_schema_test.go)
func TestMigration1_MatchesTheGORMSchemaOfTheStoreModels(t *testing.T) {
	legacyDSN := pg.NewEmptyDatabase(t)
	legacy := testsupport.OpenDSN(t, legacyDSN)
	require.NoError(t, legacy.AutoMigrate(&pgstore.UserModel{}, &pgstore.TaskModel{}))
	_, migrated := migratedTo(t, 1)

	assert.Equal(t, snapshotSchema(t, legacy), snapshotSchema(t, migrated))

	require.NoError(t, legacy.Exec("INSERT INTO user_models (name, email) VALUES ('Ada', 'ada@x.io')").Error)
	require.NoError(t, legacy.Exec("INSERT INTO task_models (title, description, user_id, completed) VALUES ('Keep me', '', 1, true)").Error)
	require.NoError(t, pgsetup.Migrate(legacyDSN))
	assert.Equal(t, []string{"ada@x.io"}, emails(t, legacy))
	var taskCount int64
	require.NoError(t, legacy.Raw("SELECT count(*) FROM task_models WHERE title = 'Keep me' AND completed").Scan(&taskCount).Error)
	assert.Equal(t, int64(1), taskCount)
}

// GIVEN version 1 with emails in mixed case or padded / WHEN migration 2 runs / THEN they are stored trimmed and lower-cased
func TestMigration2_NormalizesExistingEmails(t *testing.T) {
	dsn, db := migratedTo(t, 1)
	require.NoError(t, db.Exec("INSERT INTO user_models (name, email) VALUES ('Ada', 'Ada@X.io'), ('Bob', E'  bob@y.io\\t'), ('Carl', 'carl@z.io')").Error)

	require.NoError(t, pgsetup.MigrateTo(dsn, 2))

	assert.Equal(t, []string{"ada@x.io", "bob@y.io", "carl@z.io"}, emails(t, db))
}

// GIVEN version 1 with emails that differ only by one letter / WHEN migration 2 runs / THEN it passes (boundary of the duplicate check)
func TestMigration2_DistinctEmailsPass(t *testing.T) {
	dsn, db := migratedTo(t, 1)
	require.NoError(t, db.Exec("INSERT INTO user_models (name, email) VALUES ('Ada', 'ada@x.io'), ('Adb', 'adb@x.io')").Error)

	require.NoError(t, pgsetup.MigrateTo(dsn, 2))

	assert.Equal(t, migrationState{Version: 2}, readMigrationState(t, db))
}

// GIVEN version 1 with two emails equal once trimmed and lower-cased / WHEN migration 2 runs
// THEN it is refused without any email in the error, nothing is changed, and the fix is explained
func TestMigration2_RefusesCaseDuplicates(t *testing.T) {
	dsn, db := migratedTo(t, 1)
	// One group differs by case only, the other by surrounding spaces only.
	require.NoError(t, db.Exec("INSERT INTO user_models (name, email) VALUES ('Ada', 'ada@x.io'), ('Ada2', 'ADA@X.IO'), ('Bob', 'bob@y.io'), ('Bob2', ' bob@y.io'), ('Carl', 'Carl@Z.io')").Error)

	err := pgsetup.MigrateTo(dsn, 2)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot normalize emails: 2 group(s)")
	assert.Contains(t, err.Error(), "make migrate-force version=1")
	for _, email := range []string{"ada@x.io", "bob@y.io", "carl@z.io"} {
		assert.NotContains(t, strings.ToLower(err.Error()), email)
	}
	assert.Equal(t, []string{"ada@x.io", "ADA@X.IO", "bob@y.io", " bob@y.io", "Carl@Z.io"}, emails(t, db), "nothing may be changed")
	assert.Equal(t, migrationState{Version: 2, Dirty: true}, readMigrationState(t, db))
}

// GIVEN a database left dirty by a refused migration / WHEN Migrate runs again / THEN it refuses and tells how to recover
func TestMigrate_DirtyDatabase_Refuses(t *testing.T) {
	dsn, db := migratedTo(t, 1)
	require.NoError(t, db.Exec("INSERT INTO user_models (name, email) VALUES ('Ada', 'ada@x.io'), ('Ada2', 'ADA@X.IO')").Error)
	require.Error(t, pgsetup.Migrate(dsn))

	err := pgsetup.Migrate(dsn)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "migrate-force")
}

// GIVEN version 2 with a task whose user does not exist / WHEN migration 3 runs
// THEN it is refused, no task is deleted, and the fix is explained (B-02: no silent deletion)
func TestMigration3_RefusesOrphanTasks(t *testing.T) {
	dsn, db := migratedTo(t, 2)
	require.NoError(t, db.Exec("INSERT INTO user_models (name, email) VALUES ('Ada', 'ada@x.io')").Error)
	require.NoError(t, db.Exec("INSERT INTO task_models (title, description, user_id, completed) VALUES ('Owned', '', 1, false), ('Orphan', '', 999, false)").Error)

	err := pgsetup.MigrateTo(dsn, 3)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot add foreign key: 1 task(s)")
	assert.Contains(t, err.Error(), "make migrate-force version=2")
	var taskCount int64
	require.NoError(t, db.Raw("SELECT count(*) FROM task_models").Scan(&taskCount).Error)
	assert.Equal(t, int64(2), taskCount, "no task may be deleted")
	assert.False(t, hasConstraint(t, db, "fk_task_models_user"))
}

// GIVEN version 2 where every task has an existing owner / WHEN migration 3 runs / THEN the foreign key is added
func TestMigration3_NoOrphanPasses(t *testing.T) {
	dsn, db := migratedTo(t, 2)
	require.NoError(t, db.Exec("INSERT INTO user_models (name, email) VALUES ('Ada', 'ada@x.io')").Error)
	require.NoError(t, db.Exec("INSERT INTO task_models (title, description, user_id, completed) VALUES ('Owned', '', 1, false)").Error)

	require.NoError(t, pgsetup.MigrateTo(dsn, 3))

	assert.True(t, hasConstraint(t, db, "fk_task_models_user"))
}

// GIVEN an unreachable database / WHEN Migrate / THEN an error, never a success
func TestMigrate_UnreachableDatabase(t *testing.T) {
	err := pgsetup.Migrate("postgres://testuser:testpass@127.0.0.1:1/testdb?sslmode=disable&connect_timeout=2")

	assert.Error(t, err)
}

// GIVEN a DATABASE_URL that is not a postgres:// URL / WHEN Migrate / THEN ErrInvalidDatabaseURL, without the value
func TestMigrate_RejectsNonURLDatabaseSettings(t *testing.T) {
	for _, tc := range []struct{ name, dsn string }{
		{"key-value form", "host=localhost password=secretpw dbname=x"},
		{"scheme without authority", "postgres:/x"},
		{"scheme in capitals", "POSTGRES://u:secretpw@h/x"},
		{"driver scheme", "pgx5://u:secretpw@h/x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := pgsetup.Migrate(tc.dsn)

			require.ErrorIs(t, err, pgsetup.ErrInvalidDatabaseURL, tc.dsn)
			assert.Equal(t, "DATABASE_URL must be a postgres:// URL", err.Error(), tc.dsn)
			assert.NotContains(t, err.Error(), "secretpw", tc.dsn)
		})
	}
}

// GIVEN the postgresql:// spelling / WHEN Migrate / THEN it is accepted like postgres://
func TestMigrate_AcceptsPostgresqlScheme(t *testing.T) {
	dsn := pg.NewEmptyDatabase(t)

	require.NoError(t, pgsetup.Migrate("postgresql://"+dsn[len("postgres://"):]))
}

// GIVEN a user with a task / WHEN the user is deleted directly in the database
// THEN the foreign key cascades: the task goes with the user, in the same statement (USER-R13; was ON DELETE RESTRICT before migration 4)
func TestForeignKey_CascadesDeletingAUserToItsTasks(t *testing.T) {
	t.Run("USER-R13: deleting a user in SQL takes its tasks with it (happy path)", func(t *testing.T) {
		db := setupTestDB(t)
		ownerID := createTestUser(t, db, "owner@example.com")
		createTestTask(t, pgstore.NewTaskRepository(db), ownerID, "Goes with its owner")

		require.NoError(t, db.Exec("DELETE FROM user_models WHERE id = ?", ownerID).Error)

		var remaining int64
		require.NoError(t, db.Raw("SELECT count(*) FROM task_models").Scan(&remaining).Error)
		assert.Zero(t, remaining)
	})
}

// GIVEN a stored email / WHEN the same email in another case is inserted directly, bypassing the code
// THEN the database itself refuses it (unique index on lower(email))
func TestEmailIndex_IsCaseInsensitiveInTheDatabase(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO user_models (name, email) VALUES ('Ada', 'ada@x.io')").Error)

	err := db.Exec("INSERT INTO user_models (name, email) VALUES ('Other', 'ADA@X.IO')").Error

	require.Error(t, err)
	assert.Equal(t, "23505", sqlState(err))
}

// GIVEN no user with this id / WHEN a task is created for it / THEN ErrUserNotFound and nothing is stored (B-02)
func TestTaskRepository_Create_UnknownUser_ReturnsUserNotFound(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := pgstore.NewTaskRepository(db)
	createTestUser(t, db, "owner@example.com")

	for _, userID := range []uint{999, maxContractID} {
		t.Run(fmt.Sprintf("user %d", userID), func(t *testing.T) {
			_, err := repo.Create(ctx, &tasks.Task{Title: "Orphan", UserID: userID})

			assert.ErrorIs(t, err, users.ErrUserNotFound, "user %d", userID)
		})
	}
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count)
}

// GIVEN another unique index on the users table / WHEN it is violated / THEN the error is technical, not EMAIL_TAKEN
func TestUserRepository_Create_OtherUniqueViolationIsNotEmailTaken(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	require.NoError(t, db.Exec("CREATE UNIQUE INDEX qa_unique_name ON user_models (name)").Error)
	t.Cleanup(func() { _ = db.Exec("DROP INDEX IF EXISTS qa_unique_name").Error })
	repo := pgstore.NewUserRepository(db)
	_, err := repo.Create(ctx, &users.User{Name: "Ada", Email: "ada@x.io"})
	require.NoError(t, err)

	_, err = repo.Create(ctx, &users.User{Name: "Ada", Email: "other@x.io"})

	require.Error(t, err)
	require.NotErrorIs(t, err, users.ErrUserEmailTaken)
	assert.Equal(t, "23505", sqlState(err))
}
