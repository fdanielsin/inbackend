package application

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/application/task/queries"
	"github.com/fdanielsin/inbackend/internal/domain/tasks"
	"github.com/fdanielsin/inbackend/internal/domain/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GIVEN a valid request, completed=true / WHEN CreateTask / THEN the entity is stored as sent and the id returned
func TestCreateTaskHandler_Execute_StoresRequestAsSent(t *testing.T) {
	repo := &stubTaskRepository{createFn: func(*tasks.Task) (uint, error) { return 9, nil }}

	resp, err := commands.NewCreateTaskHandler(repo).Execute(context.Background(), commands.CreateTaskRequest{
		UserID: 3, Title: "Buy milk", Description: "Semi-skimmed", Completed: true,
	})

	require.NoError(t, err)
	assert.Equal(t, commands.CreateTaskResponse{ID: 9, UserID: 3, Title: "Buy milk", Description: "Semi-skimmed", Completed: true}, *resp)
	assert.Equal(t, []tasks.Task{{ID: 0, UserID: 3, Title: "Buy milk", Description: "Semi-skimmed", Completed: true}}, repo.created)
}

// GIVEN an invalid request / WHEN CreateTask / THEN the domain error is returned and nothing is stored
func TestCreateTaskHandler_Execute_ValidationErrorDoesNotStore(t *testing.T) {
	for name, tc := range map[string]struct {
		req  commands.CreateTaskRequest
		want error
	}{
		"empty title": {commands.CreateTaskRequest{UserID: 1, Title: ""}, tasks.ErrTaskTitleEmpty},
		"user id 0":   {commands.CreateTaskRequest{UserID: 0, Title: "Buy milk"}, tasks.ErrTaskUserIDInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			repo := &stubTaskRepository{}

			resp, err := commands.NewCreateTaskHandler(repo).Execute(context.Background(), tc.req)

			assert.Nil(t, resp)
			require.ErrorIs(t, err, tc.want)
			assert.Empty(t, repo.created)
		})
	}
}

// GIVEN the database fails / WHEN CreateTask / THEN the error is propagated
func TestCreateTaskHandler_Execute_RepositoryErrorIsPropagated(t *testing.T) {
	repo := &stubTaskRepository{createFn: func(*tasks.Task) (uint, error) { return 0, errDatabaseDown }}

	resp, err := commands.NewCreateTaskHandler(repo).Execute(context.Background(),
		commands.CreateTaskRequest{UserID: 1, Title: "Buy milk"})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errDatabaseDown)
}

// GIVEN a task id / WHEN DeleteTask / THEN exactly this id is deleted
func TestDeleteTaskHandler_Execute_DeletesRequestedID(t *testing.T) {
	repo := &stubTaskRepository{deleteFn: func(uint) error { return nil }}

	err := commands.NewDeleteTaskHandler(repo).Execute(context.Background(), commands.DeleteTaskRequest{ID: 5})

	require.NoError(t, err)
	assert.Equal(t, []uint{5}, repo.deleted)
}

// GIVEN the database fails / WHEN DeleteTask / THEN the error is propagated
func TestDeleteTaskHandler_Execute_RepositoryErrorIsPropagated(t *testing.T) {
	repo := &stubTaskRepository{deleteFn: func(uint) error { return errDatabaseDown }}

	err := commands.NewDeleteTaskHandler(repo).Execute(context.Background(), commands.DeleteTaskRequest{ID: 5})

	assert.ErrorIs(t, err, errDatabaseDown)
}

// GIVEN a user's tasks / WHEN ListUserTasks / THEN they are mapped in repository order for this user
func TestListUserTasksHandler_Execute_MapsTasksInOrder(t *testing.T) {
	var requested uint
	repo := &stubTaskRepository{listFn: func(userID uint) ([]*tasks.Task, error) {
		requested = userID
		return []*tasks.Task{
			{ID: 1, UserID: userID, Title: "First", Description: "a", Completed: true},
			{ID: 2, UserID: userID, Title: "Second", Description: "", Completed: false},
		}, nil
	}}

	resp, err := queries.NewListUserTasksHandler(repo).Execute(context.Background(), queries.ListUserTasksRequest{UserID: 4})

	require.NoError(t, err)
	assert.Equal(t, uint(4), requested)
	require.Len(t, resp, 2)
	assert.Equal(t, queries.ListUserTasksResponse{ID: 1, UserID: 4, Title: "First", Description: "a", Completed: true}, *resp[0])
	assert.Equal(t, queries.ListUserTasksResponse{ID: 2, UserID: 4, Title: "Second", Description: "", Completed: false}, *resp[1])
}

// GIVEN a user without tasks / WHEN ListUserTasks / THEN an empty, non-nil slice
func TestListUserTasksHandler_Execute_EmptyIsNotNil(t *testing.T) {
	repo := &stubTaskRepository{listFn: func(uint) ([]*tasks.Task, error) { return nil, nil }}

	resp, err := queries.NewListUserTasksHandler(repo).Execute(context.Background(), queries.ListUserTasksRequest{UserID: 4})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp)
}

