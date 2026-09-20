package shared

import (
	"errors"
	"testing"

	"github.com/fdanielsin/inbackend/internal/domain/user"
)

func TestMapErrorToMessage_WrappedKnownError(t *testing.T) {
	err := errors.Join(errors.New("repository context"), user.ErrUserNotFound)

	if got := MapErrorToMessage(err); got != "User not found" {
		t.Fatalf("MapErrorToMessage() = %q, want %q", got, "User not found")
	}
}

func TestMapErrorToMessage_UnknownErrorIsGeneric(t *testing.T) {
	if got := MapErrorToMessage(errors.New("database password leaked")); got != "Internal server error" {
		t.Fatalf("MapErrorToMessage() = %q, want %q", got, "Internal server error")
	}
}
