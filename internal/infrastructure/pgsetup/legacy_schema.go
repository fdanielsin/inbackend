package pgsetup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" // driver "pgx" de database/sql
)

// ErrLegacySchema est renvoyée quand la base contient les tables créées par la 1.0.0.
var ErrLegacySchema = errors.New("database has 1.0.0 tables")

// legacyTables liste les colonnes des tables que la 1.0.0 créait avec AutoMigrate
// (models.User et models.Task). La 1.1.0 lit user_models et task_models : sans ce
// refus, elle démarrerait sur des tables vides et masquerait les données existantes.
// Les colonnes sont vérifiées en plus du nom pour ne pas refuser une base qui
// partagerait une table sans rapport appelée users ou tasks.
var legacyTables = map[string][]string{
	"tasks": {"id", "title", "description", "user_id", "completed"},
	"users": {"id", "name", "email"},
}

// rejectLegacySchema refuse la base si elle contient une table de la 1.0.0 dans son
// schéma courant, quel que soit l'état de schema_migrations. Rien n'est écrit.
func rejectLegacySchema(dsn string) error {
	if _, err := pgx5URL(dsn); err != nil {
		return err
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("connect to check the schema: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(context.Background()); err != nil {
		return fmt.Errorf("connect to the database: %w", err)
	}

	var found []string
	for _, table := range []string{"users", "tasks"} {
		var present int
		err := db.QueryRowContext(context.Background(), `
			SELECT count(*) FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = $1 AND column_name = ANY($2)`,
			table, legacyTables[table]).Scan(&present)
		if err != nil {
			return fmt.Errorf("check for 1.0.0 tables: %w", err)
		}
		if present == len(legacyTables[table]) {
			found = append(found, table)
		}
	}
	if len(found) == 0 {
		return nil
	}
	return fmt.Errorf("%w (%s): this version reads user_models and task_models and would ignore your data; "+
		"copy it first, see docs/troubleshooting.md in the inbackend repository",
		ErrLegacySchema, strings.Join(found, ", "))
}
