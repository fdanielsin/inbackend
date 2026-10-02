package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN an existing user / WHEN POST /api/v1/tasks / THEN 201 with the task (completed=false by default), listed afterwards
func TestCreateTask_ReturnsAndPersistsTask(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")

	r := s.do(http.MethodPost, "/api/v1/tasks", fmt.Sprintf(`{"user_id":%d,"title":"Buy milk","description":"Semi-skimmed"}`, owner.ID))

	require.Equal(t, http.StatusCreated, r.code, "body: %s", r.body)
	env := decode[taskJSON](t, r)
	assert.True(t, env.Success)
	assert.Equal(t, "Task created", env.Message)
	want := taskJSON{ID: 1, Title: "Buy milk", Description: "Semi-skimmed", UserID: owner.ID, Completed: false}
	assert.Equal(t, want, env.Data)
	assert.Equal(t, []taskJSON{want}, s.listTasks(owner.ID))
}

// GIVEN completed=true and no description / WHEN POST / THEN both are stored as sent (description "")
func TestCreateTask_CompletedTrueAndNoDescription(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")

	r := s.do(http.MethodPost, "/api/v1/tasks", fmt.Sprintf(`{"user_id":%d,"title":"Done already","completed":true}`, owner.ID))

	require.Equal(t, http.StatusCreated, r.code, "body: %s", r.body)
	want := taskJSON{ID: 1, Title: "Done already", Description: "", UserID: owner.ID, Completed: true}
	assert.Equal(t, want, decode[taskJSON](t, r).Data)
	assert.Equal(t, []taskJSON{want}, s.listTasks(owner.ID))
}

// GIVEN the smallest valid task (1-char title, user id 1) / WHEN POST / THEN 201
func TestCreateTask_BoundaryValues(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")
	require.Equal(t, uint(1), owner.ID)

	created := s.createTask(1, "B", "")

	assert.Equal(t, taskJSON{ID: 1, Title: "B", Description: "", UserID: 1}, created)
}

// GIVEN Unicode, emoji, quotes and markup / WHEN POST then list / THEN returned byte for byte
func TestCreateTask_KeepsSpecialCharacters(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")
	title := `Tâche ✅ 日本語 "quoted" <script>alert(1)</script>`
	description := "Ligne 1\nLigne 2\t' OR 1=1 --"

	created := s.createTask(owner.ID, title, description)

	assert.Equal(t, []taskJSON{{ID: created.ID, Title: title, Description: description, UserID: owner.ID}}, s.listTasks(owner.ID))
}

// GIVEN an invalid payload / WHEN POST /api/v1/tasks / THEN 400 with the exact message and nothing stored
func TestCreateTask_RejectsInvalidPayloads(t *testing.T) {
	cases := map[string]struct {
		body    string
		code    string
		message string
	}{
		"empty title":         {`{"user_id":1,"title":""}`, "VALIDATION_FAILED", "Task title cannot be empty"},
		"missing title":       {`{"user_id":1}`, "VALIDATION_FAILED", "Task title cannot be empty"},
		"user id 0":           {`{"user_id":0,"title":"Buy milk"}`, "VALIDATION_FAILED", "Task user_id cannot be empty"},
		"missing user id":     {`{"title":"Buy milk"}`, "VALIDATION_FAILED", "Task user_id cannot be empty"},
		"negative user id":    {`{"user_id":-1,"title":"Buy milk"}`, "INVALID_BODY", "Invalid request body"},
		"user id as a string": {`{"user_id":"1","title":"Buy milk"}`, "INVALID_BODY", "Invalid request body"},
		"malformed JSON":      {`{"user_id":1,"title":`, "INVALID_BODY", "Invalid request body"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)
			s.createUser("Owner", "owner@example.com")

			requireError(t, s.do(http.MethodPost, "/api/v1/tasks", tc.body), http.StatusBadRequest, tc.code, tc.message)
			assert.Zero(t, s.countRows(&pgstore.TaskModel{}), "nothing may be stored")
		})
	}
}

