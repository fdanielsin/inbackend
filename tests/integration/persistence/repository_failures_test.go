package persistence

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// repositoryCalls runs every repository method once and returns the error of each call.
func repositoryCalls(ctx context.Context, db *gorm.DB) map[string]error {
	users := pgstore.NewUserRepository(db)
	tasks := pgstore.NewTaskRepository(db)
	errs := map[string]error{}

	_, errs["User.Create"] = users.Create(ctx, &user.User{Name: "Ada", Email: "ada@example.com"})
	_, errs["User.GetByID"] = users.GetByID(ctx, 1)
	_, errs["User.GetByEmail"] = users.GetByEmail(ctx, "ada@example.com")
	_, errs["User.List"] = users.List(ctx)
	_, errs["User.Count"] = users.Count(ctx)
	_, errs["Task.Create"] = tasks.Create(ctx, &task.Task{Title: "Buy milk", UserID: 1})
	_, errs["Task.GetByID"] = tasks.GetByID(ctx, 1)
	_, errs["Task.ListByUserID"] = tasks.ListByUserID(ctx, 1)
	errs["Task.Update"] = tasks.Update(ctx, &task.Task{ID: 1, Title: "Buy milk", UserID: 1})
	errs["Task.Delete"] = tasks.Delete(ctx, 1)
	_, errs["Task.Count"] = tasks.Count(ctx)
	return errs
}

// A technical failure must surface as an error, never as success nor as a
// domain error that the controllers would turn into a 404 or a 409.
func assertTechnicalFailure(t *testing.T, name string, err error) {
	t.Helper()
	require.Error(t, err, name)
	assert.NotErrorIs(t, err, user.ErrUserNotFound, name)
	assert.NotErrorIs(t, err, user.ErrUserEmailTaken, name)
	assert.NotErrorIs(t, err, task.ErrTaskNotFound, name)
}

// GIVEN the database connection is closed / WHEN any repository method runs / THEN it returns a technical error
func TestRepositories_ClosedConnection_ReturnTechnicalErrors(t *testing.T) {
	db := setupTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	for name, err := range repositoryCalls(context.Background(), db) {
		assertTechnicalFailure(t, name, err)
	}
}

// GIVEN the request context is already cancelled / WHEN any repository method runs
// THEN it returns context.Canceled and writes nothing
func TestRepositories_CancelledContext_ReturnContextCanceled(t *testing.T) {
	db := setupTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for name, err := range repositoryCalls(ctx, db) {
		assertTechnicalFailure(t, name, err)
		assert.ErrorIs(t, err, context.Canceled, name)
	}

	var users, tasks int64
	require.NoError(t, db.Model(&pgstore.UserModel{}).Count(&users).Error)
	require.NoError(t, db.Model(&pgstore.TaskModel{}).Count(&tasks).Error)
	assert.Zero(t, users, "no user may be written with a cancelled context")
	assert.Zero(t, tasks, "no task may be written with a cancelled context")
}
