package pgsetup

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // driver pgx5 de golang-migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrate applique toutes les migrations en attente (ADR 0007), après avoir
// refusé une base qui contient les tables de la 1.0.0 (ErrLegacySchema).
func Migrate(dsn string) error {
	if err := rejectLegacySchema(dsn); err != nil {
		return err
	}
	return run(dsn, func(m *migrate.Migrate) error { return m.Up() })
}

// MigrateTo amène le schéma à la version donnée, en montant ou en descendant
// (0 = aucune migration appliquée). Le serveur n'en a pas besoin (il appelle
// Migrate) : elle sert aux tests de migration et aux retours de schéma ciblés.
func MigrateTo(dsn string, version uint) error {
	return run(dsn, func(m *migrate.Migrate) error {
		if version == 0 {
			return m.Down()
		}
		return m.Migrate(version)
	})
}

func run(dsn string, apply func(*migrate.Migrate) error) (err error) {
	url, err := pgx5URL(dsn)
	if err != nil {
		return err
	}
	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, url)
	if err != nil {
		return fmt.Errorf("connect for migrations: %w", err)
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		err = errors.Join(err, sourceErr, databaseErr)
	}()

	if err := apply(m); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return describe(m, err)
	}
	return nil
}

// describe rend l'erreur lisible et donne la marche à suivre : un échec laisse
// la base « dirty » et golang-migrate refuse d'avancer tant que la version n'est
// pas forcée. Le texte vient de PostgreSQL et des messages de nos migrations,
// qui ne contiennent aucune donnée ; le SQL de la migration, que golang-migrate
// joint à l'erreur, est volontairement omis.
func describe(m *migrate.Migrate, err error) error {
	var dbErr database.Error
	if errors.As(err, &dbErr) && dbErr.OrigErr != nil {
		err = dbErr.OrigErr
	}
	version, dirty, versionErr := m.Version()
	if versionErr != nil || !dirty {
		return fmt.Errorf("migration failed: %w", err)
	}
	return fmt.Errorf("migration %d failed and the database is marked dirty: fix the cause, then run "+
		"`make migrate-force version=%d`: %w", version, previousVersion(version), err)
}

// previousVersion est la dernière version entièrement appliquée quand la
// migration version a échoué ; -1 désigne « aucune » pour migrate force.
func previousVersion(version uint) int {
	return int(version) - 1
}

// ErrInvalidDatabaseURL est renvoyée quand l'URL n'est pas une URL postgres://.
// Son message ne reprend jamais la valeur reçue : elle contient un mot de passe.
var ErrInvalidDatabaseURL = errors.New("DATABASE_URL must be a postgres:// URL")

// pgx5URL adapte une URL postgres:// au schéma du driver pgx5 de golang-migrate.
func pgx5URL(dsn string) (string, error) {
	for _, scheme := range []string{"postgres://", "postgresql://"} {
		if rest, found := strings.CutPrefix(dsn, scheme); found {
			return "pgx5://" + rest, nil
		}
	}
	return "", ErrInvalidDatabaseURL
}
