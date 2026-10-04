package persistence

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/pgsetup"
	"github.com/fdanielsin/inbackend/tests/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const troubleshootingDoc = "../../../docs/troubleshooting.md"

func tableExists(t *testing.T, db *gorm.DB, table string) bool {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?", table).Scan(&n).Error)
	return n == 1
}

func count(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw("SELECT count(*) FROM "+table).Scan(&n).Error)
	return n
}

// legacyDatabase returns a database created by v1.0.0, holding a few rows with
// ids that are not contiguous (the sequences stay at 1: rows are inserted with their id).
func legacyDatabase(t *testing.T) (string, *gorm.DB) {
	t.Helper()
	pg.Require(t)
	dsn := pg.NewDatabaseFromRelease1_0_0(t)
	db := testsupport.OpenDSN(t, dsn)
	require.NoError(t, db.Exec(`
		INSERT INTO users (id, name, email) VALUES (1, 'Ada', 'Ada@X.io'), (5, 'Bob', E'  bob@y.io\t'), (9, 'Cy', NULL);
		INSERT INTO tasks (id, title, description, user_id, completed) VALUES (2, 't1', 'd', 1, true), (7, 't2', '', 9, false)`).Error)
	return dsn, db
}

// GIVEN the DDL that v1.0.0 AutoMigrate produced (read from the released tag: id bigserial, text columns,
// unique index on users.email) / WHEN the test helper rebuilds it / THEN it is exactly that DDL
func TestLegacyHelper_ReproducesTheRelease1_0_0Schema(t *testing.T) {
	pg.Require(t)
	db := testsupport.OpenDSN(t, pg.NewDatabaseFromRelease1_0_0(t))

	var columns, indexes []string
	require.NoError(t, db.Raw(`
		SELECT table_name || '.' || column_name || ' ' || data_type || ' nullable=' || is_nullable || ' default=' || coalesce(column_default, '')
		FROM information_schema.columns WHERE table_schema = 'public' ORDER BY 1`).Scan(&columns).Error)
	require.NoError(t, db.Raw("SELECT indexdef FROM pg_indexes WHERE schemaname = 'public' ORDER BY 1").Scan(&indexes).Error)

	assert.Equal(t, []string{
		"tasks.completed boolean nullable=YES default=",
		"tasks.description text nullable=YES default=",
		"tasks.id bigint nullable=NO default=nextval('tasks_id_seq'::regclass)",
		"tasks.title text nullable=YES default=",
		"tasks.user_id bigint nullable=YES default=",
		"users.email text nullable=YES default=",
		"users.id bigint nullable=NO default=nextval('users_id_seq'::regclass)",
		"users.name text nullable=YES default=",
	}, columns)
	assert.Equal(t, []string{
		"CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email)",
		"CREATE UNIQUE INDEX tasks_pkey ON public.tasks USING btree (id)",
		"CREATE UNIQUE INDEX users_pkey ON public.users USING btree (id)",
	}, indexes)
}

// GIVEN a database created by v1.0.0, never started by 1.1.0 / WHEN InitDB or Migrate / THEN both refuse with
// ErrLegacySchema before writing anything: no schema_migrations, no user_models, the 1.0.0 data untouched
func TestMigrate_Release1_0_0Database_IsRefusedWithoutWritingAnything(t *testing.T) {
	dsn, db := legacyDatabase(t)

	_, initErr := pgsetup.InitDB(dsn, slog.New(slog.DiscardHandler))
	migrateErr := pgsetup.Migrate(dsn)

	require.ErrorIs(t, initErr, pgsetup.ErrLegacySchema)
	require.ErrorIs(t, migrateErr, pgsetup.ErrLegacySchema)
	for _, table := range []string{"schema_migrations", "user_models", "task_models"} {
		assert.False(t, tableExists(t, db, table), "%s must not be created", table)
	}
	assert.Equal(t, int64(3), count(t, db, "users"))
	assert.Equal(t, int64(2), count(t, db, "tasks"))
}

// GIVEN a 1.0.0 database on which 1.1.0 already started once (schema_migrations at 3, user_models empty, users still there)
// WHEN InitDB / THEN it is refused too: the check does not depend on the migration state
func TestInitDB_Release1_0_0DatabaseAlreadyStartedOnce_IsRefused(t *testing.T) {
	dsn, db := legacyDatabase(t)
	require.NoError(t, db.Exec("ALTER TABLE users RENAME TO users_away; ALTER TABLE tasks RENAME TO tasks_away").Error)
	require.NoError(t, pgsetup.Migrate(dsn))
	require.NoError(t, db.Exec("ALTER TABLE users_away RENAME TO users; ALTER TABLE tasks_away RENAME TO tasks").Error)
	require.Equal(t, migrationState{Version: latestMigration}, readMigrationState(t, db))
	require.Zero(t, count(t, db, "user_models"))

	_, err := pgsetup.InitDB(dsn, slog.New(slog.DiscardHandler))

	require.ErrorIs(t, err, pgsetup.ErrLegacySchema)
}

