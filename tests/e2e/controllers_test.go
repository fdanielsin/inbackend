package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	usercommands "github.com/fdanielsin/inbackend/internal/application/user/commands"
	userqueries "github.com/fdanielsin/inbackend/internal/application/user/queries"
	ctrlhttp "github.com/fdanielsin/inbackend/internal/controllers/http"
	pgimpl "github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
	"github.com/fdanielsin/inbackend/internal/models"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for E2E tests
// GIVEN: E2E test needs database / WHEN: setupTestDB() / THEN: SQLite in-memory ready
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to setup test database")

	err = db.AutoMigrate(&models.User{}, &models.Task{})
	require.NoError(t, err, "failed to migrate test database")

	return db
}

// GIVEN: POST /api/v1/users with valid payload / WHEN: CreateUser / THEN: 201 Created
func TestUserController_CreateUser_HappyPath(t *testing.T) {
	// GIVEN: Setup
	e := echo.New()
	db := setupTestDB(t)

	userRepo := pgimpl.NewUserRepository(db)
	createUserHandler := usercommands.NewCreateUserHandler(userRepo)
	getUserHandler := userqueries.NewGetUserHandler(userRepo)
	listUsersHandler := userqueries.NewListUsersHandler(userRepo)
	userCtrl := ctrlhttp.NewUserController(createUserHandler, getUserHandler, listUsersHandler)

	e.POST("/api/v1/users", userCtrl.CreateUser)

	// WHEN: Send POST request
	body := []byte(`{"name":"John Doe","email":"john@example.com"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(rec, req)

	// THEN: Verify response
	require.Equal(t, http.StatusCreated, rec.Code)
	var resp shared.APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

// GIVEN: POST /api/v1/users with invalid payload / WHEN: CreateUser / THEN: 400 Bad Request
func TestUserController_CreateUser_SadPath_InvalidPayload(t *testing.T) {
	// GIVEN: Setup
	e := echo.New()
	db := setupTestDB(t)

	userRepo := pgimpl.NewUserRepository(db)
	createUserHandler := usercommands.NewCreateUserHandler(userRepo)
	getUserHandler := userqueries.NewGetUserHandler(userRepo)
	listUsersHandler := userqueries.NewListUsersHandler(userRepo)
	userCtrl := ctrlhttp.NewUserController(createUserHandler, getUserHandler, listUsersHandler)

	e.POST("/api/v1/users", userCtrl.CreateUser)

	// WHEN: Send POST request with empty name
	body := []byte(`{"name":"","email":"john@example.com"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(rec, req)

	// THEN: Verify error
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// GIVEN: GET /api/v1/users for multiple users / WHEN: ListUsers / THEN: 200 OK with list
func TestUserController_ListUsers_HappyPath(t *testing.T) {
	// GIVEN: Setup and create users
	e := echo.New()
	db := setupTestDB(t)

	userRepo := pgimpl.NewUserRepository(db)
	createUserHandler := usercommands.NewCreateUserHandler(userRepo)
	getUserHandler := userqueries.NewGetUserHandler(userRepo)
	listUsersHandler := userqueries.NewListUsersHandler(userRepo)
	userCtrl := ctrlhttp.NewUserController(createUserHandler, getUserHandler, listUsersHandler)

	db.Create(&models.User{Name: "Alice", Email: "alice@example.com"})
	db.Create(&models.User{Name: "Bob", Email: "bob@example.com"})

	e.GET("/api/v1/users", userCtrl.ListUsers)

	// WHEN: Send GET request
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	e.ServeHTTP(rec, req)

	// THEN: Verify response
	require.Equal(t, http.StatusOK, rec.Code)
	var resp shared.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// GIVEN: GET /api/v1/users/:id for existing user / WHEN: GetUser / THEN: 200 OK
func TestUserController_GetUser_HappyPath(t *testing.T) {
	// GIVEN: Setup and create user
	e := echo.New()
	db := setupTestDB(t)

	userRepo := pgimpl.NewUserRepository(db)
	createUserHandler := usercommands.NewCreateUserHandler(userRepo)
	getUserHandler := userqueries.NewGetUserHandler(userRepo)
	listUsersHandler := userqueries.NewListUsersHandler(userRepo)
	userCtrl := ctrlhttp.NewUserController(createUserHandler, getUserHandler, listUsersHandler)

	user := &models.User{Name: "Jane", Email: "jane@example.com"}
	db.Create(user)

	e.GET("/api/v1/users/:id", userCtrl.GetUser)

	// WHEN: Send GET request
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/1", nil)
	e.ServeHTTP(rec, req)

	// THEN: Verify response
	require.Equal(t, http.StatusOK, rec.Code)
	var resp shared.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// GIVEN: POST /api/v1/tasks with valid payload / WHEN: CreateTask / THEN: 201 Created
func TestTaskController_CreateTask_HappyPath(t *testing.T) {
	// GIVEN: Setup
	e := echo.New()
	db := setupTestDB(t)

	taskRepo := pgimpl.NewTaskRepository(db)
	createTaskHandler := commands.NewCreateTaskHandler(taskRepo)
	updateTaskHandler := commands.NewUpdateTaskHandler(taskRepo)
	deleteTaskHandler := commands.NewDeleteTaskHandler(taskRepo)
	listUserTasksHandler := queries.NewListUserTasksHandler(taskRepo)
	taskCtrl := ctrlhttp.NewTaskController(createTaskHandler, updateTaskHandler, deleteTaskHandler, listUserTasksHandler)

	e.POST("/api/v1/tasks", taskCtrl.CreateTask)

	// WHEN: Send POST request
	body := []byte(`{"user_id":1,"title":"Buy milk","description":"Get milk from store"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(rec, req)

	// THEN: Verify response
	require.Equal(t, http.StatusCreated, rec.Code)
	var resp shared.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// GIVEN: GET /api/v1/users/:id/tasks for existing user / WHEN: ListUserTasks / THEN: 200 OK
func TestTaskController_ListUserTasks_HappyPath(t *testing.T) {
	// GIVEN: Setup with user and tasks
	e := echo.New()
	db := setupTestDB(t)

	taskRepo := pgimpl.NewTaskRepository(db)
	createTaskHandler := commands.NewCreateTaskHandler(taskRepo)
	updateTaskHandler := commands.NewUpdateTaskHandler(taskRepo)
	deleteTaskHandler := commands.NewDeleteTaskHandler(taskRepo)
	listUserTasksHandler := queries.NewListUserTasksHandler(taskRepo)
	taskCtrl := ctrlhttp.NewTaskController(createTaskHandler, updateTaskHandler, deleteTaskHandler, listUserTasksHandler)

	db.Create(&models.User{ID: 1, Name: "John", Email: "john@example.com"})
	db.Create(&models.Task{ID: 1, UserID: 1, Title: "Task 1", Completed: false})
	db.Create(&models.Task{ID: 2, UserID: 1, Title: "Task 2", Completed: false})

	e.GET("/api/v1/users/:id/tasks", taskCtrl.ListUserTasks)

	// WHEN: Send GET request
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/1/tasks", nil)
	e.ServeHTTP(rec, req)

	// THEN: Verify response
	require.Equal(t, http.StatusOK, rec.Code)
	var resp shared.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// GIVEN: GET /health / WHEN: HealthCheck / THEN: 200 OK
func TestHealthController_HealthCheck_HappyPath(t *testing.T) {
	// GIVEN: Setup
	e := echo.New()
	healthCtrl := ctrlhttp.NewHealthController()
	e.GET("/health", healthCtrl.HealthCheck)

	// WHEN: Send GET request
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	e.ServeHTTP(rec, req)

	// THEN: Verify response
	require.Equal(t, http.StatusOK, rec.Code)
	var resp shared.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// GIVEN: GET /api/v1/stats with existing data / WHEN: GetStats / THEN: 200 OK
func TestStatsController_GetStats_HappyPath(t *testing.T) {
	// GIVEN: Setup with test data
	e := echo.New()
	db := setupTestDB(t)

	userRepo := pgimpl.NewUserRepository(db)
	taskRepo := pgimpl.NewTaskRepository(db)
	getStatsHandler := queries.NewGetStatsHandler(userRepo, taskRepo)
	statsCtrl := ctrlhttp.NewStatsController(getStatsHandler)

	db.Create(&models.User{Name: "Alice", Email: "alice@example.com"})
	db.Create(&models.Task{UserID: 1, Title: "Task 1", Completed: false})
	db.Create(&models.Task{UserID: 1, Title: "Task 2", Completed: true})

	e.GET("/api/v1/stats", statsCtrl.GetStats)

	// WHEN: Send GET request
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/stats", nil)
	e.ServeHTTP(rec, req)

	// THEN: Verify response
	require.Equal(t, http.StatusOK, rec.Code)
	var resp shared.APIResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}