// GIVEN the database fails / WHEN ListUserTasks / THEN the error is propagated
func TestListUserTasksHandler_Execute_RepositoryErrorIsPropagated(t *testing.T) {
	repo := &stubTaskRepository{listFn: func(uint) ([]*tasks.Task, error) { return nil, errDatabaseDown }}

	resp, err := queries.NewListUserTasksHandler(repo).Execute(context.Background(), queries.ListUserTasksRequest{UserID: 4})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errDatabaseDown)
}

// statsRepositories answers counts only: a stats query that loads the users
// (List) instead of counting them fails, because listFn is not expected.
func statsRepositories(userCount int64, usersErr error, taskCount int64, tasksErr error) (*stubUserRepository, *stubTaskRepository) {
	return &stubUserRepository{countFn: func() (int64, error) { return userCount, usersErr }},
		&stubTaskRepository{countFn: func() (int64, error) { return taskCount, tasksErr }}
}

// GIVEN 3 users and 5 tasks / WHEN GetStats / THEN the exact counters are returned
func TestGetStatsHandler_Execute_ReturnsExactCounters(t *testing.T) {
	userRepo, taskRepo := statsRepositories(3, nil, 5, nil)

	resp, err := queries.NewGetStatsHandler(userRepo, taskRepo).Execute(context.Background(), queries.GetStatsRequest{})

	require.NoError(t, err)
	assert.Equal(t, queries.GetStatsResponse{Users: 3, Tasks: 5}, *resp)
}

// GIVEN listing the users fails / WHEN GetStats / THEN the users are counted, never loaded
func TestGetStatsHandler_Execute_CountsUsersWithoutLoadingThem(t *testing.T) {
	userRepo, taskRepo := statsRepositories(2, nil, 0, nil)
	userRepo.listFn = func() ([]*users.User, error) { return nil, errDatabaseDown }

	resp, err := queries.NewGetStatsHandler(userRepo, taskRepo).Execute(context.Background(), queries.GetStatsRequest{})

	require.NoError(t, err)
	assert.Equal(t, queries.GetStatsResponse{Users: 2, Tasks: 0}, *resp)
}

// GIVEN an empty database / WHEN GetStats / THEN both counters are 0
func TestGetStatsHandler_Execute_EmptyDatabase(t *testing.T) {
	userRepo, taskRepo := statsRepositories(0, nil, 0, nil)

	resp, err := queries.NewGetStatsHandler(userRepo, taskRepo).Execute(context.Background(), queries.GetStatsRequest{})

	require.NoError(t, err)
	assert.Equal(t, queries.GetStatsResponse{Users: 0, Tasks: 0}, *resp)
}

// GIVEN either repository fails / WHEN GetStats / THEN the error is propagated and no partial result returned
func TestGetStatsHandler_Execute_RepositoryErrorIsPropagated(t *testing.T) {
	for name, repos := range map[string][2]error{"users": {errDatabaseDown, nil}, "tasks": {nil, errDatabaseDown}} {
		t.Run(name, func(t *testing.T) {
			userRepo, taskRepo := statsRepositories(1, repos[0], 2, repos[1])

			resp, err := queries.NewGetStatsHandler(userRepo, taskRepo).Execute(context.Background(), queries.GetStatsRequest{})

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, errDatabaseDown)
		})
	}
}

// GIVEN the task cannot be read (database down) / WHEN UpdateTask / THEN the error is propagated and nothing is written
func TestUpdateTaskHandler_Execute_ReadErrorIsPropagated(t *testing.T) {
	repo := &stubTaskRepository{getFn: func(uint) (*tasks.Task, error) { return nil, errDatabaseDown }}

	resp, err := commands.NewUpdateTaskHandler(repo).Execute(context.Background(),
		commands.UpdateTaskRequest{ID: 1, Completed: ptrBool(true)})

	assert.Nil(t, resp)
	require.ErrorIs(t, err, errDatabaseDown)
	assert.Empty(t, repo.updated)
}

// GIVEN the write fails / WHEN UpdateTask / THEN the error is propagated and no response is built
func TestUpdateTaskHandler_Execute_WriteErrorIsPropagated(t *testing.T) {
	repo := &stubTaskRepository{
		getFn:    func(id uint) (*tasks.Task, error) { return &tasks.Task{ID: id, UserID: 1, Title: "Buy milk"}, nil },
		updateFn: func(*tasks.Task) error { return errDatabaseDown },
	}

	resp, err := commands.NewUpdateTaskHandler(repo).Execute(context.Background(),
		commands.UpdateTaskRequest{ID: 1, Completed: ptrBool(true)})

	assert.Nil(t, resp)
	require.ErrorIs(t, err, errDatabaseDown)
	assert.Len(t, repo.updated, 1)
}
