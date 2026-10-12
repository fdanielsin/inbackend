package e2e

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *testServer) stats() statsJSON {
	s.t.Helper()
	r := s.do(http.MethodGet, "/api/v1/stats", "")
	require.Equal(s.t, http.StatusOK, r.code, "body: %s", r.body)
	return decode[statsJSON](s.t, r).Data
}

// rowsOf returns the rows of the given table for a where clause, read straight from SQL.
func (s *testServer) countWhere(table, where string, args ...any) int64 {
	s.t.Helper()
	var n int64
	require.NoError(s.t, s.db.Raw("SELECT count(*) FROM "+table+" WHERE "+where, args...).Scan(&n).Error)
	return n
}

func userPath(id uint) string { return fmt.Sprintf("/api/v1/users/%d", id) }

// GIVEN a user with 3 tasks and a neighbour with 3 / WHEN the user is deleted
// THEN the envelope says so without a name or an e-mail, the user reads 404, SQL finds neither the user nor its tasks,
// the neighbour keeps title, description and state, and the counters went down (USER-R13, USER-R15)
func TestDeleteUser_RemovesTheUserAndItsTasks(t *testing.T) {
	s := newTestServer(t)
	neighbour := s.createUser("Neighbour", "neighbour@example.com")
	victim := s.createUser("Victim Name", "victim@example.com")
	for _, title := range []string{"One", "Two", "Three"} {
		s.createTask(victim.ID, title, "details of "+title)
		s.createTask(neighbour.ID, title, "details of "+title)
	}
	s.completeTask(s.listTasks(neighbour.ID)[0].ID)
	neighbourTasks := s.listTasks(neighbour.ID)
	before := s.stats()

	r := s.do(http.MethodDelete, userPath(victim.ID), "")

	t.Run("USER-R13: the answer is a success in the envelope, with no personal data (happy path)", func(t *testing.T) {
		require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
		env := decode[any](t, r)
		assert.True(t, env.Success)
		assert.Equal(t, "User and their tasks deleted", env.Message)
		assert.Nil(t, env.Data)
		assert.NotContains(t, string(r.body), "Victim")
		assert.NotContains(t, string(r.body), "victim@")
	})
	t.Run("USER-R13: SQL finds neither the user nor any of its tasks (happy path)", func(t *testing.T) {
		assert.Zero(t, s.countWhere("user_models", "id = ?", victim.ID))
		assert.Zero(t, s.countWhere("task_models", "user_id = ?", victim.ID))
	})
	t.Run("USER-R15: its reading is 404, its tasks cannot be changed and it left the counters (happy path)", func(t *testing.T) {
		requireError(t, s.in(t).do(http.MethodGet, userPath(victim.ID), ""), http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		for id := uint(1); id <= 5; id += 2 { // the victim's tasks are the odd ones: each pair is victim, neighbour
			requireError(t, s.in(t).do(http.MethodDelete, fmt.Sprintf("/api/v1/tasks/%d", id), ""), http.StatusNotFound, "TASK_NOT_FOUND", "Task not found")
		}
		after := s.stats()
		assert.Equal(t, before.Users-1, after.Users)
		assert.Equal(t, before.Tasks-3, after.Tasks)
		listed := decode[[]userJSON](t, s.in(t).do(http.MethodGet, "/api/v1/users", "")).Data
		assert.Equal(t, []userJSON{neighbour}, listed)
	})
	t.Run("USER-R13: the neighbour keeps every task as it was (limit)", func(t *testing.T) {
		assert.Equal(t, neighbourTasks, s.listTasks(neighbour.ID))
	})
}

// GIVEN a user without any task / WHEN deleted / THEN the same success
func TestDeleteUser_WithoutTasks(t *testing.T) {
	s := newTestServer(t)
	user := s.createUser("Lonely", "lonely@example.com")

	t.Run("USER-R13: a user with no task is deleted (limit)", func(t *testing.T) {
		r := s.do(http.MethodDelete, userPath(user.ID), "")

		require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
		assert.Zero(t, s.countWhere("user_models", "true"))
	})
}

// GIVEN an absent user / WHEN deleted / THEN 404 USER_NOT_FOUND and the other users stay
func TestDeleteUser_Absent(t *testing.T) {
	s := newTestServer(t)
	kept := s.createUser("Kept", "kept@example.com")
	gone := s.createUser("Gone", "gone@example.com")
	require.Equal(t, http.StatusOK, s.do(http.MethodDelete, userPath(gone.ID), "").code)

	t.Run("USER-R14: deleting twice answers 404, the second time (sad path)", func(t *testing.T) {
		requireError(t, s.do(http.MethodDelete, userPath(gone.ID), ""), http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		assert.Equal(t, int64(1), s.countWhere("user_models", "true"))
	})
	t.Run("USER-R14: an id that never existed answers 404 (sad path)", func(t *testing.T) {
		requireError(t, s.do(http.MethodDelete, "/api/v1/users/999", ""), http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		assert.Equal(t, int64(1), s.countWhere("user_models", "id = ?", kept.ID))
	})
}

// GIVEN ids that are not a user id of the contract, and a DELETE on the collection / WHEN sent
// THEN a refusal, and user 1 still exists (USER-R14, USER-R12)
func TestDeleteUser_RefusesInvalidIDs(t *testing.T) {
	s := newTestServer(t)
	user := s.createUser("Kept", "kept@example.com")
	require.Equal(t, uint(1), user.ID)

	for _, id := range []string{"4294967296", "4294967297", "-1", "+1", "1.5", "0x1", "abc", "1%20"} {
		t.Run("USER-R14: id "+id+" is refused (boundary)", func(t *testing.T) {
			r := s.in(t).do(http.MethodDelete, "/api/v1/users/"+id, "")

			switch r.code {
			case http.StatusBadRequest:
				requireError(t, r, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
			default:
				assert.NotEqual(t, http.StatusInternalServerError, r.code)
			}
			assert.Equal(t, int64(1), s.countWhere("user_models", "id = 1"), "user 1 must still exist after DELETE %s", id)
		})
	}
	t.Run("USER-R14: id 0 is not found (boundary)", func(t *testing.T) {
		requireError(t, s.in(t).do(http.MethodDelete, "/api/v1/users/0", ""), http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		assert.Equal(t, int64(1), s.countWhere("user_models", "id = 1"))
	})
	t.Run("USER-R14: DELETE on the collection is refused (sad path)", func(t *testing.T) {
		r := s.doOutsideOperations(http.MethodDelete, "/api/v1/users")

		requireError(t, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		assert.Equal(t, int64(1), s.countWhere("user_models", "id = 1"))
	})
}

// GIVEN two deletions of the same user sent together / THEN one succeeds, the other answers 404, never a 500 (USER-R14)
func TestDeleteUser_TwoAtOnce(t *testing.T) {
	t.Run("USER-R14: two simultaneous deletions give one success and one 404, never a 500 (exception)", func(t *testing.T) {
		for round := 0; round < 20; round++ {
			s := newTestServer(t)
			user := s.createUser("Twice", "twice@example.com")
			s.createTask(user.ID, "Task", "")
			codes := make([]int, 2)
			start := make(chan struct{})
			var wg sync.WaitGroup
			for i := range codes {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					codes[i] = s.doRaw(http.MethodDelete, userPath(user.ID), "").code
				}()
			}
			close(start)
			wg.Wait()

			assert.ElementsMatch(t, []int{http.StatusOK, http.StatusNotFound}, codes, "round %d", round)
			assert.Zero(t, s.countWhere("task_models", "true"))
		}
	})
}

// GIVEN a task creation and the deletion of its user sent together / THEN, in both forced orders and when raced,
// no task is left without its user (USER-R13)
func TestDeleteUser_NoOrphanTaskWhateverTheOrder(t *testing.T) {
	t.Run("USER-R13: the task is created first, then the user is deleted (limit)", func(t *testing.T) {
		s := newTestServer(t)
		user := s.createUser("Owner", "owner@example.com")
		s.createTask(user.ID, "First", "")

		require.Equal(t, http.StatusOK, s.do(http.MethodDelete, userPath(user.ID), "").code)

		assert.Zero(t, s.countWhere("task_models", "true"))
	})
	t.Run("USER-R13: the user is deleted first, then the task creation is refused (limit)", func(t *testing.T) {
		s := newTestServer(t)
		user := s.createUser("Owner", "owner@example.com")
		require.Equal(t, http.StatusOK, s.do(http.MethodDelete, userPath(user.ID), "").code)

		r := s.do(http.MethodPost, "/api/v1/tasks", fmt.Sprintf(`{"user_id":%d,"title":"Late"}`, user.ID))

		requireError(t, r, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		assert.Zero(t, s.countWhere("task_models", "true"))
	})
	t.Run("USER-R13: raced 30 times, never an orphan (exception)", func(t *testing.T) {
		for round := 0; round < 30; round++ {
			s := newTestServer(t)
			user := s.createUser("Owner", "owner@example.com")
			start := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				<-start
				s.doRaw(http.MethodPost, "/api/v1/tasks", fmt.Sprintf(`{"user_id":%d,"title":"Racing"}`, user.ID))
			}()
			go func() {
				defer wg.Done()
				<-start
				s.doRaw(http.MethodDelete, userPath(user.ID), "")
			}()
			close(start)
			wg.Wait()

			assert.Zero(t, s.countWhere("task_models t LEFT JOIN user_models u ON u.id = t.user_id", "u.id IS NULL"), "round %d", round)
		}
	})
}

// GIVEN a deleted user's e-mail / WHEN a user is created with it (spaces and capitals included)
// THEN 409 before the deletion, success after, with a new greater id and no task (USER-R16)
func TestDeleteUser_FreesTheEmailButNeverTheID(t *testing.T) {
	s := newTestServer(t)
	old := s.createUser("Old", "a@x.io")
	s.createTask(old.ID, "Old task", "")

	t.Run("USER-R16: the e-mail is taken while the user exists (sad path)", func(t *testing.T) {
		requireError(t, s.in(t).do(http.MethodPost, "/api/v1/users", `{"name":"New","email":"  A@x.io "}`), http.StatusConflict, "EMAIL_TAKEN", "Email already in use")
	})
	t.Run("USER-R16: once deleted, the e-mail serves a new user with a greater id and no task (happy path)", func(t *testing.T) {
		require.Equal(t, http.StatusOK, s.in(t).do(http.MethodDelete, userPath(old.ID), "").code)

		r := s.in(t).do(http.MethodPost, "/api/v1/users", `{"name":"New","email":"  A@x.io "}`)

		require.Equal(t, http.StatusCreated, r.code, "body: %s", r.body)
		created := decode[userJSON](t, r).Data
		assert.Greater(t, created.ID, old.ID)
		assert.Empty(t, s.listTasks(created.ID))
		assert.Zero(t, s.countWhere("task_models", "true"))
	})
}
