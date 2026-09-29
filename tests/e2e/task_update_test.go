package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	ctrlhttp "github.com/fdanielsin/inbackend/internal/controllers/http"
	pgimpl "github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type taskBody struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	UserID      uint   `json:"user_id"`
	Completed   bool   `json:"completed"`
}

type taskEnvelope struct {
	Success bool     `json:"success"`
	Data    taskBody `json:"data"`
	Error   string   `json:"error"`
}

type taskListEnvelope struct {
	Success bool       `json:"success"`
	Data    []taskBody `json:"data"`
}

type taskUpdateFixture struct {
	e      *echo.Echo
	db     *gorm.DB
	userID uint
}

func newTaskUpdateFixture(t *testing.T) *taskUpdateFixture {
	t.Helper()
	db := setupTestDB(t)

	taskRepo := pgimpl.NewTaskRepository(db)
	taskCtrl := ctrlhttp.NewTaskController(
		commands.NewCreateTaskHandler(taskRepo),
		commands.NewUpdateTaskHandler(taskRepo),
		commands.NewDeleteTaskHandler(taskRepo),
		queries.NewListUserTasksHandler(taskRepo),
	)

	e := echo.New()
	e.PUT("/api/v1/tasks/:id", taskCtrl.UpdateTask)
	e.DELETE("/api/v1/tasks/:id", taskCtrl.DeleteTask)
	e.GET("/api/v1/users/:id/tasks", taskCtrl.ListUserTasks)

	user := &pgimpl.UserModel{Name: "Owner", Email: "owner@example.com"}
	require.NoError(t, db.Create(user).Error)

	return &taskUpdateFixture{e: e, db: db, userID: user.ID}
}

func (f *taskUpdateFixture) seedTask(t *testing.T, title, description string, completed bool) uint {
	t.Helper()
	model := &pgimpl.TaskModel{Title: title, Description: description, UserID: f.userID}
	require.NoError(t, f.db.Create(model).Error)
	if completed {
		require.NoError(t, f.db.Model(model).Update("completed", true).Error)
	}
	return model.ID
}

func (f *taskUpdateFixture) put(t *testing.T, id uint, body string) (int, taskEnvelope) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", id), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)

	var env taskEnvelope
	if rec.Body.Len() > 0 {
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "invalid JSON body: %s", rec.Body.String())
	}
	return rec.Code, env
}

func (f *taskUpdateFixture) persistedTasks(t *testing.T) []taskBody {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%d/tasks", f.userID), nil)
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var env taskListEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	return env.Data
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
	assert.Equal(t, "", stored.Description, "empty description must be persisted")
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
	assert.Equal(t, "Task not found", resp.Error)
}

// U7 (guard)
func TestTaskController_UpdateTask_SadPath_InvalidInput(t *testing.T) {
	// GIVEN: An existing task.
	f := newTaskUpdateFixture(t)
	id := f.seedTask(t, "Buy milk", "Semi-skimmed", false)

	cases := map[string]struct {
		path string
		body string
	}{
		"non numeric id": {path: "/api/v1/tasks/abc", body: `{"completed":true}`},
		"malformed body": {path: fmt.Sprintf("/api/v1/tasks/%d", id), body: `{"completed":`},
		"wrong type":     {path: fmt.Sprintf("/api/v1/tasks/%d", id), body: `{"completed":"yes"}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// WHEN: The invalid request is sent.
			req := httptest.NewRequest(http.MethodPut, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			f.e.ServeHTTP(rec, req)

			// THEN: 400 and the task is untouched.
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
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
	del := httptest.NewRecorder()
	f.e.ServeHTTP(del, httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/tasks/%d", id), nil))
	require.Equal(t, http.StatusOK, del.Code)

	// WHEN: The deleted task is updated.
	code, _ := f.put(t, id, `{"completed":true}`)

	// THEN: 404 and the task does not come back.
	require.Equal(t, http.StatusNotFound, code)
	assert.Empty(t, f.persistedTasks(t))
}
