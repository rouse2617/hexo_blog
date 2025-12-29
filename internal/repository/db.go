package repository

import (
	"ai-ops/internal/model"
	"ai-ops/pkg/logger"
	"fmt"

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

	// 获取底层 SQL DB 连接以执行 PRAGMA 命令
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	// 配置 SQLite 性能优化参数
	pragmaStatements := []string{
		// 启用 WAL 模式 (Write-Ahead Logging)
		// 优势：读写并发、更好的性能、自动检查点
		"PRAGMA journal_mode=WAL;",

		// 同步模式：NORMAL 在 WAL 模式下提供良好的安全性和性能平衡
		// FULL: 每次写入都同步到磁盘（最安全但最慢）
		// NORMAL: 每次关键操作同步（推荐）
		// OFF: 不同步（最快但不安全）
		"PRAGMA synchronous=NORMAL;",

		// 缓存大小：-2000 表示 2000KB = 2MB
		// 默认值通常是 -2000，可以根据需要调整
		// 负值表示 KB，正值表示页数（每页 4KB）
		"PRAGMA cache_size=-2000;",

		// WAL 文件大小限制：1000 个页 = 4MB
		// 当 WAL 文件达到这个大小时，执行检查点
		"PRAGMA wal_autocheckpoint=1000;",

		// 忙时超时：5 秒
		// 当数据库被锁定时，等待其他事务完成的时间
		"PRAGMA busy_timeout=5000;",

		// 临时存储：内存
		// 将临时表和索引存储在内存中
		"PRAGMA temp_store=MEMORY;",

		// MMAP 大小：256MB
		// 允许 SQLite 使用内存映射 I/O
		"PRAGMA mmap_size=268435456;",

		// 页面大小：4096 字节
		// 大多数系统上的最佳默认值
		"PRAGMA page_size=4096;",

		// 外键约束：启用
		"PRAGMA foreign_keys=ON;",

		// 查询优化器设置
		"PRAGMA optimize;",
	}

	// 执行所有 PRAGMA 命令
	for _, pragma := range pragmaStatements {
		if _, err := sqlDB.Exec(pragma); err != nil {
			logger.Warn("执行 PRAGMA 命令失败", zap.String("pragma", pragma), zap.Error(err))
			// 继续执行，不因为单个 PRAGMA 失败而终止
		}
	}

	// 自动迁移
	err = db.AutoMigrate(
		&model.Host{},
		&model.Group{},
		&model.Session{},
		&model.Message{},
		&model.Config{},
		&model.Analysis{},
	)
	if err != nil {
		return nil, err
	}

	// 创建额外的性能优化索引
	if err := createPerformanceIndexes(db); err != nil {
		logger.Warn("创建性能索引失败", zap.Error(err))
	}

	logger.Info("数据库初始化完成",
		zap.String("dsn", dsn),
		zap.String("journal_mode", "WAL"),
		zap.String("synchronous", "NORMAL"),
	)

	return db, nil
}

// createPerformanceIndexes 创建性能优化索引
func createPerformanceIndexes(db *gorm.DB) error {
	// 为 Message 表创建复合索引（session_id + created_at）
	// 优化按会话查询消息并按时间排序的场景
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_messages_session_created
		ON messages(session_id, created_at DESC)
	`).Error; err != nil {
		return fmt.Errorf("创建消息索引失败: %w", err)
	}

	// 为 Host 表创建状态索引
	// 优化按状态筛选主机的查询
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_hosts_status
		ON hosts(status)
	`).Error; err != nil {
		return fmt.Errorf("创建主机状态索引失败: %w", err)
	}

	// 为 Host 表创建组+状态复合索引
	// 优化按组和状态筛选的场景
	// 注意：group 是 SQLite 保留关键字，需要用引号包裹
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_hosts_group_status
		ON hosts("group", status)
	`).Error; err != nil {
		return fmt.Errorf("创建主机组状态索引失败: %w", err)
	}

	// 为 Analysis 表创建会话+时间复合索引
	// 优化查询特定会话的分析历史
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_analysis_session_created
		ON analysis(session_id, created_at DESC)
	`).Error; err != nil {
		return fmt.Errorf("创建分析索引失败: %w", err)
	}

	// 为 Message 表创建角色索引
	// 优化按角色筛选消息的查询
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_messages_role
		ON messages(role)
	`).Error; err != nil {
		return fmt.Errorf("创建消息角色索引失败: %w", err)
	}

	return nil
}

