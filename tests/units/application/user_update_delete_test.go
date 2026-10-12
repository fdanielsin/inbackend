package application

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/application"
	usercommands "github.com/fdanielsin/inbackend/internal/application/user/commands"
	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func existingUser() *stubUserRepository {
	return &stubUserRepository{
		getFn: func(id uint) (*users.User, error) {
			return &users.User{ID: id, Name: "Ada", Email: "ada@example.com"}, nil
		},
		updateFn: func(*users.User) error { return nil },
		deleteFn: func(uint) error { return nil },
	}
}

// GIVEN a name with spaces / WHEN UpdateUser / THEN the trimmed name is stored, the email untouched, and the stored state returned
func TestUpdateUserHandler_Execute_StoresTheNormalizedName(t *testing.T) {
	t.Run("USER-R17: the name is trimmed before it is stored (happy path)", func(t *testing.T) {
		repo := existingUser()

		resp, err := usercommands.NewUpdateUserHandler(repo).Execute(context.Background(), usercommands.UpdateUserRequest{ID: 7, Name: "  Grace "})

		require.NoError(t, err)
		assert.Equal(t, application.UserResponse{ID: 7, Name: "Grace", Email: "ada@example.com"}, *resp)
		assert.Equal(t, []users.User{{ID: 7, Name: "Grace", Email: "ada@example.com"}}, repo.updated)
	})
}

// GIVEN an empty name / WHEN UpdateUser / THEN the domain error is returned and the repository is not asked to write
func TestUpdateUserHandler_Execute_EmptyNameDoesNotWrite(t *testing.T) {
	for _, name := range []string{"", "   "} {
		t.Run("USER-R17: a blank name is refused without writing (sad path)", func(t *testing.T) {
			repo := existingUser()

			resp, err := usercommands.NewUpdateUserHandler(repo).Execute(context.Background(), usercommands.UpdateUserRequest{ID: 7, Name: name})

			assert.Nil(t, resp)
			require.ErrorIs(t, err, users.ErrUserNameEmpty)
			assert.Empty(t, repo.updated)
		})
	}
}

// GIVEN an absent user, or one deleted between the read and the write / WHEN UpdateUser / THEN ErrUserNotFound
func TestUpdateUserHandler_Execute_AbsentUser(t *testing.T) {
	t.Run("USER-R17: an absent user is not found (sad path)", func(t *testing.T) {
		repo := existingUser()
		repo.getFn = func(uint) (*users.User, error) { return nil, users.ErrUserNotFound }

		resp, err := usercommands.NewUpdateUserHandler(repo).Execute(context.Background(), usercommands.UpdateUserRequest{ID: 7, Name: "Grace"})

		assert.Nil(t, resp)
		require.ErrorIs(t, err, users.ErrUserNotFound)
		assert.Empty(t, repo.updated)
	})
	t.Run("USER-R17: a user deleted before the write is not found (exception)", func(t *testing.T) {
		repo := existingUser()
		repo.updateFn = func(*users.User) error { return users.ErrUserNotFound }

		resp, err := usercommands.NewUpdateUserHandler(repo).Execute(context.Background(), usercommands.UpdateUserRequest{ID: 7, Name: "Grace"})

		assert.Nil(t, resp)
		require.ErrorIs(t, err, users.ErrUserNotFound)
	})
}

// GIVEN a technical failure / WHEN UpdateUser / THEN it is propagated unchanged
func TestUpdateUserHandler_Execute_RepositoryErrorIsPropagated(t *testing.T) {
	t.Run("USER-R17: a database failure is propagated (exception)", func(t *testing.T) {
		repo := existingUser()
		repo.updateFn = func(*users.User) error { return errDatabaseDown }

		resp, err := usercommands.NewUpdateUserHandler(repo).Execute(context.Background(), usercommands.UpdateUserRequest{ID: 7, Name: "Grace"})

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, errDatabaseDown)
	})
}

// GIVEN a user id / WHEN DeleteUser / THEN exactly one deletion of that id is requested, and the repository answer is returned as is
func TestDeleteUserHandler_Execute(t *testing.T) {
	t.Run("USER-R13: the repository is asked to delete the id once (happy path)", func(t *testing.T) {
		repo := existingUser()

		err := usercommands.NewDeleteUserHandler(repo).Execute(context.Background(), usercommands.DeleteUserRequest{ID: 7})

		require.NoError(t, err)
		assert.Equal(t, []uint{7}, repo.deleted)
	})
	t.Run("USER-R14: an absent user is not found (sad path)", func(t *testing.T) {
		repo := existingUser()
		repo.deleteFn = func(uint) error { return users.ErrUserNotFound }

		err := usercommands.NewDeleteUserHandler(repo).Execute(context.Background(), usercommands.DeleteUserRequest{ID: 7})

		require.ErrorIs(t, err, users.ErrUserNotFound)
	})
	t.Run("USER-R13: a database failure is propagated (exception)", func(t *testing.T) {
		repo := existingUser()
		repo.deleteFn = func(uint) error { return errDatabaseDown }

		err := usercommands.NewDeleteUserHandler(repo).Execute(context.Background(), usercommands.DeleteUserRequest{ID: 7})

		assert.ErrorIs(t, err, errDatabaseDown)
	})
}
