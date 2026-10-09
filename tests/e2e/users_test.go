package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
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
		code    string
		message string
	}{
		"empty name":           {`{"name":"","email":"john@example.com"}`, "VALIDATION_FAILED", "User name cannot be empty"},
		"empty email":          {`{"name":"John","email":""}`, "VALIDATION_FAILED", "User email cannot be empty"},
		"email of spaces only": {`{"name":"John","email":"   "}`, "VALIDATION_FAILED", "User email cannot be empty"},
		"email without @":      {`{"name":"John","email":"john.example.com"}`, "VALIDATION_FAILED", "User email is invalid"},
		"email with two @":     {`{"name":"John","email":"john@@example.com"}`, "VALIDATION_FAILED", "User email is invalid"},
		"email without local":  {`{"name":"John","email":"@example.com"}`, "VALIDATION_FAILED", "User email is invalid"},
		"email without domain": {`{"name":"John","email":"john@"}`, "VALIDATION_FAILED", "User email is invalid"},
		"domain without a dot": {`{"name":"John","email":"john@localhost"}`, "VALIDATION_FAILED", "User email is invalid"},
		"TLD of one letter":    {`{"name":"John","email":"john@example.c"}`, "VALIDATION_FAILED", "User email is invalid"},
		"underscore in domain": {`{"name":"John","email":"john@exa_mple.com"}`, "VALIDATION_FAILED", "User email is invalid"},
		"hyphen at label edge": {`{"name":"John","email":"john@-example.com"}`, "VALIDATION_FAILED", "User email is invalid"},
		"local part of 65":     {`{"name":"John","email":"` + strings.Repeat("a", 65) + `@example.com"}`, "VALIDATION_FAILED", "User email is invalid"},
		"label of 64":          {`{"name":"John","email":"john@` + strings.Repeat("b", 64) + `.com"}`, "VALIDATION_FAILED", "User email is invalid"},
		"TLD of 25 letters":    {`{"name":"John","email":"john@example.` + strings.Repeat("c", 25) + `"}`, "VALIDATION_FAILED", "User email is invalid"},
		"empty object":         {`{}`, "VALIDATION_FAILED", "User name cannot be empty"},
		"malformed JSON":       {`{"name":"John",`, "INVALID_BODY", "Invalid request body"},
		"wrong type":           {`{"name":1,"email":"john@example.com"}`, "INVALID_BODY", "Invalid request body"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)

			r := s.do(http.MethodPost, "/api/v1/users", tc.body)

			requireError(t, r, http.StatusBadRequest, tc.code, tc.message)
			assert.Zero(t, s.countRows(&pgstore.UserModel{}), "nothing may be stored")
		})
	}
}

// GIVEN a one-character name and the shortest valid email "a@b.co" / WHEN POST / THEN 201
func TestCreateUser_AcceptsShortestValidFields(t *testing.T) {
	s := newTestServer(t)

	assert.Equal(t, userJSON{ID: 1, Name: "J", Email: "a@b.co"}, s.createUser("J", "a@b.co"))
}

// GIVEN emails at the accepted limits, an IDN and a punycode TLD / WHEN POST / THEN 201 for each
func TestCreateUser_AcceptsEmailsAtTheLimits(t *testing.T) {
	for _, email := range []string{
		strings.Repeat("a", 64) + "@example.com",
		"john@" + strings.Repeat("b", 63) + ".com",
		"john@example." + strings.Repeat("c", 24),
		"ivan@пример.рф",
		"john@example.xn--p1ai",
	} {
		t.Run(email[:min(len(email), 20)], func(t *testing.T) {
			s := newTestServer(t)

			assert.Equal(t, email, s.createUser("John", email).Email)
		})
	}
}

// GIVEN an email with surrounding spaces and capitals / WHEN POST / THEN 201 with the trimmed, lower-case email, stored so
func TestCreateUser_NormalizesEmail(t *testing.T) {
	s := newTestServer(t)

	created := s.createUser("Ada", "  Ada.Lovelace@Example.COM \t")

	assert.Equal(t, userJSON{ID: 1, Name: "Ada", Email: "ada.lovelace@example.com"}, created)
	stored := s.do(http.MethodGet, "/api/v1/users/1", "")
	require.Equal(t, http.StatusOK, stored.code)
	assert.Equal(t, created, decode[userJSON](t, stored).Data)
}

