package handlers

import (
	"net/http"

	"github.com/fdanielsin/inbackend/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

func (h *Handler) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Service is healthy",
		Data: map[string]interface{}{
			"status":  "ok",
			"version": "1.0.0",
			"id":      uuid.New().String(),
		},
	})
}

func (h *Handler) GetUsers(c echo.Context) error {
	var users []models.User
	result := h.DB.Find(&users)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   result.Error.Error(),
		})
	}
	return c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    users,
	})
}

func (h *Handler) CreateUser(c echo.Context) error {
	user := new(models.User)
	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	result := h.DB.Create(user)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   result.Error.Error(),
		})
	}

	return c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "User created",
		Data:    user,
	})
}

func (h *Handler) GetUserByID(c echo.Context) error {
	id := c.Param("id")
	var user models.User
	result := h.DB.First(&user, id)
	if result.Error != nil {
		return c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "User not found",
		})
	}
	return c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    user,
	})
}

func (h *Handler) GetUserTasks(c echo.Context) error {
	userID := c.Param("id")
	var tasks []models.Task
	result := h.DB.Where("user_id = ?", userID).Find(&tasks)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   result.Error.Error(),
		})
	}
	return c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    tasks,
	})
}

func (h *Handler) CreateTask(c echo.Context) error {
	task := new(models.Task)
	if err := c.Bind(task); err != nil {
		return c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	result := h.DB.Create(task)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   result.Error.Error(),
		})
	}

	return c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Task created",
		Data:    task,
	})
}

func (h *Handler) UpdateTask(c echo.Context) error {
	id := c.Param("id")
	task := new(models.Task)
	if err := c.Bind(task); err != nil {
		return c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	result := h.DB.Where("id = ?", id).Save(task)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   result.Error.Error(),
		})
	}

	return c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Task updated",
		Data:    task,
	})
}

func (h *Handler) DeleteTask(c echo.Context) error {
	id := c.Param("id")
	result := h.DB.Delete(&models.Task{}, id)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   result.Error.Error(),
		})
	}
	return c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Task deleted",
	})
}

func (h *Handler) GetStats(c echo.Context) error {
	var userCount int64
	var taskCount int64

	if result := h.DB.Model(&models.User{}).Count(&userCount); result.Error != nil {
		return c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   result.Error.Error(),
		})
	}

	if result := h.DB.Model(&models.Task{}).Count(&taskCount); result.Error != nil {
		return c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   result.Error.Error(),
		})
	}

	return c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"users": userCount,
			"tasks": taskCount,
		},
	})
}
