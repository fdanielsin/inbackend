package shared

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/stretchr/testify/assert"
)

// Each domain error must map to its status, stable error_code and public message,
// also when wrapped (%w) or joined (errors.Join) by an outer layer: the controllers answer from Classify.
func TestErrorClassification_EveryDomainError(t *testing.T) {
	cases := []struct {
		err     error
		status  int
		code    string
		message string
	}{
		{user.ErrUserNameEmpty, http.StatusBadRequest, "VALIDATION_FAILED", "User name cannot be empty"},
		{user.ErrUserEmailEmpty, http.StatusBadRequest, "VALIDATION_FAILED", "User email cannot be empty"},
		{user.ErrUserEmailInvalid, http.StatusBadRequest, "VALIDATION_FAILED", "User email is invalid"},
		{user.ErrUserEmailTaken, http.StatusConflict, "EMAIL_TAKEN", "Email already in use"},
		{user.ErrUserNotFound, http.StatusNotFound, "USER_NOT_FOUND", "User not found"},
		{task.ErrTaskTitleEmpty, http.StatusBadRequest, "VALIDATION_FAILED", "Task title cannot be empty"},
		{task.ErrTaskUserIDInvalid, http.StatusBadRequest, "VALIDATION_FAILED", "Task user_id cannot be empty"},
		{task.ErrTaskNotFound, http.StatusNotFound, "TASK_NOT_FOUND", "Task not found"},
	}
	for _, tc := range cases {
		for name, err := range map[string]error{
			"direct":  tc.err,
			"wrapped": fmt.Errorf("repository: %w", tc.err),
			"joined":  errors.Join(errors.New("repository context"), tc.err),
		} {
			t.Run(tc.err.Error()+"/"+name, func(t *testing.T) {
				assert.Equal(t, shared.APIError{Status: tc.status, Code: tc.code, Message: tc.message}, shared.Classify(err))
			})
		}
	}
}

// GIVEN: A technical error (e.g. the database is down) / THEN: neither 400 nor 404, generic message
func TestErrorClassification_TechnicalErrorIsInternal(t *testing.T) {
	err := errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")

	assert.Equal(t, shared.APIError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "Internal server error"}, shared.Classify(err))
}
