package persistence

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN: TaskRepository / WHEN: Create valid task / THEN: Task saved with ID
func TestTaskRepository_Create_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)

	tsk := &task.Task{Title: "Buy milk", Description: "Get milk from store", UserID: 1, Completed: false}
	id, err := repo.Create(context.Background(), tsk)

	require.NoError(t, err)
	assert.NotZero(t, id)

	// THEN: Verify task was persisted
	retrieved, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "Buy milk", retrieved.Title)
	assert.Equal(t, uint(1), retrieved.UserID)
}

// GIVEN: TaskRepository / WHEN: Get non-existent task / THEN: Error ErrTaskNotFound
func TestTaskRepository_GetByID_SadPath_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)

	retrieved, err := repo.GetByID(context.Background(), 999)

	require.Error(t, err)
	assert.Nil(t, retrieved)
	assert.Equal(t, task.ErrTaskNotFound, err)
}

// GIVEN: TaskRepository with multiple tasks / WHEN: List by UserID / THEN: Return only user's tasks
func TestTaskRepository_ListByUserID_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)

	t1 := &task.Task{Title: "Task 1", Description: "Desc 1", UserID: 1, Completed: false}
	t2 := &task.Task{Title: "Task 2", Description: "Desc 2", UserID: 1, Completed: false}
	t3 := &task.Task{Title: "Task 3", Description: "Desc 3", UserID: 2, Completed: false}

	repo.Create(context.Background(), t1)
	repo.Create(context.Background(), t2)
	repo.Create(context.Background(), t3)

	// THEN: List only user 1's tasks
	tasks, err := repo.ListByUserID(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "Task 1", tasks[0].Title)
	assert.Equal(t, "Task 2", tasks[1].Title)
}