// GIVEN no user with this id / WHEN POST /api/v1/tasks for it / THEN 404 USER_NOT_FOUND and nothing stored (B-02)
func TestCreateTask_UnknownUserIsNotFound(t *testing.T) {
	s := newTestServer(t)
	s.createUser("Owner", "owner@example.com")

	for _, userID := range []string{"999", "4294967295"} {
		r := s.do(http.MethodPost, "/api/v1/tasks", `{"user_id":`+userID+`,"title":"Orphan"}`)

		requireError(t, r, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}
	assert.Zero(t, s.countRows(&pgstore.TaskModel{}), "no orphan task may be stored")
}

// GIVEN tasks of two users / WHEN GET /api/v1/users/{id}/tasks / THEN only this user's tasks, by ascending id
func TestListUserTasks_OnlyThisUserInIDOrder(t *testing.T) {
	s := newTestServer(t)
	alice := s.createUser("Alice", "alice@example.com")
	bob := s.createUser("Bob", "bob@example.com")
	first := s.createTask(alice.ID, "Task 1", "a")
	s.createTask(bob.ID, "Task of Bob", "b")
	second := s.createTask(alice.ID, "Task 2", "")

	r := s.do(http.MethodGet, fmt.Sprintf("/api/v1/users/%d/tasks", alice.ID), "")

	require.Equal(t, http.StatusOK, r.code)
	env := decode[[]taskJSON](t, r)
	assert.True(t, env.Success)
	assert.Equal(t, []taskJSON{first, second}, env.Data)
}

// GIVEN a user without tasks, an unknown user and boundary ids / WHEN listing tasks / THEN data is []
func TestListUserTasks_EmptyListsAreArrays(t *testing.T) {
	s := newTestServer(t)
	idle := s.createUser("Idle", "idle@example.com")
	busy := s.createUser("Busy", "busy@example.com")
	s.createTask(busy.ID, "Task", "")

	for _, id := range []string{fmt.Sprint(idle.ID), "999", "0", "4294967295"} {
		r := s.do(http.MethodGet, "/api/v1/users/"+id+"/tasks", "")
		require.Equal(t, http.StatusOK, r.code, "user %s", id)
		assert.JSONEq(t, `{"success":true,"data":[]}`, string(r.body), "user %s", id)
	}
}

// GIVEN an invalid user id / WHEN listing tasks / THEN 400 « Invalid user ID »
func TestListUserTasks_InvalidIDs(t *testing.T) {
	s := newTestServer(t)

	for _, id := range []string{"abc", "-1", "4294967296"} {
		requireError(t, s.do(http.MethodGet, "/api/v1/users/"+id+"/tasks", ""), http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
	}
}

// GIVEN two tasks / WHEN DELETE one / THEN 200 with a message and no data, only that task is gone
func TestDeleteTask_RemovesOnlyTargetedTask(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")
	kept := s.createTask(owner.ID, "Keep me", "")
	removed := s.createTask(owner.ID, "Delete me", "")

	r := s.do(http.MethodDelete, fmt.Sprintf("/api/v1/tasks/%d", removed.ID), "")

	require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
	assert.JSONEq(t, `{"success":true,"message":"Task deleted"}`, string(r.body))
	assert.Equal(t, []taskJSON{kept}, s.listTasks(owner.ID))
}

// GIVEN an invalid task id / WHEN DELETE / THEN 400 « Invalid task ID » and nothing deleted
func TestDeleteTask_InvalidIDs(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")
	task := s.createTask(owner.ID, "Keep me", "")

	for _, id := range []string{"abc", "-1", "4294967296"} {
		requireError(t, s.do(http.MethodDelete, "/api/v1/tasks/"+id, ""), http.StatusBadRequest, "INVALID_ID", "Invalid task ID")
	}
	assert.Equal(t, []taskJSON{task}, s.listTasks(owner.ID))
}

// GIVEN a task and an unknown id / WHEN DELETE the unknown id, or the same task twice / THEN 404 TASK_NOT_FOUND and the other tasks stay (B-04)
func TestDeleteTask_AbsentTaskIsNotFound(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")
	kept := s.createTask(owner.ID, "Keep me", "")
	removed := s.createTask(owner.ID, "Delete me", "")

	requireError(t, s.do(http.MethodDelete, "/api/v1/tasks/999", ""), http.StatusNotFound, "TASK_NOT_FOUND", "Task not found")

	first := s.do(http.MethodDelete, fmt.Sprintf("/api/v1/tasks/%d", removed.ID), "")
	require.Equal(t, http.StatusOK, first.code, "body: %s", first.body)
	second := s.do(http.MethodDelete, fmt.Sprintf("/api/v1/tasks/%d", removed.ID), "")
	requireError(t, second, http.StatusNotFound, "TASK_NOT_FOUND", "Task not found")

	assert.Equal(t, []taskJSON{kept}, s.listTasks(owner.ID))
}

// GIVEN valid ids at the uint32 boundaries with no task / WHEN DELETE / THEN 404 TASK_NOT_FOUND (B-04 boundaries)
func TestDeleteTask_BoundaryIDsAreNotFound(t *testing.T) {
	s := newTestServer(t)

	for _, id := range []string{"0", "4294967295"} {
		requireError(t, s.do(http.MethodDelete, "/api/v1/tasks/"+id, ""), http.StatusNotFound, "TASK_NOT_FOUND", "Task not found")
	}
}

type updatedTaskJSON struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// GIVEN a task / WHEN PUT with an empty object / THEN 200 and nothing changes
// (title null is refused since D4: TestRequestValidation_RefusesNullOnOptionalFields)
func TestUpdateTask_AbsentAndNullFieldsAreUnchanged(t *testing.T) {
	for name, body := range map[string]string{"empty object": `{}`} {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)
			owner := s.createUser("Owner", "owner@example.com")
			task := s.createTask(owner.ID, "Buy milk", "Semi-skimmed")

			r := s.do(http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", task.ID), body)

			require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
			env := decode[updatedTaskJSON](t, r)
			assert.Equal(t, "Task updated", env.Message)
			assert.Equal(t, updatedTaskJSON{ID: task.ID, Title: "Buy milk", Description: "Semi-skimmed"}, env.Data)
			assert.Equal(t, []taskJSON{task}, s.listTasks(owner.ID))
		})
	}
}