// GIVEN only one of the two 1.0.0 tables / WHEN InitDB / THEN it is refused and the message names that table only
func TestInitDB_OnlyOneRelease1_0_0Table_IsRefusedAndNamed(t *testing.T) {
	for _, tc := range []struct{ kept, dropped string }{{"users", "tasks"}, {"tasks", "users"}} {
		t.Run(tc.kept, func(t *testing.T) {
			dsn, db := legacyDatabase(t)
			require.NoError(t, db.Exec("DROP TABLE "+tc.dropped).Error)

			_, err := pgsetup.InitDB(dsn, slog.New(slog.DiscardHandler))

			require.ErrorIs(t, err, pgsetup.ErrLegacySchema)
			assert.Contains(t, err.Error(), "("+tc.kept+")")
			assert.NotContains(t, err.Error(), tc.dropped)
		})
	}
}

// GIVEN a refused 1.0.0 database holding names and emails / WHEN the error is read
// THEN it points to the documentation of the repository (the image has no docs/) and carries no user data
func TestInitDB_Release1_0_0Database_ErrorHasNoUserData(t *testing.T) {
	dsn, _ := legacyDatabase(t)

	_, err := pgsetup.InitDB(dsn, slog.New(slog.DiscardHandler))

	require.ErrorIs(t, err, pgsetup.ErrLegacySchema)
	assert.Contains(t, err.Error(), "see docs/troubleshooting.md in the inbackend repository")
	for _, secret := range []string{"Ada", "bob@y.io", "X.io", "testpass"} {
		assert.NotContains(t, err.Error(), secret)
	}
}

