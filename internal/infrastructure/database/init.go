// Package database ouvre la connexion PostgreSQL et applique les migrations.
package database

import (
	"time"

	"go.uber.org/zap"
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
func InitDB(dsn string, logger *zap.Logger) (*gorm.DB, error) {
	logger.Info("Initializing PostgreSQL database")

	if err := Migrate(dsn); err != nil {
		logger.Error("Failed to migrate database", zap.Error(err))
		return nil, err
	}

	// Silent : le logger GORM imprime les requêtes avec leurs valeurs (emails, noms) et le
	// texte brut des erreurs PostgreSQL ; les erreurs sont journalisées par les controllers
	// avec leur seul code.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		logger.Error("Failed to connect to database", zap.Error(err))
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("Failed to access the SQL connection pool", zap.Error(err))
		return nil, err
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	logger.Info("Database initialized successfully")
	return db, nil
}
