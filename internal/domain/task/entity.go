// Package task contient l'entité, les erreurs et le port de persistance des tâches.
package task

// Task est l'entité métier Task indépendante de toute technologie.
type Task struct {
	ID          uint
	Title       string
	Description string
	UserID      uint
	Completed   bool
}

// NewTask crée une nouvelle entité Task avec validation basique.
func NewTask(id, userID uint, title, description string, completed bool) (*Task, error) {
	if title == "" {
		return nil, ErrTaskTitleEmpty
	}
	if userID == 0 {
		return nil, ErrTaskUserIDInvalid
	}

	return &Task{
		ID:          id,
		Title:       title,
		Description: description,
		UserID:      userID,
		Completed:   completed,
	}, nil
}

// CompleteTask marque une tâche comme complétée.
func (t *Task) CompleteTask() {
	t.Completed = true
}

// UncompleteTask marque une tâche comme non complétée.
func (t *Task) UncompleteTask() {
	t.Completed = false
}
