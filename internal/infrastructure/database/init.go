// Package database ouvre la connexion PostgreSQL et applique les migrations.
package database

import (
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	persistencepg "github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
)

// InitDB ouvre la connexion PostgreSQL et applique les migrations automatiques.
func InitDB(dsn string, logger *zap.Logger) (*gorm.DB, error) {
	logger.Info("Initializing PostgreSQL database")

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

	err = db.AutoMigrate(&persistencepg.UserModel{}, &persistencepg.TaskModel{})
	if err != nil {
		logger.Error("Failed to migrate database", zap.Error(err))
		return nil, err
	}

	logger.Info("Database initialized successfully")
	return db, nil
}