// GIVEN Alice's task / WHEN PUT carries another id and Bob's user_id in the body
// THEN 400 INVALID_BODY (unknown fields), and neither Alice's nor Bob's task changes
func TestUpdateTask_BodyCannotRetargetOrReassign(t *testing.T) {
	s := newTestServer(t)
	alice := s.createUser("Alice", "alice@example.com")
	bob := s.createUser("Bob", "bob@example.com")
	aliceTask := s.createTask(alice.ID, "Alice task", "")
	bobTask := s.createTask(bob.ID, "Bob task", "")
	body, err := json.Marshal(map[string]any{"id": bobTask.ID, "user_id": bob.ID, "completed": true})
	require.NoError(t, err)

	r := s.do(http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", aliceTask.ID), string(body))

	requireError(t, r, http.StatusBadRequest, "INVALID_BODY", shared.Message(shared.LangEnglish, shared.MsgUnknownField))
	assert.Equal(t, []taskJSON{aliceTask}, s.listTasks(alice.ID), "Alice's task must be untouched")
	assert.Equal(t, []taskJSON{bobTask}, s.listTasks(bob.ID), "Bob's task must be untouched")
}

// GIVEN boundary task ids / WHEN PUT / THEN 404 for valid unknown ids, 400 outside uint32
func TestUpdateTask_BoundaryIDs(t *testing.T) {
	s := newTestServer(t)

	for id, want := range map[string]struct {
		status  int
		code    string
		message string
	}{
		"0":          {http.StatusNotFound, "TASK_NOT_FOUND", "Task not found"},
		"4294967295": {http.StatusNotFound, "TASK_NOT_FOUND", "Task not found"},
		"4294967296": {http.StatusBadRequest, "INVALID_ID", "Invalid task ID"},
		"-1":         {http.StatusBadRequest, "INVALID_ID", "Invalid task ID"},
	} {
		requireError(t, s.do(http.MethodPut, "/api/v1/tasks/"+id, `{"completed":true}`), want.status, want.code, want.message)
	}
	assert.Zero(t, s.countRows(&pgstore.TaskModel{}), "an update must never create a task")
}

// GIVEN 2 users and 3 tasks / WHEN GET /api/v1/stats / THEN the exact counters
func TestStats_ReturnsExactCounters(t *testing.T) {
	s := newTestServer(t)
	alice := s.createUser("Alice", "alice@example.com")
	bob := s.createUser("Bob", "bob@example.com")
	s.createTask(alice.ID, "Task 1", "")
	s.createTask(alice.ID, "Task 2", "")
	s.createTask(bob.ID, "Task 3", "")

	r := s.do(http.MethodGet, "/api/v1/stats", "")

	require.Equal(t, http.StatusOK, r.code)
	env := decode[statsJSON](t, r)
	assert.True(t, env.Success)
	assert.Equal(t, statsJSON{Users: 2, Tasks: 3}, env.Data)
}

// GIVEN an empty database / WHEN GET /api/v1/stats / THEN both counters are 0 (and present)
func TestStats_EmptyDatabase(t *testing.T) {
	s := newTestServer(t)

	r := s.do(http.MethodGet, "/api/v1/stats", "")

	require.Equal(t, http.StatusOK, r.code)
	assert.JSONEq(t, `{"success":true,"data":{"users":0,"tasks":0}}`, string(r.body))
}
