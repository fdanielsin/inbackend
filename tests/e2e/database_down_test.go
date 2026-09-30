package e2e

import (
	"net/http"
	"testing"
)

// GIVEN the database becomes unavailable / WHEN any endpoint that reads or writes runs
// THEN 500 with the generic message: no technical detail leaks to the client
func TestEndpoints_DatabaseDown_Return500WithoutDetails(t *testing.T) {
	requests := []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/users", ""},
		{http.MethodPost, "/api/v1/users", `{"name":"John","email":"john@example.com"}`},
		{http.MethodGet, "/api/v1/users/1", ""},
		{http.MethodGet, "/api/v1/users/1/tasks", ""},
		{http.MethodPost, "/api/v1/tasks", `{"user_id":1,"title":"Buy milk"}`},
		{http.MethodPut, "/api/v1/tasks/1", `{"completed":true}`},
		{http.MethodDelete, "/api/v1/tasks/1", ""},
		{http.MethodGet, "/api/v1/stats", ""},
	}
	for _, req := range requests {
		t.Run(req.method+" "+req.path, func(t *testing.T) {
			s := newTestServer(t)
			owner := s.createUser("Owner", "owner@example.com")
			s.createTask(owner.ID, "Buy milk", "")
			s.closeDatabase()

			requireError(t, s.do(req.method, req.path, req.body), http.StatusInternalServerError, "Internal server error")
		})
	}
}
