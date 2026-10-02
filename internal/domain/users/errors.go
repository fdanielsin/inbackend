package users

import "errors"

var (
	// ErrUserNameEmpty est levée quand le nom d'utilisateur est vide.
	ErrUserNameEmpty = errors.New("user name cannot be empty")

	// ErrUserEmailEmpty est levée quand l'email est vide.
	ErrUserEmailEmpty = errors.New("user email cannot be empty")

	// ErrUserEmailInvalid est levée quand l'email n'a pas le format minimal attendu.
	ErrUserEmailInvalid = errors.New("user email is invalid")

	// ErrUserNotFound est levée quand un utilisateur n'est pas trouvé.
	ErrUserNotFound = errors.New("user not found")

	// ErrUserEmailTaken est levée quand un utilisateur avec le même email existe déjà.
	ErrUserEmailTaken = errors.New("user email already in use")
)
