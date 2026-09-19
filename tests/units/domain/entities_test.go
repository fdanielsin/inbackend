package domain

import (
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN: User entity / WHEN: Create with valid data / THEN: User created
func TestNewUser_Valid(t *testing.T) {
	u, err := user.NewUser(1, "John Doe", "john@example.com")
	require.NoError(t, err)
	assert.Equal(t, uint(1), u.ID)
	assert.Equal(t, "John Doe", u.Name)
	assert.Equal(t, "john@example.com", u.Email)
}

// GIVEN: User entity / WHEN: Create with empty name / THEN: Error
func TestNewUser_EmptyName(t *testing.T) {
	u, err := user.NewUser(1, "", "john@example.com")
	require.Error(t, err)
	assert.Nil(t, u)
	assert.Equal(t, user.ErrUserNameEmpty, err)
}

// GIVEN: Task entity / WHEN: Create with valid data / THEN: Task created
func TestNewTask_Valid(t *testing.T) {
	tk, err := task.NewTask(1, 1, "Buy milk", "Get milk from store", false)
	require.NoError(t, err)
	assert.Equal(t, uint(1), tk.ID)
	assert.Equal(t, uint(1), tk.UserID)
	assert.Equal(t, "Buy milk", tk.Title)
	assert.Equal(t, false, tk.Completed)
}

// GIVEN: Task entity / WHEN: Complete task / THEN: Status changes
func TestCompleteTask(t *testing.T) {
	tk, _ := task.NewTask(1, 1, "Buy milk", "Get milk", false)
	assert.Equal(t, false, tk.Completed)
	tk.CompleteTask()
	assert.Equal(t, true, tk.Completed)
}
