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

func TestCreateUser(t *testing.T) {
	ctx := context.Background()
	database, testDB := setupTestDB(t, ctx)
	defer testDB.Stop(ctx)

	h := &handlers.Handler{DB: database}

	user := models.User{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	payload, _ := json.Marshal(user)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	err := h.CreateUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response models.APIResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.True(t, response.Success)
	assert.Equal(t, "User created", response.Message)
}

func TestGetUsers(t *testing.T) {
	ctx := context.Background()
	database, testDB := setupTestDB(t, ctx)
	defer testDB.Stop(ctx)

	h := &handlers.Handler{DB: database}

	// Create test user
	user := models.User{Name: "Jane Doe", Email: "jane@example.com"}
	database.Create(&user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	err := h.GetUsers(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response models.APIResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.True(t, response.Success)
}

func TestGetUserByID(t *testing.T) {
	ctx := context.Background()
	database, testDB := setupTestDB(t, ctx)
	defer testDB.Stop(ctx)

	h := &handlers.Handler{DB: database}

	user := models.User{Name: "Bob", Email: "bob@example.com"}
	database.Create(&user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/:id", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := h.GetUserByID(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCreateTask(t *testing.T) {
	ctx := context.Background()
	database, testDB := setupTestDB(t, ctx)
	defer testDB.Stop(ctx)

	h := &handlers.Handler{DB: database}

	// Create user first
	user := models.User{Name: "Alice", Email: "alice@example.com"}
	database.Create(&user)

	task := models.Task{
		Title:       "Test Task",
		Description: "Test Description",
		UserID:      user.ID,
		Completed:   false,
	}

	payload, _ := json.Marshal(task)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	err := h.CreateTask(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}
