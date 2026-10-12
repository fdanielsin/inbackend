package application

import (
	"context"

	"github.com/fdanielsin/inbackend/internal/domain/tasks"
	"github.com/fdanielsin/inbackend/internal/domain/users"
)

// stubUserRepository records calls and returns canned results.
// A nil function means "not expected": the call panics so the test fails loudly.
type stubUserRepository struct {
	createFn func(u *users.User) (uint, error)
	getFn    func(id uint) (*users.User, error)
	listFn   func() ([]*users.User, error)
	countFn  func() (int64, error)
	updateFn func(u *users.User) error
	deleteFn func(id uint) error
	created  []users.User
	updated  []users.User
	deleted  []uint
}

func (r *stubUserRepository) Update(_ context.Context, u *users.User) error {
	r.updated = append(r.updated, *u)
	return r.updateFn(u)
}

func (r *stubUserRepository) Delete(_ context.Context, id uint) error {
	r.deleted = append(r.deleted, id)
	return r.deleteFn(id)
}

func (r *stubUserRepository) Create(_ context.Context, u *users.User) (uint, error) {
	r.created = append(r.created, *u)
	return r.createFn(u)
}

func (r *stubUserRepository) GetByID(_ context.Context, id uint) (*users.User, error) {
	return r.getFn(id)
}

func (r *stubUserRepository) GetByEmail(_ context.Context, _ string) (*users.User, error) {
	panic("GetByEmail not expected")
}

func (r *stubUserRepository) List(_ context.Context) ([]*users.User, error) {
	return r.listFn()
}

func (r *stubUserRepository) Count(_ context.Context) (int64, error) {
	return r.countFn()
}

// stubTaskRepository records calls and returns canned results, like stubUserRepository.
type stubTaskRepository struct {
	createFn func(t *tasks.Task) (uint, error)
	getFn    func(id uint) (*tasks.Task, error)
	listFn   func(userID uint) ([]*tasks.Task, error)
	updateFn func(t *tasks.Task) error
	deleteFn func(id uint) error
	countFn  func() (int64, error)
	created  []tasks.Task
	updated  []tasks.Task
	deleted  []uint
	filters  []tasks.ListFilter
}

func (r *stubTaskRepository) Create(_ context.Context, t *tasks.Task) (uint, error) {
	r.created = append(r.created, *t)
	return r.createFn(t)
}

func (r *stubTaskRepository) GetByID(_ context.Context, id uint) (*tasks.Task, error) {
	return r.getFn(id)
}

func (r *stubTaskRepository) ListByUserID(_ context.Context, userID uint, filter tasks.ListFilter) ([]*tasks.Task, error) {
	r.filters = append(r.filters, filter)
	return r.listFn(userID)
}

func (r *stubTaskRepository) Update(_ context.Context, t *tasks.Task) error {
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
