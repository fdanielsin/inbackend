package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type storedUser struct {
	Name  string
	Email string
}

func (s *testServer) storedUser(id uint) storedUser {
	s.t.Helper()
	var u storedUser
	require.NoError(s.t, s.db.Raw("SELECT name, email FROM user_models WHERE id = ?", id).Scan(&u).Error)
	return u
}

func nameBody(name string) string { return fmt.Sprintf(`{"name":%q}`, name) }

// GIVEN a user with tasks / WHEN its name is changed with spaces around / THEN the answer and the next reading give the trimmed name,
// the same e-mail, the same id and the same tasks (USER-R17)
func TestUpdateUser_RenamesOnly(t *testing.T) {
	s := newTestServer(t)
	user := s.createUser("Ada", "ada@example.com")
	s.createTask(user.ID, "Keep", "me")
	tasks := s.listTasks(user.ID)

	r := s.do(http.MethodPut, userPath(user.ID), `{"name":"  Grace  "}`)

	t.Run("USER-R17: the answer carries the name as stored (happy path)", func(t *testing.T) {
		require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
		env := decode[userJSON](t, r)
		assert.Equal(t, "User updated", env.Message)
		assert.Equal(t, userJSON{ID: user.ID, Name: "Grace", Email: "ada@example.com"}, env.Data)
	})
	t.Run("USER-R17: the next reading, SQL and the tasks agree (happy path)", func(t *testing.T) {
		assert.Equal(t, userJSON{ID: user.ID, Name: "Grace", Email: "ada@example.com"}, decode[userJSON](t, s.in(t).do(http.MethodGet, userPath(user.ID), "")).Data)
		assert.Equal(t, storedUser{Name: "Grace", Email: "ada@example.com"}, s.storedUser(user.ID))
		assert.Equal(t, tasks, s.listTasks(user.ID))
	})
}

// GIVEN names around the limits / WHEN sent / THEN accepted or refused as at creation, and a refusal leaves the stored name (USER-R17)
func TestUpdateUser_NameLimits(t *testing.T) {
	s := newTestServer(t)
	user := s.createUser("Ada", "ada@example.com")

	for name, body := range map[string]string{
		"an empty object":             `{}`,
		"a null name":                 `{"name":null}`,
		"an empty name":               `{"name":""}`,
		"a name of spaces":            nameBody("   "),
		"101 characters":              nameBody(strings.Repeat("a", 101)),
		"100 characters and 2 spaces": nameBody(" " + strings.Repeat("a", 100) + " "),
		"an empty body":               ``,
	} {
		t.Run("USER-R17: "+name+" is refused with VALIDATION_FAILED and the name stays (sad path)", func(t *testing.T) {
			r := s.in(t).do(http.MethodPut, userPath(user.ID), body)

			require.Equal(t, http.StatusBadRequest, r.code, "body: %s", r.body)
			assert.Equal(t, "VALIDATION_FAILED", decode[any](t, r).ErrorCode)
			assert.Equal(t, "Ada", s.storedUser(user.ID).Name)
		})
	}
	for name, want := range map[string]string{
		"1 character":    "G",
		"100 characters": strings.Repeat("a", 100),
		"100 emojis":     strings.Repeat("😀", 100),
	} {
		t.Run("USER-R17: "+name+" is accepted (boundary)", func(t *testing.T) {
			r := s.in(t).do(http.MethodPut, userPath(user.ID), nameBody(want))

			require.Equal(t, http.StatusOK, r.code, "body: %s", r.body)
			assert.Equal(t, want, s.storedUser(user.ID).Name)
		})
	}
}

