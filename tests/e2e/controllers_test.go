package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	usercommands "github.com/fdanielsin/inbackend/internal/application/user/commands"
	userqueries "github.com/fdanielsin/inbackend/internal/application/user/queries"
	ctrlhttp "github.com/fdanielsin/inbackend/internal/controllers/http"
	"github.com/fdanielsin/inbackend/internal/infrastructure/database"
	pgimpl "github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/fdanielsin/inbackend/internal/models"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupTestDB initializes a test database using Podman/Docker (testcontainers).
// Returns GORM connection and TestDB handle for cleanup.
func setupTestDB(t *testing.T) (*gorm.DB, *database.TestDB) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testDB, err := database.SetupTestDB(ctx)
	require.NoError(t, err, "failed to setup test container")

	// Open GORM connection to testcontainer PostgreSQL
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: testDB.DSN,
	}), &gorm.Config{})
	require.NoError(t, err, "failed to connect to test database")

	// Auto-migrate models
	err = db.AutoMigrate(&models.User{}, &models.Task{})
	require.NoError(t, err, "failed to migrate test database")

	return db, testDB
}

// cleanupTestDB stops the test container.
func cleanupTestDB(t *testing.T, testDB *database.TestDB) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := testDB.Stop(ctx); err != nil {
		t.Logf("warning: failed to stop test container: %v", err)
	}
}

// setupControllers initializes HTTP controllers with repositories.
func setupControllers(db *gorm.DB) (*ctrlhttp.UserController, *ctrlhttp.TaskController, *ctrlhttp.StatsController, *ctrlhttp.HealthController) {
	userRepo := pgimpl.NewUserRepository(db)
	taskRepo := pgimpl.NewTaskRepository(db)

	userCtrl := ctrlhttp.NewUserController(
		usercommands.NewCreateUserHandler(userRepo),
		userqueries.NewGetUserHandler(userRepo),
		userqueries.NewListUsersHandler(userRepo),
	)

	taskCtrl := ctrlhttp.NewTaskController(
		commands.NewCreateTaskHandler(taskRepo),
		commands.NewUpdateTaskHandler(taskRepo),
		commands.NewDeleteTaskHandler(taskRepo),
		queries.NewListUserTasksHandler(taskRepo),
	)

	statsCtrl := ctrlhttp.NewStatsController(queries.NewGetStatsHandler(userRepo, taskRepo))
	healthCtrl := ctrlhttp.NewHealthController()

	return userCtrl, taskCtrl, statsCtrl, healthCtrl
}

