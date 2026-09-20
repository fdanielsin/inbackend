package database

import (
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	persistencepg "github.com/fdanielsin/inbackend/internal/infrastructure/persistence/postgres"
)

func InitDB(dsn string, logger *zap.Logger) (*gorm.DB, error) {
	logger.Info("Initializing PostgreSQL database")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
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
