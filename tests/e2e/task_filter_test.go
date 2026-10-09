package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *testServer) completeTask(id uint) {
	s.t.Helper()
	r := s.do(http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", id), `{"completed":true}`)
	require.Equal(s.t, http.StatusOK, r.code, "body: %s", r.body)
}

func taskIDsOf(tasks []taskJSON) []uint {
	ids := make([]uint, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID
	}
	return ids
}

func (s *testServer) listTasksFiltered(userID uint, query string) response {
	s.t.Helper()
	return s.do(http.MethodGet, fmt.Sprintf("/api/v1/users/%d/tasks%s", userID, query), "")
}

// GIVEN a user with completed and pending tasks, and another user's tasks
// WHEN GET /users/{id}/tasks with completed=true, completed=false and without the parameter
// THEN only the completed, only the pending, then all tasks of that user, by ascending id
func TestListUserTasks_FiltersOnCompleted(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")
	other := s.createUser("Other", "other@example.com")
	first := s.createTask(owner.ID, "First", "")
	second := s.createTask(owner.ID, "Second", "")
	third := s.createTask(owner.ID, "Third", "")
	s.createTask(other.ID, "Not mine", "")
	s.completeTask(first.ID)
	s.completeTask(third.ID)

	for name, tc := range map[string]struct {
		query string
		want  []uint
	}{
		"completed":       {"?completed=true", []uint{first.ID, third.ID}},
		"not completed":   {"?completed=false", []uint{second.ID}},
		"no parameter":    {"", []uint{first.ID, second.ID, third.ID}},
		"unknown ignored": {"?other=1", []uint{first.ID, second.ID, third.ID}},
		"other and valid": {"?other=1&completed=true", []uint{first.ID, third.ID}},
	} {
		t.Run(name, func(t *testing.T) {
			r := s.in(t).listTasksFiltered(owner.ID, tc.query)

			require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
			listed := decode[[]taskJSON](t, r).Data
			assert.Equal(t, tc.want, taskIDsOf(listed))
			for _, task := range listed {
				assert.Equal(t, owner.ID, task.UserID)
			}
		})
	}
}

// GIVEN a user whose tasks are all pending, all completed, or who has none
// WHEN the filter selects nothing / THEN 200 with an empty list (never null), like an unknown user
func TestListUserTasks_FilterSelectingNothing_ReturnsEmptyList(t *testing.T) {
	s := newTestServer(t)
	pending := s.createUser("Pending", "pending@example.com")
	done := s.createUser("Done", "done@example.com")
	idle := s.createUser("Idle", "idle@example.com")
	s.createTask(pending.ID, "Pending", "")
	s.completeTask(s.createTask(done.ID, "Done", "").ID)

	for name, tc := range map[string]struct {
		userID uint
		query  string
	}{
		"pending user, completed":  {pending.ID, "?completed=true"},
		"done user, not completed": {done.ID, "?completed=false"},
		"idle user, completed":     {idle.ID, "?completed=true"},
		"idle user, not completed": {idle.ID, "?completed=false"},
		"unknown user, completed":  {4294967295, "?completed=true"},
	} {
		t.Run(name, func(t *testing.T) {
			r := s.in(t).listTasksFiltered(tc.userID, tc.query)

			require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
			assert.JSONEq(t, `{"success":true,"data":[]}`, string(r.body))
		})
	}
}

// GIVEN a completed filter that is not exactly "true" or "false" (case, number, word, empty, repeated)
// WHEN GET and HEAD /users/{id}/tasks / THEN 400 VALIDATION_FAILED with the localized message, nothing changed
func TestListUserTasks_InvalidCompletedFilter_Returns400(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")
	s.createTask(owner.ID, "Keep", "")
	base := liveServer(t, s)

	for name, query := range map[string]string{
		"capitalized":        "?completed=True",
		"upper case":         "?completed=FALSE",
		"number":             "?completed=1",
		"zero":               "?completed=0",
		"word":               "?completed=yes",
		"short form":         "?completed=t",
		"empty value":        "?completed=",
		"no value":           "?completed",
		"spaces":             "?completed=%20true",
		"encoded true":       "?completed=tru%65%20",
		"repeated identical": "?completed=true&completed=true",
		"repeated different": "?completed=true&completed=false",
		"repeated, one bad":  "?completed=true&completed=x",
		"very long":          "?completed=" + strings.Repeat("x", 5000),
		"injection":          "?completed=true%27%20OR%201%3D1--",
		"unicode":            "?completed=vrai%E2%9C%93",
	} {
		t.Run(name, func(t *testing.T) {
			path := fmt.Sprintf("/api/v1/users/%d/tasks%s", owner.ID, query)

			requireError(t, s.in(t).do(http.MethodGet, path, ""), http.StatusBadRequest, "VALIDATION_FAILED", english(shared.MsgTaskCompletedFilter))
			head := callOperation(t, http.MethodHead, base+path, "", nil)
			assert.Equal(t, http.StatusBadRequest, head.status)
			assert.Empty(t, head.body)
		})
	}
	assert.Len(t, s.listTasks(owner.ID), 1)
}

// GIVEN an invalid completed filter and a French Accept-Language / THEN the French message, same code
func TestListUserTasks_InvalidCompletedFilter_IsLocalized(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")

	r := s.sendWithLanguages(http.MethodGet, fmt.Sprintf("/api/v1/users/%d/tasks?completed=oui", owner.ID), "", true, "fr")

	requireError(t, r, http.StatusBadRequest, "VALIDATION_FAILED", shared.Message(shared.LangFrench, shared.MsgTaskCompletedFilter))
	requireLanguageHeaders(t, r, "fr")
}

// GIVEN an invalid user id and an invalid completed filter / THEN the user id is reported first (INVALID_ID)
func TestListUserTasks_InvalidIDIsCheckedBeforeTheFilter(t *testing.T) {
	s := newTestServer(t)

	requireError(t, s.do(http.MethodGet, "/api/v1/users/abc/tasks?completed=maybe", ""), http.StatusBadRequest, "INVALID_ID", english(shared.MsgInvalidUserID))
}

// GIVEN the database is down / WHEN GET with a valid filter / THEN 500 without detail, like without the filter
func TestListUserTasks_FilterWithDatabaseDown_Returns500(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")
	s.closeDatabase()

	requireError(t, s.do(http.MethodGet, fmt.Sprintf("/api/v1/users/%d/tasks?completed=false", owner.ID), ""),
		http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
}
