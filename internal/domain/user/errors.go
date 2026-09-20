package user

import "errors"

var (
	// ErrUserNameEmpty est levée quand le nom d'utilisateur est vide.
	ErrUserNameEmpty = errors.New("user name cannot be empty")

	// ErrUserEmailEmpty est levée quand l'email est vide.
	ErrUserEmailEmpty = errors.New("user email cannot be empty")

	// ErrUserNotFound est levée quand un utilisateur n'est pas trouvé.
	ErrUserNotFound = errors.New("user not found")
)
