package persistence

import (
	"context"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// maxContractID is the largest id allowed by the contract (uint32, openapi.yaml).
const maxContractID = uint(4294967295)

// GIVEN two stored users / WHEN List / THEN both are returned with their fields
func TestUserRepository_List_ReturnsAllUsers(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := postgres.NewUserRepository(db)
	adaID := createTestUser(t, db, "ada@example.com")
	bobID, err := repo.Create(ctx, &user.User{Name: "Bob", Email: "bob@example.com"})
	require.NoError(t, err)

	users, err := repo.List(ctx)

	require.NoError(t, err)
	got := make([]user.User, len(users))
	for i, u := range users {
		got[i] = *u
	}
	assert.ElementsMatch(t, []user.User{
		{ID: adaID, Name: "Owner", Email: "ada@example.com"},
		{ID: bobID, Name: "Bob", Email: "bob@example.com"},
	}, got)
}

// GIVEN no user / WHEN List / THEN an empty, non-nil slice
func TestUserRepository_List_EmptyTable(t *testing.T) {
	users, err := postgres.NewUserRepository(setupTestDB(t)).List(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, users)
	assert.Empty(t, users)
}

// GIVEN a stored user / WHEN GetByID with 0 or the largest contract id / THEN ErrUserNotFound
func TestUserRepository_GetByID_BoundaryIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewUserRepository(db)
	createTestUser(t, db, "owner@example.com")

	for _, id := range []uint{0, maxContractID} {
		found, err := repo.GetByID(context.Background(), id)

		assert.Nil(t, found, "id %d", id)
		assert.ErrorIs(t, err, user.ErrUserNotFound, "id %d", id)
	}
}

// GIVEN a long Unicode name and an email with special characters / WHEN stored and read back / THEN unchanged
func TestUserRepository_Create_KeepsLongUnicodeValues(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewUserRepository(setupTestDB(t))
	name := strings.Repeat("Zoë 🚀 ' OR 1=1 -- ", 600)
	email := "zoë.o'brien+tâches@exemple.fr"

	id, err := repo.Create(ctx, &user.User{Name: name, Email: email})
	require.NoError(t, err)

	stored, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, name, stored.Name)
	assert.Equal(t, email, stored.Email)
	byEmail, err := repo.GetByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, id, byEmail.ID)
}

// GIVEN a task owned by Alice / WHEN Update carries Bob's id / THEN the owner does not change
func TestTaskRepository_Update_NeverReassignsOwner(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)
	aliceID := createTestUser(t, db, "alice@example.com")
	bobID := createTestUser(t, db, "bob@example.com")
	id := createTestTask(t, repo, aliceID, "Task 1")

	err := repo.Update(ctx, &task.Task{ID: id, UserID: bobID, Title: "Task 1 edited", Description: "d", Completed: true})

	require.NoError(t, err)
	stored, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, task.Task{ID: id, UserID: aliceID, Title: "Task 1 edited", Description: "d", Completed: true}, *stored)
}

// GIVEN three tasks / WHEN one is deleted / THEN only that one disappears and Count follows
func TestTaskRepository_Delete_RemovesOnlyTargetedTask(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)
	ownerID := createTestUser(t, db, "owner@example.com")
	first := createTestTask(t, repo, ownerID, "Task 1")
	second := createTestTask(t, repo, ownerID, "Task 2")
	third := createTestTask(t, repo, ownerID, "Task 3")

	require.NoError(t, repo.Delete(ctx, second))

	_, err := repo.GetByID(ctx, second)
	assert.ErrorIs(t, err, task.ErrTaskNotFound)
	remaining, err := repo.ListByUserID(ctx, ownerID)
	require.NoError(t, err)
	require.Len(t, remaining, 2)
	assert.Equal(t, []uint{first, third}, []uint{remaining[0].ID, remaining[1].ID})
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

