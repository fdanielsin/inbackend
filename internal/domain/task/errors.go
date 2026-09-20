package task

import "errors"

var (
	// ErrTaskTitleEmpty est levée quand le titre est vide.
	ErrTaskTitleEmpty = errors.New("task title cannot be empty")

	// ErrTaskUserIDInvalid est levée quand l'ID utilisateur est invalide.
	ErrTaskUserIDInvalid = errors.New("task user_id cannot be empty")

	// ErrTaskNotFound est levée quand une tâche n'est pas trouvée.
	ErrTaskNotFound = errors.New("task not found")
)