// Test suite : User Controller
func TestUserController_CreateUser(t *testing.T) {
	db, testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	userCtrl, _, _, _ := setupControllers(db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader([]byte(`{"name": "John", "email": "john@example.com"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := userCtrl.CreateUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp shared.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "User created", resp.Message)
}

func TestUserController_CreateUser_InvalidPayload(t *testing.T) {
	db, testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	userCtrl, _, _, _ := setupControllers(db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader([]byte(`{"name": "", "email": "john@example.com"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := userCtrl.CreateUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp shared.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestUserController_GetUser(t *testing.T) {
	db, testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	userCtrl, _, _, _ := setupControllers(db)

	// Create user first
	userRepo := pgimpl.NewUserRepository(db)
	userHandler := usercommands.NewCreateUserHandler(userRepo)
	resp, _ := userHandler.Execute(context.Background(), usercommands.CreateUserRequest{
		Name:  "John",
		Email: "john@example.com",
	})

	// Get user
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := userCtrl.GetUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var apiResp shared.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &apiResp)
	require.NoError(t, err)
	assert.True(t, apiResp.Success)
	assert.NotNil(t, resp)
}

func TestUserController_ListUsers(t *testing.T) {
	db, testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	userCtrl, _, _, _ := setupControllers(db)

	// Create users
	userRepo := pgimpl.NewUserRepository(db)
	userHandler := usercommands.NewCreateUserHandler(userRepo)
	userHandler.Execute(context.Background(), usercommands.CreateUserRequest{Name: "John", Email: "john@example.com"})
	userHandler.Execute(context.Background(), usercommands.CreateUserRequest{Name: "Jane", Email: "jane@example.com"})

	// List users
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := userCtrl.ListUsers(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var apiResp shared.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &apiResp)
	require.NoError(t, err)
	assert.True(t, apiResp.Success)
}

// Test suite : Task Controller
func TestTaskController_CreateTask(t *testing.T) {
	db, testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	_, taskCtrl, _, _ := setupControllers(db)

	// Create user first
	userRepo := pgimpl.NewUserRepository(db)
	userHandler := usercommands.NewCreateUserHandler(userRepo)
	userResp, _ := userHandler.Execute(context.Background(), usercommands.CreateUserRequest{
		Name:  "John",
		Email: "john@example.com",
	})

	// Create task
	taskPayload := commands.CreateTaskRequest{
		UserID:      userResp.ID,
		Title:       "Buy milk",
		Description: "From store",
		Completed:   false,
	}
	payload, _ := json.Marshal(taskPayload)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := taskCtrl.CreateTask(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp shared.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestTaskController_UpdateTask(t *testing.T) {
	db, testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	_, taskCtrl, _, _ := setupControllers(db)

	// Setup: Create user and task
	userRepo := pgimpl.NewUserRepository(db)
	userHandler := usercommands.NewCreateUserHandler(userRepo)
	userResp, _ := userHandler.Execute(context.Background(), usercommands.CreateUserRequest{
		Name:  "John",
		Email: "john@example.com",
	})

	taskRepo := pgimpl.NewTaskRepository(db)
	taskHandler := commands.NewCreateTaskHandler(taskRepo)
	taskResp, _ := taskHandler.Execute(context.Background(), commands.CreateTaskRequest{
		UserID:      userResp.ID,
		Title:       "Old title",
		Description: "Old desc",
		Completed:   false,
	})

	// Update task
	updatePayload := commands.UpdateTaskRequest{
		ID:          taskResp.ID,
		Title:       "New title",
		Description: "New desc",
		Completed:   true,
	}
	payload, _ := json.Marshal(updatePayload)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/1", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := taskCtrl.UpdateTask(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp shared.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

// Test suite : Stats Controller
func TestStatsController_GetStats(t *testing.T) {
	db, testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	_, _, statsCtrl, _ := setupControllers(db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := statsCtrl.GetStats(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp shared.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

// Test suite : Health Controller
func TestHealthController_Health(t *testing.T) {
	db, testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	_, _, _, healthCtrl := setupControllers(db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := healthCtrl.HealthCheck(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp shared.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

// Test suite : Task Controller - Additional tests
func TestTaskController_DeleteTask(t *testing.T) {
	db, testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	_, taskCtrl, _, _ := setupControllers(db)

	// Setup: Create user and task
	userRepo := pgimpl.NewUserRepository(db)
	userHandler := usercommands.NewCreateUserHandler(userRepo)
	userResp, _ := userHandler.Execute(context.Background(), usercommands.CreateUserRequest{
		Name:  "John",
		Email: "john@example.com",
	})

	taskRepo := pgimpl.NewTaskRepository(db)
	taskHandler := commands.NewCreateTaskHandler(taskRepo)
	taskResp, _ := taskHandler.Execute(context.Background(), commands.CreateTaskRequest{
		UserID:      userResp.ID,
		Title:       "Task to delete",
		Description: "Will be deleted",
		Completed:   false,
	})

	// Delete task
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", taskResp.ID))

	err := taskCtrl.DeleteTask(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp shared.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestTaskController_ListUserTasks(t *testing.T) {
	db, testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	_, taskCtrl, _, _ := setupControllers(db)

	// Setup: Create user and tasks
	userRepo := pgimpl.NewUserRepository(db)
	userHandler := usercommands.NewCreateUserHandler(userRepo)
	userResp, _ := userHandler.Execute(context.Background(), usercommands.CreateUserRequest{
		Name:  "John",
		Email: "john@example.com",
	})

	taskRepo := pgimpl.NewTaskRepository(db)
	taskHandler := commands.NewCreateTaskHandler(taskRepo)
	taskHandler.Execute(context.Background(), commands.CreateTaskRequest{
		UserID:      userResp.ID,
		Title:       "Task 1",
		Description: "First task",
		Completed:   false,
	})
	taskHandler.Execute(context.Background(), commands.CreateTaskRequest{
		UserID:      userResp.ID,
		Title:       "Task 2",
		Description: "Second task",
		Completed:   true,
	})

	// List user tasks
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1/tasks", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := taskCtrl.ListUserTasks(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp shared.APIResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}
