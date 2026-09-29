package persistence

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// R1
func TestTaskRepository_Update_PersistsCompletedFalse(t *testing.T) {
	// GIVEN: A completed task stored in PostgreSQL.
	ctx := context.Background()
	repo := postgres.NewTaskRepository(setupTestDB(t))
	id, err := repo.Create(ctx, &task.Task{Title: "Buy milk", Description: "Semi-skimmed", UserID: 1, Completed: true})
	require.NoError(t, err)

	// WHEN: The task is updated with completed=false.
	err = repo.Update(ctx, &task.Task{ID: id, Title: "Buy milk", Description: "Semi-skimmed", UserID: 1, Completed: false})

	// THEN: completed=false is persisted.
	require.NoError(t, err)
	stored, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.False(t, stored.Completed, "completed=false must be persisted")
	assert.Equal(t, "Buy milk", stored.Title)
	assert.Equal(t, "Semi-skimmed", stored.Description)
}

// R2
func TestTaskRepository_Update_PersistsEmptyDescription(t *testing.T) {
	// GIVEN: A task with a description stored in PostgreSQL.
	ctx := context.Background()
	repo := postgres.NewTaskRepository(setupTestDB(t))
	id, err := repo.Create(ctx, &task.Task{Title: "Buy milk", Description: "Semi-skimmed", UserID: 1, Completed: true})
	require.NoError(t, err)

	// WHEN: The task is updated with an empty description.
	err = repo.Update(ctx, &task.Task{ID: id, Title: "Buy milk", Description: "", UserID: 1, Completed: true})

	// THEN: The empty description is persisted and the other fields are unchanged.
	require.NoError(t, err)
	stored, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "", stored.Description, "empty description must be persisted")
	assert.Equal(t, "Buy milk", stored.Title)
	assert.True(t, stored.Completed)
}

// R3
func TestTaskRepository_Update_SadPath_MissingTask(t *testing.T) {
	// GIVEN: No task with this id.
	ctx := context.Background()
	repo := postgres.NewTaskRepository(setupTestDB(t))

	// WHEN: The missing task is updated.
	err := repo.Update(ctx, &task.Task{ID: 999, Title: "Ghost", UserID: 1, Completed: true})

	// THEN: ErrTaskNotFound is returned and no row is created.
	assert.ErrorIs(t, err, task.ErrTaskNotFound)
	_, getErr := repo.GetByID(ctx, 999)
	assert.ErrorIs(t, getErr, task.ErrTaskNotFound, "update must not create the missing task")
}
