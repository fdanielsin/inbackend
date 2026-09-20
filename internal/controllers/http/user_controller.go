package http

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/fdanielsin/inbackend/internal/application/user/commands"
	"github.com/fdanielsin/inbackend/internal/application/user/queries"
	"github.com/fdanielsin/inbackend/internal/infrastructure/logger"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v4"
)

// UserController gère les routes User.
type UserController struct {
	createUserHandler *commands.CreateUserHandler
	getUserHandler    *queries.GetUserHandler
	listUsersHandler  *queries.ListUsersHandler
	log               *slog.Logger
}

// NewUserController crée une nouvelle instance du controller.
func NewUserController(
	createUserHandler *commands.CreateUserHandler,
	getUserHandler *queries.GetUserHandler,
	listUsersHandler *queries.ListUsersHandler,
) *UserController {
	return &UserController{
		createUserHandler: createUserHandler,
		getUserHandler:    getUserHandler,
		listUsersHandler:  listUsersHandler,
		log:               logger.GetSlog(),
	}
}

// CreateUser gère POST /api/v1/users.
func (uc *UserController) CreateUser(c echo.Context) error {
	var req commands.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		uc.log.Warn("Invalid request body", slog.String("error", err.Error()))
		return c.JSON(http.StatusBadRequest, shared.ErrorResponse("Invalid request body"))
	}

	uc.log.Info("Creating user", slog.String("email", req.Email), slog.String("name", req.Name))

	resp, err := uc.createUserHandler.Execute(c.Request().Context(), req)
	if err != nil {
		if shared.IsValidationError(err) {
			uc.log.Warn("Validation error creating user", slog.String("error", err.Error()))
			return c.JSON(http.StatusBadRequest, shared.ErrorResponse(shared.MapErrorToMessage(err)))
		}
		uc.log.Error("Error creating user", slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, shared.ErrorResponse(shared.MapErrorToMessage(err)))
	}

	return c.JSON(http.StatusCreated, shared.SuccessResponse(resp, "User created"))
}

// GetUser gère GET /api/v1/users/:id.
func (uc *UserController) GetUser(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		uc.log.Warn("Invalid user ID", slog.String("id", idStr))
		return c.JSON(http.StatusBadRequest, shared.ErrorResponse("Invalid user ID"))
	}

	uc.log.Info("Fetching user", slog.Int64("id", int64(id)))

	resp, err := uc.getUserHandler.Execute(c.Request().Context(), queries.GetUserRequest{ID: uint(id)})
	if err != nil {
		if shared.IsNotFoundError(err) {
			uc.log.Info("User not found", slog.Int64("id", int64(id)))
			return c.JSON(http.StatusNotFound, shared.ErrorResponse("User not found"))
		}
		uc.log.Error("Error fetching user", slog.Int64("id", int64(id)), slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, shared.ErrorResponse(shared.MapErrorToMessage(err)))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, ""))
}

// ListUsers gère GET /api/v1/users.
func (uc *UserController) ListUsers(c echo.Context) error {
	uc.log.Info("Listing all users")

	resp, err := uc.listUsersHandler.Execute(c.Request().Context(), queries.ListUsersRequest{})
	if err != nil {
		uc.log.Error("Error listing users", slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, shared.ErrorResponse(shared.MapErrorToMessage(err)))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, ""))
}
