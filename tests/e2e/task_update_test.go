package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type taskBody struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	UserID      uint   `json:"user_id"`
	Completed   bool   `json:"completed"`
}

type taskEnvelope struct {
	Success   bool     `json:"success"`
	Data      taskBody `json:"data"`
	Error     string   `json:"error"`
	ErrorCode string   `json:"error_code"`
}

// taskUpdateFixture seeds tasks directly in the database and updates them
// through the production router, with every response checked against the contract.
type taskUpdateFixture struct {
	s      *testServer
	userID uint
}

func newTaskUpdateFixture(t *testing.T) *taskUpdateFixture {
	t.Helper()
	s := newTestServer(t)

	user := &pgstore.UserModel{Name: "Owner", Email: "owner@example.com"}
	require.NoError(t, s.db.Create(user).Error)

	return &taskUpdateFixture{s: s, userID: user.ID}
}

func (f *taskUpdateFixture) seedTask(t *testing.T, title, description string, completed bool) uint {
	t.Helper()
	model := &pgstore.TaskModel{Title: title, Description: description, UserID: f.userID}
	require.NoError(t, f.s.db.Create(model).Error)
	if completed {
		require.NoError(t, f.s.db.Model(model).Update("completed", true).Error)
	}
	return model.ID
}

func (f *taskUpdateFixture) put(t *testing.T, id uint, body string) (int, taskEnvelope) {
	t.Helper()
	r := f.s.do(http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", id), body)

	var env taskEnvelope
	require.NoError(t, json.Unmarshal(r.body, &env), "invalid JSON body: %s", r.body)
	return r.code, env
}

func (f *taskUpdateFixture) persistedTasks(t *testing.T) []taskBody {
	t.Helper()
	var tasks []taskBody
	for _, tk := range f.s.listTasks(f.userID) {
		tasks = append(tasks, taskBody(tk))
	}
	return tasks
}

func (f *taskUpdateFixture) persistedTask(t *testing.T, id uint) taskBody {
	t.Helper()
	for _, tk := range f.persistedTasks(t) {
		if tk.ID == id {
			return tk
		}
	}
	require.FailNowf(t, "task not found", "task %d is not listed for user %d", id, f.userID)
	return taskBody{}
}

// U1
func TestTaskController_UpdateTask_UncompleteIsPersisted(t *testing.T) {
	// GIVEN: A completed task.
	f := newTaskUpdateFixture(t)
	id := f.seedTask(t, "Buy milk", "Semi-skimmed", true)

	// WHEN: Only completed=false is sent.
	code, resp := f.put(t, id, `{"completed":false}`)

	// THEN: The task is incomplete in the response and in storage.
	require.Equal(t, http.StatusOK, code)
	assert.False(t, resp.Data.Completed, "response must report completed=false")
	assert.False(t, f.persistedTask(t, id).Completed, "completed=false must be persisted")
}

// U2
func TestTaskController_UpdateTask_EmptyDescriptionIsPersisted(t *testing.T) {
	// GIVEN: A completed task with a description.
	f := newTaskUpdateFixture(t)
	id := f.seedTask(t, "Buy milk", "Semi-skimmed", true)

	// WHEN: Only an explicit empty description is sent.
	code, resp := f.put(t, id, `{"description":""}`)

	// THEN: The description is cleared and absent fields are unchanged.
	require.Equal(t, http.StatusOK, code)
	stored := f.persistedTask(t, id)
	assert.Empty(t, stored.Description, "empty description must be persisted")
	assert.Equal(t, "Buy milk", stored.Title)
	assert.True(t, stored.Completed)
	assert.Equal(t, stored.Title, resp.Data.Title, "response must reflect storage")
	assert.Equal(t, stored.Description, resp.Data.Description, "response must reflect storage")
	assert.Equal(t, stored.Completed, resp.Data.Completed, "response must reflect storage")
}

// U3
func TestTaskController_UpdateTask_CompleteOnlyKeepsOtherFields(t *testing.T) {
	// GIVEN: An open task.
	f := newTaskUpdateFixture(t)
	id := f.seedTask(t, "Buy milk", "Semi-skimmed", false)

	// WHEN: Only completed=true is sent.
	code, resp := f.put(t, id, `{"completed":true}`)

	// THEN: Title and description are unchanged in storage and in the response.
	require.Equal(t, http.StatusOK, code)
	stored := f.persistedTask(t, id)
	assert.Equal(t, taskBody{ID: id, Title: "Buy milk", Description: "Semi-skimmed", UserID: f.userID, Completed: true}, stored)
	assert.Equal(t, "Buy milk", resp.Data.Title, "response must not blank the absent title")
	assert.Equal(t, "Semi-skimmed", resp.Data.Description, "response must not blank the absent description")
	assert.True(t, resp.Data.Completed)
}

// U4
func TestTaskController_UpdateTask_TitleOnlyKeepsOtherFields(t *testing.T) {
	// GIVEN: A completed task.
	f := newTaskUpdateFixture(t)
	id := f.seedTask(t, "Buy milk", "Semi-skimmed", true)

	// WHEN: Only a new title is sent.
	code, resp := f.put(t, id, `{"title":"Buy oat milk"}`)

	// THEN: Description and completion are unchanged in storage and in the response.
	require.Equal(t, http.StatusOK, code)
	stored := f.persistedTask(t, id)
	assert.Equal(t, taskBody{ID: id, Title: "Buy oat milk", Description: "Semi-skimmed", UserID: f.userID, Completed: true}, stored)
	assert.Equal(t, "Buy oat milk", resp.Data.Title)
	assert.Equal(t, "Semi-skimmed", resp.Data.Description, "response must not blank the absent description")
	assert.True(t, resp.Data.Completed, "response must not reset the absent completed flag")
}

// U5
func TestTaskController_UpdateTask_SadPath_ExplicitEmptyTitle(t *testing.T) {
	// GIVEN: An existing task.
	f := newTaskUpdateFixture(t)
	id := f.seedTask(t, "Buy milk", "Semi-skimmed", false)

	// WHEN: An explicit empty title is sent.
	code, resp := f.put(t, id, `{"title":""}`)

	// THEN: The request is rejected and the title is unchanged.
	require.Equal(t, http.StatusBadRequest, code)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_FAILED", resp.ErrorCode)
	assert.Equal(t, "Task title cannot be empty", resp.Error)
	assert.Equal(t, "Buy milk", f.persistedTask(t, id).Title)
}

// U6 (guard)
func TestTaskController_UpdateTask_SadPath_UnknownTask(t *testing.T) {
	// GIVEN: No task with this id.
	f := newTaskUpdateFixture(t)

	// WHEN: The unknown task is updated.
	code, resp := f.put(t, 999, `{"completed":true}`)

	// THEN: 404 with the contract error message.
	require.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, "TASK_NOT_FOUND", resp.ErrorCode)
	assert.Equal(t, "Task not found", resp.Error)
}

