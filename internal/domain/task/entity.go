// Package task contient l'entité, les erreurs et le port de persistance des tâches.
package task

import "strings"

// Task est l'entité métier Task indépendante de toute technologie.
type Task struct {
	ID          uint
	Title       string
	Description string
	UserID      uint
	Completed   bool
}

// NormalizeTitle retire les espaces de début et de fin du titre ; un titre vide
// ou fait d'espaces est refusé.
func NormalizeTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", ErrTaskTitleEmpty
	}
	return title, nil
}

// NormalizeDescription retire les espaces de début et de fin de la description,
// facultative : une description faite d'espaces devient vide.
func NormalizeDescription(description string) string {
	return strings.TrimSpace(description)
}

// NewTask crée une nouvelle entité Task avec validation basique.
func NewTask(id, userID uint, title, description string, completed bool) (*Task, error) {
	title, err := NormalizeTitle(title)
	if err != nil {
		return nil, err
	}
	if userID == 0 {
		return nil, ErrTaskUserIDInvalid
	}

	return &Task{
		ID:          id,
		Title:       title,
		Description: NormalizeDescription(description),
		UserID:      userID,
		Completed:   completed,
	}, nil
}
