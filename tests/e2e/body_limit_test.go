package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fdanielsin/inbackend/internal/infrastructure/persistence/pgstore"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// maxBody is the request body limit announced by the contract: 1 MiB.
const maxBody = 1 << 20

// userBodyOfSize returns a valid create-user JSON body of exactly size bytes.
func userBodyOfSize(t *testing.T, size int) string {
	t.Helper()
	frame := `{"name":"","email":"big@x.io"}`
	require.Greater(t, size, len(frame))
	body := `{"name":"` + strings.Repeat("a", size-len(frame)) + `","email":"big@x.io"}`
	require.Len(t, body, size)
	return body
}

// sendBody posts body to path, announcing its length or sending it without one (chunked).
func (s *testServer) sendBody(method, path, body string, chunked bool) response {
	s.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if chunked {
		req.ContentLength = -1
		req.TransferEncoding = []string{"chunked"}
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	r := response{code: rec.Code, body: rec.Body.Bytes(), header: rec.Header()}
	contract.ValidateResponse(s.t, req, r.code, r.header, r.body)
	return r
}

// GIVEN a body of exactly 1 MiB, or one byte less / WHEN POST /api/v1/users, with or without Content-Length
// THEN the request reaches the handler (201)
func TestBodyLimit_UpToOneMiBIsAccepted(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		for _, size := range []int{maxBody - 1, maxBody} {
			t.Run(fmt.Sprintf("size=%d chunked=%v", size, chunked), func(t *testing.T) {
				s := newTestServer(t)

				r := s.sendBody(http.MethodPost, "/api/v1/users", userBodyOfSize(t, size), chunked)

				require.Equal(t, http.StatusCreated, r.code, "body: %.200s", r.body)
				assert.Equal(t, int64(1), s.countRows(&pgstore.UserModel{}))
			})
		}
	}
}

// GIVEN a body one byte over 1 MiB, or much bigger / WHEN POST, with or without Content-Length
// THEN 413 PAYLOAD_TOO_LARGE (never 400 INVALID_BODY) and nothing stored
func TestBodyLimit_OverOneMiBIsRefused(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		for _, size := range []int{maxBody + 1, 3_000_000} {
			t.Run(fmt.Sprintf("size=%d chunked=%v", size, chunked), func(t *testing.T) {
				s := newTestServer(t)

				r := s.sendBody(http.MethodPost, "/api/v1/users", userBodyOfSize(t, size), chunked)

				requireError(t, r, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request body too large")
				assert.Zero(t, s.countRows(&pgstore.UserModel{}))
			})
		}
	}
}

// GIVEN a body over 1 MiB / WHEN sent to the other operations with a body / THEN 413 PAYLOAD_TOO_LARGE, as the contract declares
func TestBodyLimit_AppliesToEveryOperationWithABody(t *testing.T) {
	s := newTestServer(t)
	owner := s.createUser("Owner", "owner@example.com")
	task := s.createTask(owner.ID, "Keep me", "")
	tooBig := `{"title":"` + strings.Repeat("a", maxBody) + `"}`

	for _, req := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/tasks"},
		{http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", task.ID)},
	} {
		for _, chunked := range []bool{false, true} {
			r := s.sendBody(req.method, req.path, tooBig, chunked)

			requireError(t, r, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request body too large")
		}
	}
	assert.Equal(t, []taskJSON{task}, s.listTasks(owner.ID), "nothing may change")
}
