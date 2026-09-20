package http

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	"github.com/fdanielsin/inbackend/internal/logger"
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
		tc.log.Warn("Invalid task request body", slog.String("error", err.Error()))
		return c.JSON(http.StatusBadRequest, shared.ErrorResponse("Invalid request body"))
	}

	tc.log.Info("Creating task", slog.Int64("userID", int64(req.UserID)), slog.String("title", req.Title))

	resp, err := tc.createTaskHandler.Execute(c.Request().Context(), req)
	if err != nil {
		if shared.IsValidationError(err) {
			tc.log.Warn("Validation error creating task", slog.String("error", err.Error()))
			return c.JSON(http.StatusBadRequest, shared.ErrorResponse(shared.MapErrorToMessage(err)))
		}
		tc.log.Error("Error creating task", slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, shared.ErrorResponse(err.Error()))
	}

	return c.JSON(http.StatusCreated, shared.SuccessResponse(resp, "Task created"))
}

// UpdateTask gère PUT /api/v1/tasks/:id.
func (tc *TaskController) UpdateTask(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		tc.log.Warn("Invalid task ID", slog.String("id", idStr))
		return c.JSON(http.StatusBadRequest, shared.ErrorResponse("Invalid task ID"))
	}

	var req commands.UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		tc.log.Warn("Invalid update task request body", slog.String("error", err.Error()))
		return c.JSON(http.StatusBadRequest, shared.ErrorResponse("Invalid request body"))
	}
	req.ID = uint(id)

	tc.log.Info("Updating task", slog.Int64("id", int64(id)))

	resp, err := tc.updateTaskHandler.Execute(c.Request().Context(), req)
	if err != nil {
		if shared.IsNotFoundError(err) {
			tc.log.Info("Task not found", slog.Int64("id", int64(id)))
			return c.JSON(http.StatusNotFound, shared.ErrorResponse("Task not found"))
		}
		if shared.IsValidationError(err) {
			tc.log.Warn("Validation error updating task", slog.String("error", err.Error()))
			return c.JSON(http.StatusBadRequest, shared.ErrorResponse(shared.MapErrorToMessage(err)))
		}
		tc.log.Error("Error updating task", slog.Int64("id", int64(id)), slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, shared.ErrorResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, "Task updated"))
}

// DeleteTask gère DELETE /api/v1/tasks/:id.
func (tc *TaskController) DeleteTask(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		tc.log.Warn("Invalid task ID", slog.String("id", idStr))
		return c.JSON(http.StatusBadRequest, shared.ErrorResponse("Invalid task ID"))
	}

	tc.log.Info("Deleting task", slog.Int64("id", int64(id)))

	err = tc.deleteTaskHandler.Execute(c.Request().Context(), commands.DeleteTaskRequest{ID: uint(id)})
	if err != nil {
		tc.log.Error("Error deleting task", slog.Int64("id", int64(id)), slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, shared.ErrorResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponseEmpty("Task deleted"))
}

// ListUserTasks gère GET /api/v1/users/:id/tasks.
func (tc *TaskController) ListUserTasks(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		tc.log.Warn("Invalid user ID", slog.String("id", idStr))
		return c.JSON(http.StatusBadRequest, shared.ErrorResponse("Invalid user ID"))
	}

	tc.log.Info("Listing user tasks", slog.Int64("userID", int64(id)))

	resp, err := tc.listUserTasksHandler.Execute(c.Request().Context(), queries.ListUserTasksRequest{UserID: uint(id)})
	if err != nil {
		tc.log.Error("Error listing user tasks", slog.Int64("userID", int64(id)), slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, shared.ErrorResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, ""))
}
