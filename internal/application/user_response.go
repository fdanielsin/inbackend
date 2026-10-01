// Package application regroupe ce que partagent les cas d'utilisation des différents contextes.
package application

// UserResponse est la représentation d'un utilisateur renvoyée par les cas d'utilisation.
type UserResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
