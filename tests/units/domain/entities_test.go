package domain

import (
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/tasks"
	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN: User entity / WHEN: Create with valid data / THEN: User created
func TestNewUser_Valid(t *testing.T) {
	u, err := users.NewUser(1, "John Doe", "john@example.com")
	require.NoError(t, err)
	assert.Equal(t, uint(1), u.ID)
	assert.Equal(t, "John Doe", u.Name)
	assert.Equal(t, "john@example.com", u.Email)
}

// GIVEN: User entity / WHEN: Create with empty name / THEN: Error
func TestNewUser_EmptyName(t *testing.T) {
	u, err := users.NewUser(1, "", "john@example.com")
	require.Error(t, err)
	assert.Nil(t, u)
	assert.Equal(t, users.ErrUserNameEmpty, err)
}

// GIVEN: Task entity / WHEN: Create with valid data / THEN: Task created
func TestNewTask_Valid(t *testing.T) {
	tk, err := tasks.NewTask(1, 1, "Buy milk", "Get milk from store", false)
	require.NoError(t, err)
	assert.Equal(t, uint(1), tk.ID)
	assert.Equal(t, uint(1), tk.UserID)
	assert.Equal(t, "Buy milk", tk.Title)
	assert.False(t, tk.Completed)
}
