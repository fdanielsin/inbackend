// Package database ouvre la connexion PostgreSQL et applique les migrations.
package database

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Pool de connexions : borne la charge sur PostgreSQL (100 connexions par
// défaut côté serveur, partagées entre instances) et recycle les connexions.
const (
	maxOpenConns    = 25
	maxIdleConns    = 5
	connMaxLifetime = 30 * time.Minute
)

// InitDB applique les migrations en attente, puis ouvre la connexion PostgreSQL.
// Les échecs sont retournés, enveloppés, sans être journalisés : c'est à l'appelant de le faire.
func InitDB(dsn string, log *slog.Logger) (*gorm.DB, error) {
	log.Info("Initializing PostgreSQL database")

	if err := Migrate(dsn); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	// Silent : le logger GORM imprime les requêtes avec leurs valeurs (emails, noms) et le
	// texte brut des erreurs PostgreSQL ; les erreurs sont journalisées par les controllers
	// avec leur seul code.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("access SQL connection pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	log.Info("Database initialized successfully")
	return db, nil
}
