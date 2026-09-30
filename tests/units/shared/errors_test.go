package shared

import (
	"errors"
	"fmt"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
	"github.com/fdanielsin/inbackend/internal/shared"
	"github.com/stretchr/testify/assert"
)

// Each domain error must map to its HTTP family and its public message, also when
// wrapped by an outer layer: the controllers choose 400/404/500 from these helpers.
func TestErrorClassification_EveryDomainError(t *testing.T) {
	cases := []struct {
		err          error
		isValidation bool
		isNotFound   bool
		message      string
	}{
		{user.ErrUserNameEmpty, true, false, "User name cannot be empty"},
		{user.ErrUserEmailEmpty, true, false, "User email cannot be empty"},
		{user.ErrUserNotFound, false, true, "User not found"},
		{task.ErrTaskTitleEmpty, true, false, "Task title cannot be empty"},
		{task.ErrTaskUserIDInvalid, true, false, "Task user_id cannot be empty"},
		{task.ErrTaskNotFound, false, true, "Task not found"},
	}
	for _, tc := range cases {
		for name, err := range map[string]error{
			"direct":  tc.err,
			"wrapped": fmt.Errorf("repository: %w", tc.err),
		} {
			t.Run(tc.err.Error()+"/"+name, func(t *testing.T) {
				assert.Equal(t, tc.isValidation, shared.IsValidationError(err), "IsValidationError")
				assert.Equal(t, tc.isNotFound, shared.IsNotFoundError(err), "IsNotFoundError")
				assert.Equal(t, tc.message, shared.MapErrorToMessage(err), "MapErrorToMessage")
			})
		}
	}
}

// GIVEN: A technical error (e.g. the database is down) / THEN: neither 400 nor 404, generic message
func TestErrorClassification_TechnicalErrorIsInternal(t *testing.T) {
	err := errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")

	assert.False(t, shared.IsValidationError(err))
	assert.False(t, shared.IsNotFoundError(err))
	assert.Equal(t, "Internal server error", shared.MapErrorToMessage(err))
}

// GIVEN: No error / THEN: empty message and no classification
func TestErrorClassification_NilError(t *testing.T) {
	assert.False(t, shared.IsValidationError(nil))
	assert.False(t, shared.IsNotFoundError(nil))
	assert.Equal(t, "", shared.MapErrorToMessage(nil))
}
