package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdanielsin/inbackend/internal/handlers"
	"github.com/fdanielsin/inbackend/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2EUserWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	database, testDB := setupTestDB(t, ctx)
	defer testDB.Stop(ctx)

	// Clean up existing data
	database.Exec("DELETE FROM tasks")
	database.Exec("DELETE FROM users")

	h := &handlers.Handler{DB: database}
	e := echo.New()

	// 1. Create user
	userPayload := models.User{Name: "E2E Test User", Email: "e2e@test.com"}
	payload, _ := json.Marshal(userPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var userResponse models.APIResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &userResponse)
	assert.True(t, userResponse.Success)

	// 2. Get all users
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = h.GetUsers(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// 3. Get specific user
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/:id", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err = h.GetUserByID(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// 4. Create task for user
	taskPayload := models.Task{
		Title:       "E2E Task",
		Description: "E2E Test Description",
		UserID:      1,
		Completed:   false,
	}
	taskData, _ := json.Marshal(taskPayload)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(taskData))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = h.CreateTask(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// 5. Get user's tasks
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/:id/tasks", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err = h.GetUserTasks(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var tasksResponse models.APIResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &tasksResponse)
	assert.True(t, tasksResponse.Success)
}
