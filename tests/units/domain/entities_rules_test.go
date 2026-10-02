package domain

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN: A user without email / WHEN: NewUser / THEN: ErrUserEmailEmpty
func TestNewUser_EmptyEmail(t *testing.T) {
	u, err := users.NewUser(1, "John Doe", "")

	assert.Nil(t, u)
	assert.ErrorIs(t, err, users.ErrUserEmailEmpty)
}

// GIVEN: Neither name nor email / WHEN: NewUser / THEN: the name is reported first
func TestNewUser_EmptyNameAndEmail_ReportsNameFirst(t *testing.T) {
	u, err := users.NewUser(1, "", "")

	assert.Nil(t, u)
	assert.ErrorIs(t, err, users.ErrUserNameEmpty)
}

// GIVEN: A one-character name and the shortest valid email / WHEN: NewUser / THEN: accepted
func TestNewUser_ShortestValidFields_AreAccepted(t *testing.T) {
	u, err := users.NewUser(0, "J", "a@b.c")

	require.NoError(t, err)
	assert.Equal(t, users.User{ID: 0, Name: "J", Email: "a@b.c"}, *u)
}

// GIVEN: An email with surrounding spaces and capitals, ASCII or not / WHEN: NewUser / THEN: trimmed and lower-cased; the name is kept as typed
func TestNewUser_NormalizesEmailOnly(t *testing.T) {
	for raw, want := range map[string]string{
		"  Ada@Example.COM\t":  "ada@example.com",
		"\nÉLODIE@EXEMPLE.FR ": "élodie@exemple.fr",
		"ada@mail.example.co":  "ada@mail.example.co",
	} {
		t.Run(fmt.Sprintf("%q", raw), func(t *testing.T) {
			u, err := users.NewUser(0, "Ada LOVELACE", raw)

			require.NoError(t, err, raw)
			assert.Equal(t, want, u.Email, raw)
			assert.Equal(t, "Ada LOVELACE", u.Name, raw)
		})
	}
}

// GIVEN: An email made of spaces only / WHEN: NewUser / THEN: ErrUserEmailEmpty, checked before the format
func TestNewUser_BlankEmail_IsEmpty(t *testing.T) {
	u, err := users.NewUser(0, "Ada", " \t\n ")

	assert.Nil(t, u)
	assert.ErrorIs(t, err, users.ErrUserEmailEmpty)
}

// GIVEN: An email without the minimal shape local@domain.tld (current rule) / WHEN: NewUser / THEN: ErrUserEmailInvalid
func TestNewUser_MalformedEmail_IsInvalid(t *testing.T) {
	for _, email := range []string{
		"ada.example.com",   // no @
		"ada@@example.com",  // two @
		"ada@exa@mple.com",  // @ in the domain
		"@example.com",      // empty local part
		"ada@",              // empty domain
		"ada@localhost",     // domain without a dot
		"ada@.com",          // empty first label
		"ada@example.",      // empty last label
		"ada lovelace@x.io", // inner space
		"ada@exa..mple.com", // empty middle label
		"ada\u00a0l@x.io",   // inner no-break space
		"ada\x00@x.io",      // control character
		"\x00ada@x.io",      // control character first (not removed by the trim)
	} {
		t.Run(fmt.Sprintf("%q", email), func(t *testing.T) {
			u, err := users.NewUser(0, "Ada", email)

			assert.Nil(t, u, email)
			assert.ErrorIs(t, err, users.ErrUserEmailInvalid, email)
		})
	}
}

// GIVEN: An empty name and a malformed email / WHEN: NewUser / THEN: the name is reported first
func TestNewUser_EmptyNameAndMalformedEmail_ReportsNameFirst(t *testing.T) {
	_, err := users.NewUser(0, "", "not-an-email")

	assert.ErrorIs(t, err, users.ErrUserNameEmpty)
}

// GIVEN: Long Unicode values / WHEN: NewUser / THEN: kept byte for byte
func TestNewUser_LongUnicodeFields_AreKeptUnchanged(t *testing.T) {
	name := strings.TrimSpace(strings.Repeat("Élodie 🚀 ", 1000))
	email := "élodie+tâches@exemple.fr"

	u, err := users.NewUser(7, name, email)

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
	title := strings.TrimSpace(strings.Repeat("Tâche ✅ ", 1000))
	description := "Ligne 1\nLigne 2 <b>gras</b> ' OR 1=1 --"

	tk, err := task.NewTask(3, 2, title, description, false)

	require.NoError(t, err)
	assert.Equal(t, title, tk.Title)
	assert.Equal(t, description, tk.Description)
}
