package application

import (
	"context"
	"errors"
	"testing"

	"github.com/fdanielsin/inbackend/internal/application"
	usercommands "github.com/fdanielsin/inbackend/internal/application/user/commands"
	userqueries "github.com/fdanielsin/inbackend/internal/application/user/queries"
	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errDatabaseDown = errors.New("connection refused")

// GIVEN a valid request / WHEN CreateUser / THEN the new entity (id 0) is stored and the generated id returned
func TestCreateUserHandler_Execute_ReturnsGeneratedID(t *testing.T) {
	repo := &stubUserRepository{createFn: func(*users.User) (uint, error) { return 42, nil }}

	resp, err := usercommands.NewCreateUserHandler(repo).Execute(context.Background(),
		usercommands.CreateUserRequest{Name: "Ada", Email: "ada@example.com"})

	require.NoError(t, err)
	assert.Equal(t, application.UserResponse{ID: 42, Name: "Ada", Email: "ada@example.com"}, *resp)
	assert.Equal(t, []users.User{{ID: 0, Name: "Ada", Email: "ada@example.com"}}, repo.created)
}

// GIVEN an invalid request / WHEN CreateUser / THEN the domain error is returned and nothing is stored
func TestCreateUserHandler_Execute_ValidationErrorDoesNotStore(t *testing.T) {
	for name, tc := range map[string]struct {
		req  usercommands.CreateUserRequest
		want error
	}{
		"empty name":  {usercommands.CreateUserRequest{Name: "", Email: "ada@example.com"}, users.ErrUserNameEmpty},
		"empty email": {usercommands.CreateUserRequest{Name: "Ada", Email: ""}, users.ErrUserEmailEmpty},
	} {
		t.Run(name, func(t *testing.T) {
			repo := &stubUserRepository{}

			resp, err := usercommands.NewCreateUserHandler(repo).Execute(context.Background(), tc.req)

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, tc.want)
			assert.Empty(t, repo.created)
		})
	}
}

// GIVEN the database fails / WHEN CreateUser / THEN the technical error is propagated unchanged
func TestCreateUserHandler_Execute_RepositoryErrorIsPropagated(t *testing.T) {
	repo := &stubUserRepository{createFn: func(*users.User) (uint, error) { return 0, errDatabaseDown }}

	resp, err := usercommands.NewCreateUserHandler(repo).Execute(context.Background(),
		usercommands.CreateUserRequest{Name: "Ada", Email: "ada@example.com"})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errDatabaseDown)
}

// GIVEN a stored user / WHEN GetUser / THEN its fields are returned for the requested id
func TestGetUserHandler_Execute_ReturnsUser(t *testing.T) {
	var requested uint
	repo := &stubUserRepository{getFn: func(id uint) (*users.User, error) {
		requested = id
		return &users.User{ID: id, Name: "Ada", Email: "ada@example.com"}, nil
	}}

	resp, err := userqueries.NewGetUserHandler(repo).Execute(context.Background(), userqueries.GetUserRequest{ID: 7})

	require.NoError(t, err)
	assert.Equal(t, uint(7), requested)
	assert.Equal(t, application.UserResponse{ID: 7, Name: "Ada", Email: "ada@example.com"}, *resp)
}

// GIVEN the repository fails / WHEN GetUser / THEN not-found and technical errors are propagated
func TestGetUserHandler_Execute_ErrorsArePropagated(t *testing.T) {
	for name, repoErr := range map[string]error{"not found": users.ErrUserNotFound, "database down": errDatabaseDown} {
		t.Run(name, func(t *testing.T) {
			repo := &stubUserRepository{getFn: func(uint) (*users.User, error) { return nil, repoErr }}

			resp, err := userqueries.NewGetUserHandler(repo).Execute(context.Background(), userqueries.GetUserRequest{ID: 7})

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, repoErr)
		})
	}
}

// GIVEN two users / WHEN ListUsers / THEN both are returned in repository order
func TestListUsersHandler_Execute_KeepsRepositoryOrder(t *testing.T) {
	repo := &stubUserRepository{listFn: func() ([]*users.User, error) {
		return []*users.User{{ID: 2, Name: "Bob", Email: "bob@example.com"}, {ID: 1, Name: "Ada", Email: "ada@example.com"}}, nil
	}}

	resp, err := userqueries.NewListUsersHandler(repo).Execute(context.Background(), userqueries.ListUsersRequest{})

	require.NoError(t, err)
	require.Len(t, resp, 2)
	assert.Equal(t, application.UserResponse{ID: 2, Name: "Bob", Email: "bob@example.com"}, *resp[0])
	assert.Equal(t, application.UserResponse{ID: 1, Name: "Ada", Email: "ada@example.com"}, *resp[1])
}

// GIVEN no user / WHEN ListUsers / THEN an empty, non-nil slice (serialized as [] and not null)
func TestListUsersHandler_Execute_EmptyIsNotNil(t *testing.T) {
	repo := &stubUserRepository{listFn: func() ([]*users.User, error) { return nil, nil }}

	resp, err := userqueries.NewListUsersHandler(repo).Execute(context.Background(), userqueries.ListUsersRequest{})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp)
}

// GIVEN the database fails / WHEN ListUsers / THEN the error is propagated
func TestListUsersHandler_Execute_RepositoryErrorIsPropagated(t *testing.T) {
	repo := &stubUserRepository{listFn: func() ([]*users.User, error) { return nil, errDatabaseDown }}

	resp, err := userqueries.NewListUsersHandler(repo).Execute(context.Background(), userqueries.ListUsersRequest{})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errDatabaseDown)
}
