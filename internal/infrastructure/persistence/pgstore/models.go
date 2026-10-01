// Package pgstore implémente les ports de persistance avec GORM et PostgreSQL.
package pgstore

// UserModel is the GORM model used to persist users.
type UserModel struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Email string `gorm:"uniqueIndex"`
}

// TaskModel is the GORM model used to persist tasks.
type TaskModel struct {
	ID          uint `gorm:"primaryKey"`
	Title       string
	Description string
	UserID      uint
	Completed   bool
}
