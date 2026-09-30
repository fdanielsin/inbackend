package http

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	"github.com/fdanielsin/inbackend/internal/infrastructure/logger"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v4"
)

// TaskController gère les routes Task.
type TaskController struct {
	createTaskHandler    *commands.CreateTaskHandler
	updateTaskHandler    *commands.UpdateTaskHandler
	deleteTaskHandler    *commands.DeleteTaskHandler
	listUserTasksHandler *queries.ListUserTasksHandler
	log                  *slog.Logger
}

// NewTaskController crée une nouvelle instance du controller.
func NewTaskController(
	createTaskHandler *commands.CreateTaskHandler,
	updateTaskHandler *commands.UpdateTaskHandler,
	deleteTaskHandler *commands.DeleteTaskHandler,
	listUserTasksHandler *queries.ListUserTasksHandler,
) *TaskController {
	return &TaskController{
		createTaskHandler:    createTaskHandler,
		updateTaskHandler:    updateTaskHandler,
		deleteTaskHandler:    deleteTaskHandler,
		listUserTasksHandler: listUserTasksHandler,
		log:                  logger.GetSlog(),
	}
}

// CreateTask gère POST /api/v1/tasks.
func (tc *TaskController) CreateTask(c echo.Context) error {
	var req commands.CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return respondInvalidBody(c, tc.log)
	}

	tc.log.Info("Creating task", slog.Uint64("userID", uint64(req.UserID)))

	resp, err := tc.createTaskHandler.Execute(c.Request().Context(), req)
	if err != nil {
		return respondWithError(c, tc.log, "Error creating task", err, slog.Uint64("userID", uint64(req.UserID)))
	}

	return c.JSON(http.StatusCreated, shared.SuccessResponse(resp, "Task created"))
}

// UpdateTask gère PUT /api/v1/tasks/:id.
func (tc *TaskController) UpdateTask(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return respondInvalidID(c, tc.log, "task", idStr)
	}

	var req commands.UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		return respondInvalidBody(c, tc.log)
	}
	req.ID = uint(id)

	tc.log.Info("Updating task", slog.Uint64("id", id))

	resp, err := tc.updateTaskHandler.Execute(c.Request().Context(), req)
	if err != nil {
		return respondWithError(c, tc.log, "Error updating task", err, slog.Uint64("id", id))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, "Task updated"))
}

// DeleteTask gère DELETE /api/v1/tasks/:id.
func (tc *TaskController) DeleteTask(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return respondInvalidID(c, tc.log, "task", idStr)
	}

	tc.log.Info("Deleting task", slog.Uint64("id", id))

	err = tc.deleteTaskHandler.Execute(c.Request().Context(), commands.DeleteTaskRequest{ID: uint(id)})
	if err != nil {
		return respondWithError(c, tc.log, "Error deleting task", err, slog.Uint64("id", id))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponseEmpty("Task deleted"))
}

// ListUserTasks gère GET /api/v1/users/:id/tasks.
func (tc *TaskController) ListUserTasks(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return respondInvalidID(c, tc.log, "user", idStr)
	}

	tc.log.Info("Listing user tasks", slog.Uint64("userID", id))

	resp, err := tc.listUserTasksHandler.Execute(c.Request().Context(), queries.ListUserTasksRequest{UserID: uint(id)})
	if err != nil {
		return respondWithError(c, tc.log, "Error listing user tasks", err, slog.Uint64("userID", id))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, ""))
}
