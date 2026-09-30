package e2e

import (
	"fmt"
	"net/http"
	"testing"

	pgimpl "github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN a valid payload / WHEN POST /api/v1/users / THEN 201 with the created user, readable afterwards
func TestCreateUser_ReturnsAndPersistsUser(t *testing.T) {
	s := newTestServer(t)

	r := s.do(http.MethodPost, "/api/v1/users", `{"name":"John Doe","email":"john@example.com"}`)

	require.Equal(t, http.StatusCreated, r.code, "body: %s", r.body)
	env := decode[userJSON](t, r)
	assert.True(t, env.Success)
	assert.Equal(t, "User created", env.Message)
	assert.Equal(t, userJSON{ID: 1, Name: "John Doe", Email: "john@example.com"}, env.Data)

	stored := s.do(http.MethodGet, "/api/v1/users/1", "")
	require.Equal(t, http.StatusOK, stored.code)
	assert.Equal(t, env.Data, decode[userJSON](t, stored).Data)
}

// GIVEN an invalid payload / WHEN POST /api/v1/users / THEN 400 with the exact message and nothing stored
func TestCreateUser_RejectsInvalidPayloads(t *testing.T) {
	cases := map[string]struct {
		body    string
		message string
	}{
		"empty name":     {`{"name":"","email":"john@example.com"}`, "User name cannot be empty"},
		"empty email":    {`{"name":"John","email":""}`, "User email cannot be empty"},
		"empty object":   {`{}`, "User name cannot be empty"},
		"malformed JSON": {`{"name":"John",`, "Invalid request body"},
		"wrong type":     {`{"name":1,"email":"john@example.com"}`, "Invalid request body"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)

			r := s.do(http.MethodPost, "/api/v1/users", tc.body)

			requireError(t, r, http.StatusBadRequest, tc.message)
			assert.Zero(t, s.countRows(&pgimpl.UserModel{}), "nothing may be stored")
		})
	}
}

// GIVEN one-character name and email (contract minLength 1) / WHEN POST / THEN 201
func TestCreateUser_AcceptsSingleCharacterFields(t *testing.T) {
	s := newTestServer(t)

	assert.Equal(t, userJSON{ID: 1, Name: "J", Email: "j"}, s.createUser("J", "j"))
}

// GIVEN an email without @ (the contract only requires a non-empty string) / WHEN POST / THEN 201
func TestCreateUser_AcceptsEmailWithoutFormat(t *testing.T) {
	s := newTestServer(t)

	assert.Equal(t, userJSON{ID: 1, Name: "John", Email: "bad"}, s.createUser("John", "bad"))
}

// GIVEN Unicode, emoji, quotes and markup / WHEN created then read / THEN returned byte for byte
func TestCreateUser_KeepsSpecialCharacters(t *testing.T) {
	s := newTestServer(t)
	name := `Zoë O'Brien 🚀 <b>"gras"</b> ; DROP TABLE user_models;`
	email := "zoë+tâches@exemple.fr"

	created := s.createUser(name, email)

	r := s.do(http.MethodGet, fmt.Sprintf("/api/v1/users/%d", created.ID), "")
	require.Equal(t, http.StatusOK, r.code)
	assert.Equal(t, userJSON{ID: created.ID, Name: name, Email: email}, decode[userJSON](t, r).Data)
}

// GIVEN two users / WHEN GET /api/v1/users / THEN both are listed with their fields
func TestListUsers_ReturnsAllUsers(t *testing.T) {
	s := newTestServer(t)
	alice := s.createUser("Alice", "alice@example.com")
	bob := s.createUser("Bob", "bob@example.com")

	r := s.do(http.MethodGet, "/api/v1/users?unknown=1", "")

	require.Equal(t, http.StatusOK, r.code)
	env := decode[[]userJSON](t, r)
	assert.True(t, env.Success)
	assert.ElementsMatch(t, []userJSON{alice, bob}, env.Data)
}

// GIVEN exactly one user / WHEN GET /api/v1/users / THEN a one-element list
func TestListUsers_SingleUser(t *testing.T) {
	s := newTestServer(t)
	alice := s.createUser("Alice", "alice@example.com")

	r := s.do(http.MethodGet, "/api/v1/users", "")

	require.Equal(t, http.StatusOK, r.code)
	assert.Equal(t, []userJSON{alice}, decode[[]userJSON](t, r).Data)
}

// GIVEN no user / WHEN GET /api/v1/users / THEN data is an empty array, not null
func TestListUsers_EmptyIsAnArray(t *testing.T) {
	s := newTestServer(t)

	r := s.do(http.MethodGet, "/api/v1/users", "")

	require.Equal(t, http.StatusOK, r.code)
	assert.JSONEq(t, `{"success":true,"data":[]}`, string(r.body))
}

// GIVEN a stored user / WHEN GET /api/v1/users/{id} / THEN the user, also with leading zeros in the id
func TestGetUser_ReturnsUser(t *testing.T) {
	s := newTestServer(t)
	john := s.createUser("John", "john@example.com")

	for _, path := range []string{"/api/v1/users/1", "/api/v1/users/01"} {
		r := s.do(http.MethodGet, path, "")
		require.Equal(t, http.StatusOK, r.code, path)
		env := decode[userJSON](t, r)
		assert.True(t, env.Success, path)
		assert.Equal(t, john, env.Data, path)
	}
}

// GIVEN a stored user / WHEN GET /api/v1/users/{id} with an unknown or boundary id / THEN 404 or 400
func TestGetUser_UnknownAndInvalidIDs(t *testing.T) {
	s := newTestServer(t)
	s.createUser("John", "john@example.com")

	cases := map[string]struct {
		id      string
		status  int
		message string
	}{
		"unknown":               {"999", http.StatusNotFound, "User not found"},
		"zero":                  {"0", http.StatusNotFound, "User not found"},
		"max uint32":            {"4294967295", http.StatusNotFound, "User not found"},
		"max uint32 + 1":        {"4294967296", http.StatusBadRequest, "Invalid user ID"},
		"negative":              {"-1", http.StatusBadRequest, "Invalid user ID"},
		"not a number":          {"abc", http.StatusBadRequest, "Invalid user ID"},
		"decimal":               {"1.5", http.StatusBadRequest, "Invalid user ID"},
		"explicit plus sign":    {"+1", http.StatusBadRequest, "Invalid user ID"},
		"space-padded (escape)": {"%201", http.StatusBadRequest, "Invalid user ID"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			requireError(t, s.do(http.MethodGet, "/api/v1/users/"+tc.id, ""), tc.status, tc.message)
		})
	}
}
