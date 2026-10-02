package e2e

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/app"
	"github.com/fdanielsin/inbackend/internal/controllers/httpapi"
	"github.com/fdanielsin/inbackend/tests/testsupport"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// pg is the PostgreSQL container shared by every E2E test: the tests run against
// the production router (app.NewRouter) and the production database engine.
var pg *testsupport.Postgres

// contract validates every response sent through testServer.do (QP-02).
var contract *testsupport.Contract

func TestMain(m *testing.M) {
	var err error
	contract, err = testsupport.LoadContract()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	pg = testsupport.StartPostgres()
	code := m.Run()
	pg.Terminate()
	if code == 0 {
		code = checkContractCoverage()
	}
	os.Exit(code)
}

// checkContractCoverage fails a complete run in which a response declared by the
// contract was never produced: either a test is missing or the contract is wrong.
// A filtered run (-run) only exercises part of the contract and is not checked.
func checkContractCoverage() int {
	if f := flag.Lookup("test.run"); f != nil && f.Value.String() != "" {
		return 0
	}
	if !pg.Available() {
		return 0
	}
	missing := contract.MissingPairs()
	if len(missing) == 0 {
		return 0
	}
	fmt.Fprintf(os.Stderr, "contract coverage: %d declared responses are never produced by an E2E test:\n  %s\n",
		len(missing), strings.Join(missing, "\n  "))
	return 1
}

type testServer struct {
	t  *testing.T
	e  *echo.Echo
	db *gorm.DB
}

// newTestServer builds the router as main.go does, request bodies validated against the contract.
func newTestServer(t *testing.T, opts ...app.Option) *testServer {
	t.Helper()
	validator, err := httpapi.NewRequestValidator(contractFile(t), app.ValidatedOperations...)
	require.NoError(t, err)
	db := pg.NewDB(t)
	return &testServer{t: t, e: app.NewRouter(db, append([]app.Option{app.WithRequestValidator(validator)}, opts...)...), db: db}
}

type response struct {
	code   int
	body   []byte
	header http.Header
}

// do sends a request through the full router and fails the test if the response
// does not match openapi.yaml; a non-empty body is sent as JSON.
func (s *testServer) do(method, path, body string) response {
	s.t.Helper()
	req, r := s.send(method, path, body)
	contract.ValidateResponse(s.t, req, r.code, r.header, r.body)
	return r
}

// doOutsideOperations sends a request that matches no operation of the contract
// (unknown route, unsupported method) and checks its body against ErrorResponse.
func (s *testServer) doOutsideOperations(method, path string) response {
	s.t.Helper()
	_, r := s.send(method, path, "")
	contract.ValidateErrorEnvelope(s.t, r.body)
	return r
}

// doRaw is do without contract validation, for requests outside the contract
// (unknown route, unsupported method, CORS preflight, test-only routes).
func (s *testServer) doRaw(method, path, body string) response {
	s.t.Helper()
	_, r := s.send(method, path, body)
	return r
}

func (s *testServer) send(method, path, body string) (*http.Request, response) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return req, response{code: rec.Code, body: rec.Body.Bytes(), header: rec.Header()}
}

// closeDatabase simulates a database outage for the requests that follow.
func (s *testServer) closeDatabase() {
	s.t.Helper()
	sqlDB, err := s.db.DB()
	require.NoError(s.t, err)
	require.NoError(s.t, sqlDB.Close())
}

type envelope[T any] struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Data      T      `json:"data"`
	Error     string `json:"error"`
	ErrorCode string `json:"error_code"`
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

// requireError checks an error response: status, success=false, the stable
// error_code and the exact public message.
func requireError(t *testing.T, r response, status int, code, message string) {
	t.Helper()
	require.Equal(t, status, r.code, "body: %s", r.body)
	env := decode[json.RawMessage](t, r)
	require.False(t, env.Success)
	require.Equal(t, code, env.ErrorCode)
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
