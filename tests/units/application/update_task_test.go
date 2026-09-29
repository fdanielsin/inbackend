package application

import (
	"context"
	"testing"

	"github.com/fdanielsin/inbackend/internal/application/task/commands"
	"github.com/fdanielsin/inbackend/internal/domain/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTaskRepository est un double de test en mémoire pour task.Repository.
// Il n'exerce pas le comportement GORM (Select/Updates/RowsAffected, testé
// par les tests d'intégration PostgreSQL) : il vérifie uniquement la logique
// de fusion et de validation d'UpdateTaskHandler.Execute.
type fakeTaskRepository struct {
	tasks       map[uint]*task.Task
	getByIDErr  error
	updateCalls []task.Task // copies des valeurs passées à Update, dans l'ordre
}

func newFakeTaskRepository(existing *task.Task) *fakeTaskRepository {
	tasks := make(map[uint]*task.Task)
	if existing != nil {
		cp := *existing
		tasks[existing.ID] = &cp
	}
	return &fakeTaskRepository{tasks: tasks}
}

func (r *fakeTaskRepository) Create(ctx context.Context, t *task.Task) (uint, error) {
	panic("not used by these tests")
}

func (r *fakeTaskRepository) GetByID(ctx context.Context, id uint) (*task.Task, error) {
	if r.getByIDErr != nil {
		return nil, r.getByIDErr
	}
	t, ok := r.tasks[id]
	if !ok {
		return nil, task.ErrTaskNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *fakeTaskRepository) ListByUserID(ctx context.Context, userID uint) ([]*task.Task, error) {
	panic("not used by these tests")
}

func (r *fakeTaskRepository) Update(ctx context.Context, t *task.Task) error {
	r.updateCalls = append(r.updateCalls, *t)
	r.tasks[t.ID] = t
	return nil
}

func (r *fakeTaskRepository) Delete(ctx context.Context, id uint) error {
	panic("not used by these tests")
}

func (r *fakeTaskRepository) Count(ctx context.Context) (int64, error) {
	panic("not used by these tests")
}

func ptrString(s string) *string { return &s }
func ptrBool(b bool) *bool       { return &b }

// GIVEN une tâche terminée avec titre et description
// WHEN  PUT ne fournit que completed:false
// THEN  titre et description restent inchangés, completed passe à false
func TestUpdateTaskHandler_Execute_CompletedFalseOnly_PreservesTitleAndDescription(t *testing.T) {
	existing := &task.Task{ID: 1, UserID: 1, Title: "Buy milk", Description: "whole milk", Completed: true}
	repo := newFakeTaskRepository(existing)
	handler := commands.NewUpdateTaskHandler(repo)

	resp, err := handler.Execute(context.Background(), commands.UpdateTaskRequest{
		ID:        1,
		Completed: ptrBool(false),
	})

	require.NoError(t, err)
	assert.Equal(t, "Buy milk", resp.Title)
	assert.Equal(t, "whole milk", resp.Description)
	assert.Equal(t, false, resp.Completed)

	require.Len(t, repo.updateCalls, 1)
	persisted := repo.updateCalls[0]
	assert.Equal(t, "Buy milk", persisted.Title)
	assert.Equal(t, "whole milk", persisted.Description)
	assert.Equal(t, false, persisted.Completed)
}

// GIVEN une tâche avec une description existante
// WHEN  PUT fournit explicitement description:""
// THEN  la description est vidée (valeur explicite, pas une absence)
func TestUpdateTaskHandler_Execute_DescriptionEmptyExplicit_IsApplied(t *testing.T) {
	existing := &task.Task{ID: 2, UserID: 1, Title: "Buy milk", Description: "whole milk", Completed: false}
	repo := newFakeTaskRepository(existing)
	handler := commands.NewUpdateTaskHandler(repo)

	resp, err := handler.Execute(context.Background(), commands.UpdateTaskRequest{
		ID:          2,
		Description: ptrString(""),
	})

	require.NoError(t, err)
	assert.Equal(t, "", resp.Description)
	assert.Equal(t, "Buy milk", resp.Title, "le titre non fourni doit rester inchangé")

	require.Len(t, repo.updateCalls, 1)
	assert.Equal(t, "", repo.updateCalls[0].Description)
}

// GIVEN une tâche existante
// WHEN  PUT fournit title:""
// THEN  400 métier (ErrTaskTitleEmpty), rien n'est persisté
func TestUpdateTaskHandler_Execute_TitleEmptyExplicit_ReturnsValidationError(t *testing.T) {
	existing := &task.Task{ID: 3, UserID: 1, Title: "Buy milk", Description: "whole milk", Completed: false}
	repo := newFakeTaskRepository(existing)
	handler := commands.NewUpdateTaskHandler(repo)

	resp, err := handler.Execute(context.Background(), commands.UpdateTaskRequest{
		ID:    3,
		Title: ptrString(""),
	})

	require.ErrorIs(t, err, task.ErrTaskTitleEmpty)
	assert.Nil(t, resp)
	assert.Empty(t, repo.updateCalls, "aucune écriture ne doit être tentée si la validation échoue")
}

// GIVEN une tâche existante
// WHEN  PUT ne fournit aucun champ (tous les pointeurs nil hormis l'ID)
// THEN  la tâche persistée est identique à l'état existant (idempotent)
func TestUpdateTaskHandler_Execute_NoFieldsProvided_IsNoOp(t *testing.T) {
	existing := &task.Task{ID: 4, UserID: 1, Title: "Buy milk", Description: "whole milk", Completed: true}
	repo := newFakeTaskRepository(existing)
	handler := commands.NewUpdateTaskHandler(repo)

	resp, err := handler.Execute(context.Background(), commands.UpdateTaskRequest{ID: 4})

	require.NoError(t, err)
	assert.Equal(t, "Buy milk", resp.Title)
	assert.Equal(t, "whole milk", resp.Description)
	assert.Equal(t, true, resp.Completed)

	require.Len(t, repo.updateCalls, 1)
	assert.Equal(t, *existing, repo.updateCalls[0])
}

// GIVEN un ID qui ne correspond à aucune tâche
// WHEN  PUT est exécuté
// THEN  ErrTaskNotFound remonte tel quel, Update n'est jamais appelé
func TestUpdateTaskHandler_Execute_TaskNotFound_PropagatesError(t *testing.T) {
	repo := newFakeTaskRepository(nil)
	handler := commands.NewUpdateTaskHandler(repo)

	resp, err := handler.Execute(context.Background(), commands.UpdateTaskRequest{
		ID:        99,
		Completed: ptrBool(true),
	})

	require.ErrorIs(t, err, task.ErrTaskNotFound)
	assert.Nil(t, resp)
	assert.Empty(t, repo.updateCalls)
}
