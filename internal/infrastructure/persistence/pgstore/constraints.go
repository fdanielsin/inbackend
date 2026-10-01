package pgstore

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// Codes SQLSTATE et noms des contraintes créées par les migrations (ADR 0007).
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"

	uniqueEmailIndex   = "idx_user_models_email_lower"
	taskUserForeignKey = "fk_task_models_user"
)

// isConstraintViolation détecte, à partir de la base plutôt que d'un SELECT
// préalable (correct sous requêtes concurrentes), la violation d'une contrainte
// précise : le nom évite de confondre avec une autre contrainte de la table.
func isConstraintViolation(err error, sqlState, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == sqlState && pgErr.ConstraintName == constraint
}