// GIVEN a table named users that is not the 1.0.0 one / WHEN InitDB / THEN the database is not refused (limit of the check)
func TestInitDB_UnrelatedUsersTable_IsNotRefused(t *testing.T) {
	pg.Require(t)
	dsn := pg.NewEmptyDatabase(t)
	require.NoError(t, testsupport.OpenDSN(t, dsn).Exec("CREATE TABLE users (login text, password_hash text)").Error)

	db, err := pgsetup.InitDB(dsn, slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
}

// GIVEN tables named users and tasks that have only some of the 1.0.0 columns (boundary: one column short)
// WHEN InitDB / THEN the database is not refused: every column of the 1.0.0 table must be present
func TestInitDB_TablesWithOnlySomeRelease1_0_0Columns_AreNotRefused(t *testing.T) {
	pg.Require(t)
	dsn := pg.NewEmptyDatabase(t)
	require.NoError(t, testsupport.OpenDSN(t, dsn).Exec(`
		CREATE TABLE users (id bigint, name text);
		CREATE TABLE tasks (id bigint, title text, description text, user_id bigint)`).Error)

	db, err := pgsetup.InitDB(dsn, slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
}

// GIVEN a 1.1.0 database already holding data / WHEN InitDB runs again (a restart) / THEN it starts and keeps the data
func TestInitDB_Release1_1_0DatabaseWithData_StartsNormally(t *testing.T) {
	pg.Require(t)
	dsn := pg.NewEmptyDatabase(t)
	require.NoError(t, pgsetup.Migrate(dsn))
	require.NoError(t, testsupport.OpenDSN(t, dsn).Exec("INSERT INTO user_models (name, email) VALUES ('Ada', 'ada@x.io')").Error)

	db, err := pgsetup.InitDB(dsn, slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	assert.Equal(t, int64(1), count(t, db, "user_models"))
}

// GIVEN the 1.0.0 tables in public and a connection whose search_path is another schema / WHEN Migrate
// THEN it is not refused: only the current schema is inspected, and the tables are created there
func TestMigrate_Release1_0_0TablesInAnotherSchema_AreNotSeen(t *testing.T) {
	dsn, db := legacyDatabase(t)
	require.NoError(t, db.Exec("CREATE SCHEMA app").Error)

	require.NoError(t, pgsetup.Migrate(dsn+"&search_path=app"))

	var n int64
	require.NoError(t, db.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema = 'app' AND table_name = 'user_models'").Scan(&n).Error)
	assert.Equal(t, int64(1), n)
}

// docSteps returns the SQL blocks of the upgrade section of docs/troubleshooting.md, keyed by the
// "-- step <name>" label on their first line: the test runs the documentation as written.
func docSteps(t *testing.T) map[string]string {
	t.Helper()
	doc, err := os.ReadFile(troubleshootingDoc)
	require.NoError(t, err)
	steps := map[string]string{}
	for _, m := range regexp.MustCompile("(?s)```sql\n-- step ([a-z-]+)\n(.*?)```").FindAllStringSubmatch(string(doc), -1) {
		steps[m[1]] = m[2]
	}
	for _, name := range []string{"rename", "check", "copy", "drop"} {
		require.Contains(t, steps, name, "docs/troubleshooting.md must document the SQL step %q", name)
	}
	return steps
}

type copyChecks struct{ TargetRows, DuplicateEmailGroups, OrphanTasks int64 }

func runCheck(t *testing.T, db *gorm.DB, steps map[string]string) copyChecks {
	t.Helper()
	var c copyChecks
	require.NoError(t, db.Raw(steps["check"]).Row().Scan(&c.TargetRows, &c.DuplicateEmailGroups, &c.OrphanTasks))
	return c
}

// GIVEN a 1.0.0 database with data / WHEN the documented SQL is run as written (rename, create the tables with
// the migrations, check, copy) / THEN InitDB starts, ids and rows are kept, emails normalized, and the next ids follow the copied ones
func TestUpgradeFromRelease1_0_0_DocumentedSQL_CopiesTheData(t *testing.T) {
	steps := docSteps(t)
	dsn, db := legacyDatabase(t)

	require.NoError(t, db.Exec(steps["rename"]).Error)
	require.NoError(t, pgsetup.Migrate(dsn))
	assert.Equal(t, copyChecks{}, runCheck(t, db, steps))
	require.NoError(t, db.Exec(steps["copy"]).Error)

	started, err := pgsetup.InitDB(dsn, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	sqlDB, err := started.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	var users []string
	require.NoError(t, db.Raw("SELECT id || ':' || name || ':' || coalesce(email, '<null>') FROM user_models ORDER BY id").Scan(&users).Error)
	assert.Equal(t, []string{"1:Ada:ada@x.io", "5:Bob:bob@y.io", "9:Cy:<null>"}, users)
	var tasks []string
	require.NoError(t, db.Raw("SELECT id || ':' || title || ':' || user_id || ':' || completed FROM task_models ORDER BY id").Scan(&tasks).Error)
	assert.Equal(t, []string{"2:t1:1:true", "7:t2:9:false"}, tasks)

	var nextUser, nextTask int64
	require.NoError(t, db.Raw("INSERT INTO user_models (name, email) VALUES ('New', 'new@x.io') RETURNING id").Scan(&nextUser).Error)
	require.NoError(t, db.Raw("INSERT INTO task_models (title, description, user_id, completed) VALUES ('n', '', 1, false) RETURNING id").Scan(&nextTask).Error)
	assert.Equal(t, int64(10), nextUser, "setval must continue after the highest copied id")
	assert.Equal(t, int64(8), nextTask)

	require.NoError(t, db.Exec(steps["drop"]).Error)
	assert.False(t, tableExists(t, db, "users_v1"))
	assert.False(t, tableExists(t, db, "tasks_v1"))
}

// GIVEN a 1.0.0 database holding no rows / WHEN the documented copy runs / THEN it succeeds and the sequences are usable (boundary: empty tables)
func TestUpgradeFromRelease1_0_0_DocumentedSQL_EmptyTables(t *testing.T) {
	steps := docSteps(t)
	pg.Require(t)
	dsn := pg.NewDatabaseFromRelease1_0_0(t)
	db := testsupport.OpenDSN(t, dsn)

	require.NoError(t, db.Exec(steps["rename"]).Error)
	require.NoError(t, pgsetup.Migrate(dsn))
	require.NoError(t, db.Exec(steps["copy"]).Error)

	var id int64
	require.NoError(t, db.Raw("INSERT INTO user_models (name, email) VALUES ('New', 'new@x.io') RETURNING id").Scan(&id).Error)
	assert.Equal(t, int64(1), id)
}

// GIVEN the documented check / WHEN data would make the copy fail or collide / THEN it reports it
// (target tables not empty, emails equal once trimmed and lower-cased, tasks without user), and nothing otherwise
func TestUpgradeFromRelease1_0_0_DocumentedCheck_DetectsWhatBlocksTheCopy(t *testing.T) {
	steps := docSteps(t)
	tests := map[string]struct {
		setup string
		want  copyChecks
	}{
		"target not empty":  {"INSERT INTO user_models (name, email) VALUES ('Zed', 'zed@x.io')", copyChecks{TargetRows: 1}},
		"duplicate emails":  {"INSERT INTO users (id, name, email) VALUES (20, 'Ada2', 'ada@x.io ')", copyChecks{DuplicateEmailGroups: 1}},
		"two null emails":   {"INSERT INTO users (id, name, email) VALUES (21, 'Dee', NULL)", copyChecks{}},
		"task without user": {"INSERT INTO tasks (id, title, description, user_id, completed) VALUES (30, 'x', '', 99, false)", copyChecks{OrphanTasks: 1}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dsn, db := legacyDatabase(t)
			require.NoError(t, db.Exec(steps["rename"]).Error)
			require.NoError(t, pgsetup.Migrate(dsn))
			target := tc.setup
			if strings.Contains(target, "users (") || strings.Contains(target, "tasks (") {
				target = strings.NewReplacer("INTO users ", "INTO users_v1 ", "INTO tasks ", "INTO tasks_v1 ").Replace(target)
			}
			require.NoError(t, db.Exec(target).Error, fmt.Sprint(name))

			assert.Equal(t, tc.want, runCheck(t, db, steps))
		})
	}
}
