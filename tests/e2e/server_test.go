package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/fdanielsin/inbackend/tests/testsupport"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// pg is the PostgreSQL container shared by every E2E test: the tests run against
// the production router (app.NewRouter) and the production database engine.
var pg *testsupport.Postgres

func TestMain(m *testing.M) {
	pg = testsupport.StartPostgres()
	code := m.Run()
	pg.Terminate()
	os.Exit(code)
}

type testServer struct {
	t  *testing.T
	e  *echo.Echo
	db *gorm.DB
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	db := pg.NewDB(t)
	return &testServer{t: t, e: app.NewRouter(db), db: db}
}

type response struct {
	code   int
	body   []byte
	header http.Header
}

// do sends a request through the full router; a non-empty body is sent as JSON.
func (s *testServer) do(method, path, body string) response {
	s.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return response{code: rec.Code, body: rec.Body.Bytes(), header: rec.Header()}
}

// closeDatabase simulates a database outage for the requests that follow.
func (s *testServer) closeDatabase() {
	s.t.Helper()
	sqlDB, err := s.db.DB()
	require.NoError(s.t, err)
	require.NoError(s.t, sqlDB.Close())
}

type envelope[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Error   string `json:"error"`
}

// decode parses the response envelope and rejects any field the test does not expect.
func decode[T any](t *testing.T, r response) envelope[T] {
	t.Helper()
	var env envelope[T]
	dec := json.NewDecoder(bytes.NewReader(r.body))
	dec.DisallowUnknownFields()
	require.NoError(t, dec.Decode(&env), "unexpected body: %s", r.body)
	return env
}

// requireError checks an error response: status, success=false and the exact public message.
func requireError(t *testing.T, r response, status int, message string) {
	t.Helper()
	require.Equal(t, status, r.code, "body: %s", r.body)
	env := decode[json.RawMessage](t, r)
	require.False(t, env.Success)
	require.Equal(t, message, env.Error)
	require.Nil(t, env.Data)
}

type userJSON struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type taskJSON struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	UserID      uint   `json:"user_id"`
	Completed   bool   `json:"completed"`
}

type statsJSON struct {
	Users int64 `json:"users"`
	Tasks int64 `json:"tasks"`
}

func (s *testServer) createUser(name, email string) userJSON {
	s.t.Helper()
	body, err := json.Marshal(map[string]string{"name": name, "email": email})
	require.NoError(s.t, err)
	r := s.do(http.MethodPost, "/api/v1/users", string(body))
	require.Equal(s.t, http.StatusCreated, r.code, "body: %s", r.body)
	return decode[userJSON](s.t, r).Data
}

func (s *testServer) createTask(userID uint, title, description string) taskJSON {
	s.t.Helper()
	body, err := json.Marshal(map[string]any{"user_id": userID, "title": title, "description": description})
	require.NoError(s.t, err)
	r := s.do(http.MethodPost, "/api/v1/tasks", string(body))
	require.Equal(s.t, http.StatusCreated, r.code, "body: %s", r.body)
	return decode[taskJSON](s.t, r).Data
}

func (s *testServer) listTasks(userID uint) []taskJSON {
	s.t.Helper()
	r := s.do(http.MethodGet, fmt.Sprintf("/api/v1/users/%d/tasks", userID), "")
	require.Equal(s.t, http.StatusOK, r.code, "body: %s", r.body)
	return decode[[]taskJSON](s.t, r).Data
}

func (s *testServer) countRows(model any) int64 {
	s.t.Helper()
	var n int64
	require.NoError(s.t, s.db.Model(model).Count(&n).Error)
	return n
}
