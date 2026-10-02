package persistence

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/tasks"
	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// repositoryCalls runs every repository method once and returns the error of each call.
func repositoryCalls(ctx context.Context, db *gorm.DB) map[string]error {
	userRepo := pgstore.NewUserRepository(db)
	taskRepo := pgstore.NewTaskRepository(db)
	errs := map[string]error{}

	_, errs["User.Create"] = userRepo.Create(ctx, &users.User{Name: "Ada", Email: "ada@example.com"})
	_, errs["User.GetByID"] = userRepo.GetByID(ctx, 1)
	_, errs["User.GetByEmail"] = userRepo.GetByEmail(ctx, "ada@example.com")
	_, errs["User.List"] = userRepo.List(ctx)
	_, errs["User.Count"] = userRepo.Count(ctx)
	_, errs["Task.Create"] = taskRepo.Create(ctx, &tasks.Task{Title: "Buy milk", UserID: 1})
	_, errs["Task.GetByID"] = taskRepo.GetByID(ctx, 1)
	_, errs["Task.ListByUserID"] = taskRepo.ListByUserID(ctx, 1)
	errs["Task.Update"] = taskRepo.Update(ctx, &tasks.Task{ID: 1, Title: "Buy milk", UserID: 1})
	errs["Task.Delete"] = taskRepo.Delete(ctx, 1)
	_, errs["Task.Count"] = taskRepo.Count(ctx)
	return errs
}

// A technical failure must surface as an error, never as success nor as a
// domain error that the controllers would turn into a 404 or a 409.
func assertTechnicalFailure(t *testing.T, name string, err error) {
	t.Helper()
	require.Error(t, err, name)
	require.NotErrorIs(t, err, users.ErrUserNotFound, name)
	require.NotErrorIs(t, err, users.ErrUserEmailTaken, name)
	assert.NotErrorIs(t, err, tasks.ErrTaskNotFound, name)
}

// GIVEN the database connection is closed / WHEN any repository method runs / THEN it returns a technical error
func TestRepositories_ClosedConnection_ReturnTechnicalErrors(t *testing.T) {
	db := setupTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	for name, err := range repositoryCalls(context.Background(), db) {
		t.Run(name, func(t *testing.T) {
			assertTechnicalFailure(t, name, err)
		})
	}
}

// GIVEN the request context is already cancelled / WHEN any repository method runs
// THEN it returns context.Canceled and writes nothing
func TestRepositories_CancelledContext_ReturnContextCanceled(t *testing.T) {
	db := setupTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for name, err := range repositoryCalls(ctx, db) {
		t.Run(name, func(t *testing.T) {
			assertTechnicalFailure(t, name, err)
			assert.ErrorIs(t, err, context.Canceled, name)
		})
	}

	var userCount, taskCount int64
	require.NoError(t, db.Model(&pgstore.UserModel{}).Count(&userCount).Error)
	require.NoError(t, db.Model(&pgstore.TaskModel{}).Count(&taskCount).Error)
	assert.Zero(t, userCount, "no user may be written with a cancelled context")
	assert.Zero(t, taskCount, "no task may be written with a cancelled context")
}