// U7 (guard)
func TestTaskController_UpdateTask_SadPath_InvalidInput(t *testing.T) {
	// GIVEN: An existing task.
	f := newTaskUpdateFixture(t)
	id := f.seedTask(t, "Buy milk", "Semi-skimmed", false)

	cases := map[string]struct {
		path    string
		body    string
		code    string
		message string
	}{
		"non numeric id": {"/api/v1/tasks/abc", `{"completed":true}`, "INVALID_ID", "Invalid task ID"},
		"malformed body": {fmt.Sprintf("/api/v1/tasks/%d", id), `{"completed":`, "INVALID_BODY", "Invalid request body"},
		"wrong type":     {fmt.Sprintf("/api/v1/tasks/%d", id), `{"completed":"yes"}`, "INVALID_BODY", "Invalid request body"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// WHEN: The invalid request is sent.
			r := f.s.do(http.MethodPut, tc.path, tc.body)

			// THEN: 400 with the stable code, and the task is untouched.
			requireError(t, r, http.StatusBadRequest, tc.code, tc.message)
			assert.Equal(t, taskBody{ID: id, Title: "Buy milk", Description: "Semi-skimmed", UserID: f.userID}, f.persistedTask(t, id))
		})
	}
}

// U8 (guard)
func TestTaskController_UpdateTask_IdenticalValuesIsNotNotFound(t *testing.T) {
	// GIVEN: A completed task.
	f := newTaskUpdateFixture(t)
	id := f.seedTask(t, "Buy milk", "Semi-skimmed", true)

	// WHEN: The exact current values are sent again.
	code, resp := f.put(t, id, `{"title":"Buy milk","description":"Semi-skimmed","completed":true}`)

	// THEN: The idempotent update succeeds.
	require.Equal(t, http.StatusOK, code, resp.Error)
	assert.Equal(t, taskBody{ID: id, Title: "Buy milk", Description: "Semi-skimmed", UserID: f.userID, Completed: true}, f.persistedTask(t, id))
}

// U9 (guard)
func TestTaskController_UpdateTask_SadPath_DeletedTaskIsNotRecreated(t *testing.T) {
	// GIVEN: A task that has been deleted through the API.
	f := newTaskUpdateFixture(t)
	id := f.seedTask(t, "Buy milk", "Semi-skimmed", false)
	del := f.s.do(http.MethodDelete, fmt.Sprintf("/api/v1/tasks/%d", id), "")
	require.Equal(t, http.StatusOK, del.code, "body: %s", del.body)

	// WHEN: The deleted task is updated.
	code, _ := f.put(t, id, `{"completed":true}`)

	// THEN: 404 and the task does not come back.
	require.Equal(t, http.StatusNotFound, code)
	assert.Empty(t, f.persistedTasks(t))
}
