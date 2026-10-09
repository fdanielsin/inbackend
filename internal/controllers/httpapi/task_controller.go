package httpapi

import (
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v5"
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
	log *slog.Logger,
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
		log:                  log,
	}
}

// CreateTask gère POST /api/v1/tasks.
func (tc *TaskController) CreateTask(c *echo.Context) error {
	log := requestLogger(c, tc.log)
	var req commands.CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return respondInvalidBody(c, log, err)
	}

	log.Info("Creating task", slog.Uint64("userID", uint64(req.UserID)))

	resp, err := tc.createTaskHandler.Execute(c.Request().Context(), req)
	if err != nil {
		return respondWithError(c, log, "Error creating task", err, slog.Uint64("userID", uint64(req.UserID)))
	}

	return c.JSON(http.StatusCreated, shared.SuccessResponse(resp, shared.Message(requestLanguage(c), shared.MsgTaskCreated)))
}

// UpdateTask gère PUT /api/v1/tasks/:id.
func (tc *TaskController) UpdateTask(c *echo.Context) error {
	log := requestLogger(c, tc.log)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return respondInvalidID(c, log, shared.MsgInvalidTaskID, idStr)
	}

	var req commands.UpdateTaskRequest
	if err = c.Bind(&req); err != nil {
		return respondInvalidBody(c, log, err)
	}
	req.ID = uint(id)

	log.Info("Updating task", slog.Uint64("id", id))

	resp, err := tc.updateTaskHandler.Execute(c.Request().Context(), req)
	if err != nil {
		return respondWithError(c, log, "Error updating task", err, slog.Uint64("id", id))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, shared.Message(requestLanguage(c), shared.MsgTaskUpdated)))
}

// DeleteTask gère DELETE /api/v1/tasks/:id.
func (tc *TaskController) DeleteTask(c *echo.Context) error {
	log := requestLogger(c, tc.log)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return respondInvalidID(c, log, shared.MsgInvalidTaskID, idStr)
	}

	log.Info("Deleting task", slog.Uint64("id", id))

	err = tc.deleteTaskHandler.Execute(c.Request().Context(), commands.DeleteTaskRequest{ID: uint(id)})
	if err != nil {
		return respondWithError(c, log, "Error deleting task", err, slog.Uint64("id", id))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponseEmpty(shared.Message(requestLanguage(c), shared.MsgTaskDeleted)))
}

// ListUserTasks gère GET /api/v1/users/:id/tasks.
func (tc *TaskController) ListUserTasks(c *echo.Context) error {
	log := requestLogger(c, tc.log)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return respondInvalidID(c, log, shared.MsgInvalidUserID, idStr)
	}

	completed, valid := parseCompletedFilter(c)
	if !valid {
		return respondInvalidCompletedFilter(c, log)
	}

	log.Info("Listing user tasks", slog.Uint64("userID", id), slog.Bool("filtered", completed != nil))

	resp, err := tc.listUserTasksHandler.Execute(c.Request().Context(), queries.ListUserTasksRequest{UserID: uint(id), Completed: completed})
	if err != nil {
		return respondWithError(c, log, "Error listing user tasks", err, slog.Uint64("userID", id))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, ""))
}

// parseCompletedFilter lit le paramètre de requête « completed » dans la
// requête brute : absent, il ne filtre pas (nil) ; sinon il doit apparaître une
// seule fois, valant exactement « true » ou « false » une fois décodé (pas de
// strconv.ParseBool, qui accepterait 1, t, T, TRUE…). La requête brute est lue
// paire par paire parce que url.Query écarte en silence les paires invalides
// (« ; » brut, pourcentage mal formé) : « completed=true; » serait lu comme absent.
// Les paires dont la clé n'est pas « completed » (sensible à la casse) sont ignorées.
func parseCompletedFilter(c *echo.Context) (completed *bool, valid bool) {
	occurrences := 0
	for pair := range strings.SplitSeq(c.Request().URL.RawQuery, "&") {
		rawKey, rawValue, _ := strings.Cut(pair, "=")
		if key, err := url.QueryUnescape(rawKey); err != nil || key != "completed" {
			continue
		}
		occurrences++
		if occurrences > 1 {
			return nil, false
		}
		value, _ := url.QueryUnescape(rawValue) // une valeur mal encodée devient vide, donc refusée ci-dessous
		switch value {
		case "true":
			completed = new(true)
		case "false":
			completed = new(false)
		default:
			return nil, false
		}
	}
	return completed, true
}
