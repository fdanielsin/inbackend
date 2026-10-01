package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/fdanielsin/inbackend/internal/application/user/commands"
	"github.com/fdanielsin/inbackend/internal/application/user/queries"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/labstack/echo/v5"
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
	log *slog.Logger,
	createUserHandler *commands.CreateUserHandler,
	getUserHandler *queries.GetUserHandler,
	listUsersHandler *queries.ListUsersHandler,
) *UserController {
	return &UserController{
		createUserHandler: createUserHandler,
		getUserHandler:    getUserHandler,
		listUsersHandler:  listUsersHandler,
		log:               log,
	}
}

// CreateUser gère POST /api/v1/users.
func (uc *UserController) CreateUser(c *echo.Context) error {
	log := requestLogger(c, uc.log)
	var req commands.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return respondInvalidBody(c, log, err)
	}

	log.Info("Creating user")

	resp, err := uc.createUserHandler.Execute(c.Request().Context(), req)
	if err != nil {
		return respondWithError(c, log, "Error creating user", err)
	}

	return c.JSON(http.StatusCreated, shared.SuccessResponse(resp, shared.Message(requestLanguage(c), shared.MsgUserCreated)))
}

// GetUser gère GET /api/v1/users/:id.
func (uc *UserController) GetUser(c *echo.Context) error {
	log := requestLogger(c, uc.log)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return respondInvalidID(c, log, shared.MsgInvalidUserID, idStr)
	}

	log.Info("Fetching user", slog.Uint64("id", id))

	resp, err := uc.getUserHandler.Execute(c.Request().Context(), queries.GetUserRequest{ID: uint(id)})
	if err != nil {
		return respondWithError(c, log, "Error fetching user", err, slog.Uint64("id", id))
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, ""))
}

// ListUsers gère GET /api/v1/users.
func (uc *UserController) ListUsers(c *echo.Context) error {
	log := requestLogger(c, uc.log)
	log.Info("Listing all users")

	resp, err := uc.listUsersHandler.Execute(c.Request().Context(), queries.ListUsersRequest{})
	if err != nil {
		return respondWithError(c, log, "Error listing users", err)
	}

	return c.JSON(http.StatusOK, shared.SuccessResponse(resp, ""))
}
