package application

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/fdanielsin/inbackend/internal/domain/user"
)

// stubUserRepository records calls and returns canned results.
// A nil function means "not expected": the call panics so the test fails loudly.
type stubUserRepository struct {
	createFn func(u *user.User) (uint, error)
	getFn    func(id uint) (*user.User, error)
	listFn   func() ([]*user.User, error)
	created  []user.User
}

func (r *stubUserRepository) Create(_ context.Context, u *user.User) (uint, error) {
	r.created = append(r.created, *u)
	return r.createFn(u)
}

func (r *stubUserRepository) GetByID(_ context.Context, id uint) (*user.User, error) {
	return r.getFn(id)
}

func (r *stubUserRepository) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	panic("GetByEmail not expected")
}

func (r *stubUserRepository) List(_ context.Context) ([]*user.User, error) {
	return r.listFn()
}

// stubTaskRepository records calls and returns canned results, like stubUserRepository.
type stubTaskRepository struct {
	createFn func(t *task.Task) (uint, error)
	getFn    func(id uint) (*task.Task, error)
	listFn   func(userID uint) ([]*task.Task, error)
	updateFn func(t *task.Task) error
	deleteFn func(id uint) error
	countFn  func() (int64, error)
	created  []task.Task
	updated  []task.Task
	deleted  []uint
}

func (r *stubTaskRepository) Create(_ context.Context, t *task.Task) (uint, error) {
	r.created = append(r.created, *t)
	return r.createFn(t)
}

func (r *stubTaskRepository) GetByID(_ context.Context, id uint) (*task.Task, error) {
	return r.getFn(id)
}

func (r *stubTaskRepository) ListByUserID(_ context.Context, userID uint) ([]*task.Task, error) {
	return r.listFn(userID)
}

func (r *stubTaskRepository) Update(_ context.Context, t *task.Task) error {
	r.updated = append(r.updated, *t)
	return r.updateFn(t)
}

func (r *stubTaskRepository) Delete(_ context.Context, id uint) error {
	r.deleted = append(r.deleted, id)
	return r.deleteFn(id)
}

func (r *stubTaskRepository) Count(_ context.Context) (int64, error) {
	return r.countFn()
}
