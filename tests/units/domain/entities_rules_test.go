package domain

import (
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN: A user without email / WHEN: NewUser / THEN: ErrUserEmailEmpty
func TestNewUser_EmptyEmail(t *testing.T) {
	u, err := user.NewUser(1, "John Doe", "")

	assert.Nil(t, u)
	assert.ErrorIs(t, err, user.ErrUserEmailEmpty)
}

// GIVEN: Neither name nor email / WHEN: NewUser / THEN: the name is reported first
func TestNewUser_EmptyNameAndEmail_ReportsNameFirst(t *testing.T) {
	u, err := user.NewUser(1, "", "")

	assert.Nil(t, u)
	assert.ErrorIs(t, err, user.ErrUserNameEmpty)
}

// GIVEN: One-character name and email (contract minLength 1) / WHEN: NewUser / THEN: accepted
func TestNewUser_SingleCharacterFields_AreAccepted(t *testing.T) {
	u, err := user.NewUser(0, "J", "j")

	require.NoError(t, err)
	assert.Equal(t, user.User{ID: 0, Name: "J", Email: "j"}, *u)
}

// GIVEN: Long Unicode values / WHEN: NewUser / THEN: kept byte for byte
func TestNewUser_LongUnicodeFields_AreKeptUnchanged(t *testing.T) {
	name := strings.Repeat("Élodie 🚀 ", 1000)
	email := "élodie+tâches@exemple.fr"

	u, err := user.NewUser(7, name, email)

	require.NoError(t, err)
	assert.Equal(t, name, u.Name)
	assert.Equal(t, email, u.Email)
}

// GIVEN: A task without title / WHEN: NewTask / THEN: ErrTaskTitleEmpty
func TestNewTask_EmptyTitle(t *testing.T) {
	tk, err := task.NewTask(1, 1, "", "desc", false)

	assert.Nil(t, tk)
	assert.ErrorIs(t, err, task.ErrTaskTitleEmpty)
}

// GIVEN: A task with user id 0 / WHEN: NewTask / THEN: ErrTaskUserIDInvalid
func TestNewTask_ZeroUserID(t *testing.T) {
	tk, err := task.NewTask(1, 0, "Buy milk", "desc", false)

	assert.Nil(t, tk)
	assert.ErrorIs(t, err, task.ErrTaskUserIDInvalid)
}

// GIVEN: Neither title nor user id / WHEN: NewTask / THEN: the title is reported first
func TestNewTask_EmptyTitleAndZeroUserID_ReportsTitleFirst(t *testing.T) {
	tk, err := task.NewTask(1, 0, "", "", false)

	assert.Nil(t, tk)
	assert.ErrorIs(t, err, task.ErrTaskTitleEmpty)
}

// GIVEN: The smallest valid task (1-char title, user id 1, empty description, completed)
// WHEN: NewTask / THEN: every field is kept
func TestNewTask_BoundaryValues_AreAccepted(t *testing.T) {
	tk, err := task.NewTask(0, 1, "B", "", true)

	require.NoError(t, err)
	assert.Equal(t, task.Task{ID: 0, UserID: 1, Title: "B", Description: "", Completed: true}, *tk)
}

// GIVEN: Long Unicode title and description / WHEN: NewTask / THEN: kept byte for byte
func TestNewTask_LongUnicodeFields_AreKeptUnchanged(t *testing.T) {
	title := strings.Repeat("Tâche ✅ ", 1000)
	description := "Ligne 1\nLigne 2 <b>gras</b> ' OR 1=1 --"

	tk, err := task.NewTask(3, 2, title, description, false)

	require.NoError(t, err)
	assert.Equal(t, title, tk.Title)
	assert.Equal(t, description, tk.Description)
}

// GIVEN: A completed task / WHEN: UncompleteTask / THEN: completed is false, other fields unchanged
func TestUncompleteTask(t *testing.T) {
	tk, err := task.NewTask(1, 1, "Buy milk", "Get milk", true)
	require.NoError(t, err)

	tk.UncompleteTask()

	assert.Equal(t, task.Task{ID: 1, UserID: 1, Title: "Buy milk", Description: "Get milk", Completed: false}, *tk)
}
