package repository

import (
	"ai-ops/internal/model"
	"ai-ops/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// InitDB 初始化数据库
func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移
	err = db.AutoMigrate(
		&model.Host{},
		&model.Group{},
		&model.Session{},
		&model.Message{},
		&model.Config{},
	)
	if err != nil {
		return nil, err
	}

	logger.Info("数据库初始化完成", zap.String("dsn", dsn))
	return db, nil
}