// GIVEN an existing user / WHEN POST with the same email, differently cased or padded / THEN 409 EMAIL_TAKEN and nothing stored (B-03)
func TestCreateUser_RejectsTakenEmail(t *testing.T) {
	for name, email := range map[string]string{
		"same email":           "ada@x.io",
		"other case":           "ADA@X.IO",
		"surrounded by spaces": " ada@x.io ",
	} {
		t.Run(name, func(t *testing.T) {
			s := newTestServer(t)
			s.createUser("Ada", "ada@x.io")
			body, err := json.Marshal(map[string]string{"name": "Other", "email": email})
			require.NoError(t, err)

			r := s.do(http.MethodPost, "/api/v1/users", string(body))

			requireError(t, r, http.StatusConflict, "EMAIL_TAKEN", "Email already in use")
			assert.Equal(t, int64(1), s.countRows(&pgstore.UserModel{}), "the duplicate must not be stored")
		})
	}
}

// GIVEN an existing user / WHEN POST with an email differing by one character / THEN 201 (B-03 boundary)
func TestCreateUser_EmailDifferingByOneCharacterIsFree(t *testing.T) {
	s := newTestServer(t)
	s.createUser("Ada", "ada@x.io")

	assert.Equal(t, userJSON{ID: 2, Name: "Bob", Email: "adb@x.io"}, s.createUser("Bob", "adb@x.io"))
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

// GIVEN users created as Charlie, Bob, Alice, the first one rewritten after the others (PostgreSQL then
// stores it last) / WHEN GET /api/v1/users / THEN exactly in ascending id order, as the contract promises
func TestListUsers_OrderedByIDNotByStorage(t *testing.T) {
	s := newTestServer(t)
	charlie := s.createUser("Charlie", "charlie@example.com")
	bob := s.createUser("Bob", "bob@example.com")
	alice := s.createUser("Alice", "alice@example.com")
	require.NoError(t, s.db.Exec("UPDATE user_models SET name = name WHERE id = ?", charlie.ID).Error)
	var physical []uint
	require.NoError(t, s.db.Raw("SELECT id FROM user_models").Scan(&physical).Error)
	require.Equal(t, []uint{bob.ID, alice.ID, charlie.ID}, physical, "the setup must change the physical order")

	r := s.do(http.MethodGet, "/api/v1/users", "")

	require.Equal(t, http.StatusOK, r.code)
	assert.Equal(t, []userJSON{charlie, bob, alice}, decode[[]userJSON](t, r).Data)
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
		t.Run(path, func(t *testing.T) {
			r := s.in(t).do(http.MethodGet, path, "")
			require.Equal(t, http.StatusOK, r.code, path)
			env := decode[userJSON](t, r)
			assert.True(t, env.Success, path)
			assert.Equal(t, john, env.Data, path)
		})
	}
}

// GIVEN a stored user / WHEN GET /api/v1/users/{id} with an unknown or boundary id / THEN 404 or 400
func TestGetUser_UnknownAndInvalidIDs(t *testing.T) {
	s := newTestServer(t)
	s.createUser("John", "john@example.com")

	cases := map[string]struct {
		id      string
		status  int
		code    string
		message string
	}{
		"unknown":               {"999", http.StatusNotFound, "USER_NOT_FOUND", "User not found"},
		"zero":                  {"0", http.StatusNotFound, "USER_NOT_FOUND", "User not found"},
		"max uint32":            {"4294967295", http.StatusNotFound, "USER_NOT_FOUND", "User not found"},
		"max uint32 + 1":        {"4294967296", http.StatusBadRequest, "INVALID_ID", "Invalid user ID"},
		"negative":              {"-1", http.StatusBadRequest, "INVALID_ID", "Invalid user ID"},
		"not a number":          {"abc", http.StatusBadRequest, "INVALID_ID", "Invalid user ID"},
		"decimal":               {"1.5", http.StatusBadRequest, "INVALID_ID", "Invalid user ID"},
		"explicit plus sign":    {"+1", http.StatusBadRequest, "INVALID_ID", "Invalid user ID"},
		"space-padded (escape)": {"%201", http.StatusBadRequest, "INVALID_ID", "Invalid user ID"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			requireError(t, s.do(http.MethodGet, "/api/v1/users/"+tc.id, ""), tc.status, tc.code, tc.message)
		})
	}
}