// GIVEN no task with these ids, then a deleted task / WHEN Delete / THEN ErrTaskNotFound and nothing else is removed (B-04)
func TestTaskRepository_Delete_AbsentTask_ReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)
	ownerID := createTestUser(t, db, "owner@example.com")
	kept := createTestTask(t, repo, ownerID, "Keep me")
	removed := createTestTask(t, repo, ownerID, "Delete me")

	for _, id := range []uint{0, 999, maxContractID} {
		assert.ErrorIs(t, repo.Delete(ctx, id), task.ErrTaskNotFound, "id %d", id)
	}
	require.NoError(t, repo.Delete(ctx, removed))
	assert.ErrorIs(t, repo.Delete(ctx, removed), task.ErrTaskNotFound, "a second delete must report the task as absent")

	remaining, err := repo.ListByUserID(ctx, ownerID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, kept, remaining[0].ID)
}

// GIVEN a stored user / WHEN Create with the same email / THEN ErrUserEmailTaken, no raw database error, nothing stored (B-03)
func TestUserRepository_Create_DuplicateEmail_ReturnsEmailTaken(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := postgres.NewUserRepository(db)
	_, err := repo.Create(ctx, &user.User{Name: "Ada", Email: "ada@x.io"})
	require.NoError(t, err)

	_, err = repo.Create(ctx, &user.User{Name: "Other", Email: "ada@x.io"})

	require.ErrorIs(t, err, user.ErrUserEmailTaken)
	assert.NotContains(t, err.Error(), "ada@x.io", "the error must not carry the email")
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

// GIVEN an empty table, then 2 users / WHEN Count / THEN 0, then 2
func TestUserRepository_Count(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := postgres.NewUserRepository(db)

	empty, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), empty)

	createTestUser(t, db, "alice@example.com")
	createTestUser(t, db, "bob@example.com")

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

// GIVEN tasks of several users / WHEN Count / THEN all tasks are counted, 0 on an empty table
func TestTaskRepository_Count(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)

	empty, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), empty)

	aliceID := createTestUser(t, db, "alice@example.com")
	bobID := createTestUser(t, db, "bob@example.com")
	createTestTask(t, repo, aliceID, "Task 1")
	createTestTask(t, repo, aliceID, "Task 2")
	createTestTask(t, repo, bobID, "Task 3")

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

// GIVEN a user without tasks while another user has some / WHEN ListByUserID / THEN an empty, non-nil slice
func TestTaskRepository_ListByUserID_UserWithoutTasks(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)
	busyID := createTestUser(t, db, "busy@example.com")
	idleID := createTestUser(t, db, "idle@example.com")
	createTestTask(t, repo, busyID, "Task 1")

	tasks, err := repo.ListByUserID(context.Background(), idleID)

	require.NoError(t, err)
	assert.NotNil(t, tasks)
	assert.Empty(t, tasks)
}

// GIVEN a stored task / WHEN GetByID with 0 or the largest contract id / THEN ErrTaskNotFound
func TestTaskRepository_GetByID_BoundaryIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)
	createTestTask(t, repo, createTestUser(t, db, "owner@example.com"), "Task 1")

	for _, id := range []uint{0, maxContractID} {
		found, err := repo.GetByID(context.Background(), id)

		assert.Nil(t, found, "id %d", id)
		assert.ErrorIs(t, err, task.ErrTaskNotFound, "id %d", id)
	}
}

// GIVEN a long Unicode title and a multi-line description with markup / WHEN stored and read back / THEN unchanged
func TestTaskRepository_Create_KeepsLongUnicodeValues(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := postgres.NewTaskRepository(db)
	ownerID := createTestUser(t, db, "owner@example.com")
	title := strings.Repeat("Tâche ✅ 日本語 ", 800)
	description := "Ligne 1\nLigne 2\t<script>alert('x')</script> ; DROP TABLE task_models;"

	id, err := repo.Create(ctx, &task.Task{Title: title, Description: description, UserID: ownerID})
	require.NoError(t, err)

	stored, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, title, stored.Title)
	assert.Equal(t, description, stored.Description)
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "the description must not be executed as SQL")
}