// GIVEN a body with anything else than name / WHEN sent / THEN 400 INVALID_BODY before any database access:
// the name and the e-mail stay as they were (USER-R18)
func TestUpdateUser_RefusesEverythingButName(t *testing.T) {
	s := newTestServer(t)
	user := s.createUser("Ada", "ada@example.com")
	want := storedUser{Name: "Ada", Email: "ada@example.com"}

	for name, body := range map[string]string{
		"email alone":                  `{"email":"new@example.com"}`,
		"email with its current value": `{"email":"ada@example.com"}`,
		"name then email":              `{"name":"Grace","email":"new@example.com"}`,
		"email then name":              `{"email":"new@example.com","name":"Grace"}`,
		"Email":                        `{"name":"Grace","Email":"new@example.com"}`,
		"EMAIL":                        `{"name":"Grace","EMAIL":"new@example.com"}`,
		"id":                           `{"name":"Grace","id":7}`,
		"an unknown property":          `{"name":"Grace","nickname":"G"}`,
		"Name by case":                 `{"Name":"Grace"}`,
		"a name of the wrong type":     `{"name":5}`,
		"an array":                     `[]`,
	} {
		t.Run("USER-R18: "+name+" is refused with INVALID_BODY (sad path)", func(t *testing.T) {
			r := s.in(t).do(http.MethodPut, userPath(user.ID), body)

			require.Equal(t, http.StatusBadRequest, r.code, "body: %s", r.body)
			assert.Equal(t, "INVALID_BODY", decode[any](t, r).ErrorCode)
			assert.Equal(t, want, s.storedUser(user.ID))
		})
	}
	t.Run("USER-R18: without a JSON content type the body is refused (sad path)", func(t *testing.T) {
		r := s.sendRaw(http.MethodPut, userPath(user.ID), "text/plain", `{"name":"Grace"}`)

		requireError(t, r, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		assert.Equal(t, want, s.storedUser(user.ID))
	})
}

// GIVEN an invalid id and a body / THEN the body is checked first (USER-R17)
func TestUpdateUser_BodyBeforeID(t *testing.T) {
	s := newTestServer(t)
	s.createUser("Ada", "ada@example.com")

	t.Run("USER-R17: an e-mail in the body wins over an invalid id (sad path)", func(t *testing.T) {
		requireError(t, s.do(http.MethodPut, "/api/v1/users/abc", `{"email":"x@y.io"}`), http.StatusBadRequest, "INVALID_BODY", "Request body contains an unknown field")
	})
	t.Run("USER-R17: an e-mail in the body wins over an absent user (sad path)", func(t *testing.T) {
		requireError(t, s.do(http.MethodPut, "/api/v1/users/999", `{"email":"x@y.io"}`), http.StatusBadRequest, "INVALID_BODY", "Request body contains an unknown field")
	})
	t.Run("USER-R17: a malformed body wins over an invalid id (sad path)", func(t *testing.T) {
		requireError(t, s.doRaw2(http.MethodPut, "/api/v1/users/abc", `{"name":`), http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	})
	t.Run("USER-R17: a valid body on an invalid id answers INVALID_ID, also for {} (sad path)", func(t *testing.T) {
		requireError(t, s.do(http.MethodPut, "/api/v1/users/abc", nameBody("Grace")), http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		requireError(t, s.do(http.MethodPut, "/api/v1/users/abc", `{}`), http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
	})
}

func (s *testServer) doRaw2(method, path, body string) response {
	s.t.Helper()
	return s.sendRaw(method, path, "application/json", body)
}

// GIVEN a deleted or unknown user / WHEN renamed / THEN 404 USER_NOT_FOUND and no row is created (USER-R17)
func TestUpdateUser_AbsentUserIsNeverRecreated(t *testing.T) {
	s := newTestServer(t)
	gone := s.createUser("Gone", "gone@example.com")
	require.Equal(t, http.StatusOK, s.do(http.MethodDelete, userPath(gone.ID), "").code)

	for name, id := range map[string]uint{"deleted": gone.ID, "unknown": 999} {
		t.Run("USER-R17: a "+name+" user is not found and not recreated (sad path)", func(t *testing.T) {
			requireError(t, s.in(t).do(http.MethodPut, userPath(id), nameBody("Back")), http.StatusNotFound, "USER_NOT_FOUND", "User not found")

			assert.Zero(t, s.countWhere("user_models", "true"))
		})
	}
}
