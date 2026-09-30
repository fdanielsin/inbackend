package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	pgimpl "github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
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
		message string
	}{
		"empty title":         {`{"user_id":1,"title":""}`, "Task title cannot be empty"},
		"missing title":       {`{"user_id":1}`, "Task title cannot be empty"},
		"user id 0":           {`{"user_id":0,"title":"Buy milk"}`, "Task user_id cannot be empty"},
		"missing user id":     {`{"title":"Buy milk"}`, "Task user_id cannot be empty"},
		"negative user id":    {`{"user_id":-1,"title":"Buy milk"}`, "Invalid request body"},
		"user id as a string": {`{"user_id":"1","title":"Buy milk"}`, "Invalid request body"},
		"malformed JSON":      {`{"user_id":1,"title":`, "Invalid request body"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)
			s.createUser("Owner", "owner@example.com")

			requireError(t, s.do(http.MethodPost, "/api/v1/tasks", tc.body), http.StatusBadRequest, tc.message)
			assert.Zero(t, s.countRows(&pgimpl.TaskModel{}), "nothing may be stored")
		})
	}
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
		requireError(t, s.do(http.MethodGet, "/api/v1/users/"+id+"/tasks", ""), http.StatusBadRequest, "Invalid user ID")
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
		requireError(t, s.do(http.MethodDelete, "/api/v1/tasks/"+id, ""), http.StatusBadRequest, "Invalid task ID")
	}
	assert.Equal(t, []taskJSON{task}, s.listTasks(owner.ID))
}

type updatedTaskJSON struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// GIVEN a task / WHEN PUT with an empty object or with title null / THEN 200 and nothing changes
func TestUpdateTask_AbsentAndNullFieldsAreUnchanged(t *testing.T) {
	for name, body := range map[string]string{"empty object": `{}`, "title null": `{"title":null}`} {
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

// GIVEN Alice's task / WHEN PUT carries another id and Bob's user_id in the body / THEN the path id wins and the owner stays Alice
func TestUpdateTask_BodyCannotRetargetOrReassign(t *testing.T) {
	s := newTestServer(t)
	alice := s.createUser("Alice", "alice@example.com")
	bob := s.createUser("Bob", "bob@example.com")
	aliceTask := s.createTask(alice.ID, "Alice task", "")
	bobTask := s.createTask(bob.ID, "Bob task", "")
	body, err := json.Marshal(map[string]any{"id": bobTask.ID, "user_id": bob.ID, "completed": true})
	require.NoError(t, err)

	r := s.do(http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", aliceTask.ID), string(body))

	require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
	assert.Equal(t, updatedTaskJSON{ID: aliceTask.ID, Title: "Alice task", Completed: true}, decode[updatedTaskJSON](t, r).Data)
	assert.Equal(t, []taskJSON{{ID: aliceTask.ID, Title: "Alice task", UserID: alice.ID, Completed: true}}, s.listTasks(alice.ID))
	assert.Equal(t, []taskJSON{bobTask}, s.listTasks(bob.ID), "Bob's task must be untouched")
}

// GIVEN boundary task ids / WHEN PUT / THEN 404 for valid unknown ids, 400 outside uint32
func TestUpdateTask_BoundaryIDs(t *testing.T) {
	s := newTestServer(t)

	for id, want := range map[string]struct {
		status  int
		message string
	}{
		"0":          {http.StatusNotFound, "Task not found"},
		"4294967295": {http.StatusNotFound, "Task not found"},
		"4294967296": {http.StatusBadRequest, "Invalid task ID"},
		"-1":         {http.StatusBadRequest, "Invalid task ID"},
	} {
		requireError(t, s.do(http.MethodPut, "/api/v1/tasks/"+id, `{"completed":true}`), want.status, want.message)
	}
	assert.Zero(t, s.countRows(&pgimpl.TaskModel{}), "an update must never create a task")
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
