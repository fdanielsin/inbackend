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

// createTestUser persists a real user so that task fixtures reference an existing owner.
func createTestUser(t *testing.T, db *gorm.DB, email string) uint {
	t.Helper()
	id, err := pgstore.NewUserRepository(db).Create(context.Background(), &users.User{Name: "Owner", Email: email})
	require.NoError(t, err)
	return id
}

func createTestTask(t *testing.T, repo *pgstore.TaskRepository, userID uint, title string) uint {
	t.Helper()
	id, err := repo.Create(context.Background(), &tasks.Task{Title: title, Description: "Desc " + title, UserID: userID})
	require.NoError(t, err)
	return id
}

// GIVEN: TaskRepository / WHEN: Create valid task / THEN: Task saved with ID
func TestTaskRepository_Create_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	repo := pgstore.NewTaskRepository(db)
	ownerID := createTestUser(t, db, "owner@example.com")

	tsk := &tasks.Task{Title: "Buy milk", Description: "Get milk from store", UserID: ownerID, Completed: false}
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
	repo := pgstore.NewTaskRepository(db)

	retrieved, err := repo.GetByID(context.Background(), 999)

	require.Error(t, err)
	assert.Nil(t, retrieved)
	assert.Equal(t, tasks.ErrTaskNotFound, err)
}

// GIVEN: TaskRepository with multiple tasks / WHEN: List by UserID / THEN: Return only user's tasks
func TestTaskRepository_ListByUserID_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	repo := pgstore.NewTaskRepository(db)
	aliceID := createTestUser(t, db, "alice@example.com")
	bobID := createTestUser(t, db, "bob@example.com")

	createTestTask(t, repo, aliceID, "Task 1")
	createTestTask(t, repo, aliceID, "Task 2")
	createTestTask(t, repo, bobID, "Task 3")

	// THEN: List only Alice's tasks
	aliceTasks, err := repo.ListByUserID(context.Background(), aliceID, tasks.ListFilter{})
	require.NoError(t, err)
	require.Len(t, aliceTasks, 2)
	assert.Equal(t, "Task 1", aliceTasks[0].Title)
	assert.Equal(t, "Task 2", aliceTasks[1].Title)
}

// GIVEN: A user's first task was updated after the others were created
// WHEN: The user's tasks are listed
// THEN: They are returned by ascending id, not in physical storage order
func TestTaskRepository_ListByUserID_OrderedByIDAfterUpdate(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := pgstore.NewTaskRepository(db)
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

	listed, err := repo.ListByUserID(ctx, ownerID, tasks.ListFilter{})

	require.NoError(t, err)
	ids := make([]uint, len(listed))
	for i, tk := range listed {
		ids[i] = tk.ID
	}
	assert.Equal(t, []uint{first, second, third}, ids, "tasks must be ordered by id")
}

// completeTestTask marks a stored task as completed.
func completeTestTask(t *testing.T, repo *pgstore.TaskRepository, id uint) {
	t.Helper()
	stored, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	stored.Completed = true
	require.NoError(t, repo.Update(context.Background(), stored))
}

func taskIDs(listed []*tasks.Task) []uint {
	ids := make([]uint, len(listed))
	for i, task := range listed {
		ids[i] = task.ID
	}
	return ids
}

// GIVEN a user with completed and pending tasks, and another user with both
// WHEN ListByUserID with no filter, true and false
// THEN all, only the completed, only the pending tasks of that user, each by ascending id
func TestTaskRepository_ListByUserID_FiltersOnCompleted(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := pgstore.NewTaskRepository(db)
	aliceID := createTestUser(t, db, "alice@example.com")
	bobID := createTestUser(t, db, "bob@example.com")
	first := createTestTask(t, repo, aliceID, "Task 1")
	second := createTestTask(t, repo, aliceID, "Task 2")
	third := createTestTask(t, repo, aliceID, "Task 3")
	bobTask := createTestTask(t, repo, bobID, "Bob task")
	completeTestTask(t, repo, first)
	completeTestTask(t, repo, third)
	completeTestTask(t, repo, bobTask)
	yes, no := true, false

	for name, tc := range map[string]struct {
		filter tasks.ListFilter
		want   []uint
	}{
		"no filter":     {tasks.ListFilter{}, []uint{first, second, third}},
		"completed":     {tasks.ListFilter{Completed: &yes}, []uint{first, third}},
		"not completed": {tasks.ListFilter{Completed: &no}, []uint{second}},
	} {
		t.Run(name, func(t *testing.T) {
			listed, err := repo.ListByUserID(ctx, aliceID, tc.filter)

			require.NoError(t, err)
			assert.Equal(t, tc.want, taskIDs(listed))
			for _, task := range listed {
				assert.Equal(t, aliceID, task.UserID)
			}
		})
	}
}

// GIVEN a user whose tasks are all completed, or all pending, or who has none
// WHEN ListByUserID with each filter / THEN the matching set, or an empty non-nil slice
func TestTaskRepository_ListByUserID_FilterBoundaries(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := pgstore.NewTaskRepository(db)
	doneUserID := createTestUser(t, db, "done@example.com")
	pendingUserID := createTestUser(t, db, "pending@example.com")
	idleUserID := createTestUser(t, db, "idle@example.com")
	doneTask := createTestTask(t, repo, doneUserID, "Done")
	completeTestTask(t, repo, doneTask)
	pendingTask := createTestTask(t, repo, pendingUserID, "Pending")
	yes, no := true, false

	for name, tc := range map[string]struct {
		userID uint
		filter tasks.ListFilter
		want   []uint
	}{
		"all completed, completed":     {doneUserID, tasks.ListFilter{Completed: &yes}, []uint{doneTask}},
		"all completed, not completed": {doneUserID, tasks.ListFilter{Completed: &no}, []uint{}},
		"all pending, completed":       {pendingUserID, tasks.ListFilter{Completed: &yes}, []uint{}},
		"all pending, not completed":   {pendingUserID, tasks.ListFilter{Completed: &no}, []uint{pendingTask}},
		"no task, completed":           {idleUserID, tasks.ListFilter{Completed: &yes}, []uint{}},
		"no task, not completed":       {idleUserID, tasks.ListFilter{Completed: &no}, []uint{}},
	} {
		t.Run(name, func(t *testing.T) {
			listed, err := repo.ListByUserID(ctx, tc.userID, tc.filter)

			require.NoError(t, err)
			assert.NotNil(t, listed)
			assert.Equal(t, tc.want, taskIDs(listed))
		})
	}
}
