package persistence

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/user"
	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN: UserRepository / WHEN: Create valid user / THEN: User saved with ID
func TestUserRepository_Create_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewUserRepository(db)

	u := &user.User{Name: "John Doe", Email: "john@example.com"}
	id, err := repo.Create(context.Background(), u)

	require.NoError(t, err)
	assert.NotZero(t, id)

	// THEN: Verify user was persisted
	retrieved, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "John Doe", retrieved.Name)
	assert.Equal(t, "john@example.com", retrieved.Email)
}

// GIVEN: UserRepository / WHEN: Get non-existent user / THEN: Error ErrUserNotFound
func TestUserRepository_GetByID_SadPath_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewUserRepository(db)

	retrieved, err := repo.GetByID(context.Background(), 999)

	require.Error(t, err)
	assert.Nil(t, retrieved)
	assert.Equal(t, user.ErrUserNotFound, err)
}

// GIVEN: UserRepository with multiple users / WHEN: Get by email / THEN: Return matching user
func TestUserRepository_GetByEmail_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewUserRepository(db)

	u1 := &user.User{Name: "John", Email: "john@example.com"}
	u2 := &user.User{Name: "Jane", Email: "jane@example.com"}

	_, err := repo.Create(context.Background(), u1)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), u2)
	require.NoError(t, err)

	// THEN: Find by email
	found, err := repo.GetByEmail(context.Background(), "jane@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Jane", found.Name)
	assert.Equal(t, "jane@example.com", found.Email)
}

// GIVEN: UserRepository / WHEN: Get by non-existent email / THEN: Error or nil
func TestUserRepository_GetByEmail_SadPath_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewUserRepository(db)

	found, err := repo.GetByEmail(context.Background(), "nonexistent@example.com")

	require.Error(t, err)
	assert.Nil(t, found)
}
