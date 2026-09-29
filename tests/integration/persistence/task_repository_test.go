package persistence

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createTestUser persists a real user so that task fixtures reference an existing owner.
func createTestUser(t *testing.T, db *gorm.DB, email string) uint {
	t.Helper()
	id, err := postgres.NewUserRepository(db).Create(context.Background(), &user.User{Name: "Owner", Email: email})
	require.NoError(t, err)
	return id
}

func createTestTask(t *testing.T, repo *postgres.TaskRepository, userID uint, title string) uint {
	t.Helper()
	id, err := repo.Create(context.Background(), &task.Task{Title: title, Description: "Desc " + title, UserID: userID})
	require.NoError(t, err)
	return id
}

// GIVEN: TaskRepository / WHEN: Create valid task / THEN: Task saved with ID
func TestTaskRepository_Create_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)
	ownerID := createTestUser(t, db, "owner@example.com")

	tsk := &task.Task{Title: "Buy milk", Description: "Get milk from store", UserID: ownerID, Completed: false}
	id, err := repo.Create(context.Background(), tsk)

	require.NoError(t, err)
	assert.NotZero(t, id)

	// THEN: Verify task was persisted
	retrieved, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "Buy milk", retrieved.Title)
	assert.Equal(t, ownerID, retrieved.UserID)
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
	aliceID := createTestUser(t, db, "alice@example.com")
	bobID := createTestUser(t, db, "bob@example.com")

	createTestTask(t, repo, aliceID, "Task 1")
	createTestTask(t, repo, aliceID, "Task 2")
	createTestTask(t, repo, bobID, "Task 3")

	// THEN: List only Alice's tasks
	tasks, err := repo.ListByUserID(context.Background(), aliceID)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	assert.Equal(t, "Task 1", tasks[0].Title)
	assert.Equal(t, "Task 2", tasks[1].Title)
}

// GIVEN: A user's first task was updated after the others were created
// WHEN: The user's tasks are listed
// THEN: They are returned by ascending id, not in physical storage order
func TestTaskRepository_ListByUserID_OrderedByIDAfterUpdate(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)
	ownerID := createTestUser(t, db, "owner@example.com")
	first := createTestTask(t, repo, ownerID, "Task 1")
	second := createTestTask(t, repo, ownerID, "Task 2")
	third := createTestTask(t, repo, ownerID, "Task 3")

	// An UPDATE writes a new row version further in the heap (PostgreSQL MVCC),
	// so a sequential scan without ORDER BY would return it last.
	stored, err := repo.GetByID(ctx, first)
	require.NoError(t, err)
	stored.Title = "Task 1 edited"
	require.NoError(t, repo.Update(ctx, stored))

	tasks, err := repo.ListByUserID(ctx, ownerID)

	require.NoError(t, err)
	ids := make([]uint, len(tasks))
	for i, tk := range tasks {
		ids[i] = tk.ID
	}
	assert.Equal(t, []uint{first, second, third}, ids, "tasks must be ordered by id")
}
