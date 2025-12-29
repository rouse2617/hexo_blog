package main

import (
	"ai-ops/internal/crypto"
	"ai-ops/internal/model"
	"ai-ops/pkg/logger"
	"flag"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 命令行参数
	dbPath := flag.String("db", "./data/aiops.db", "数据库路径")
	keyBase64 := flag.String("key", "", "Base64 编码的加密密钥（或从 ENCRYPTION_BASE64_KEY 环境变量读取）")
	dryRun := flag.Bool("dry-run", false, "模拟运行，不实际修改数据")
	force := flag.Bool("force", false, "强制重新加密已加密的数据")

	flag.Parse()

	// 初始化日志
	logger.Init(nil)
	defer logger.Sync()

	// 获取加密密钥
	var encryptor *crypto.Encryptor
	var err error

	if *keyBase64 != "" {
		encryptor, err = crypto.NewEncryptorFromBase64Key(*keyBase64)
	} else {
		encryptor, err = crypto.NewEncryptorFromEnv()
	}

	if err != nil {
		logger.Fatal("无法创建加密器",
			zap.Error(err),
			zap.String("hint", "请设置 -key 参数或 ENCRYPTION_BASE64_KEY 环境变量"))
	}

	logger.Info("加密器初始化成功",
		zap.String("algorithm", "AES-256-GCM"),
		zap.Bool("dry_run", *dryRun))

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{})
	if err != nil {
		logger.Fatal("数据库连接失败", zap.Error(err))
	}

	// 自动迁移
	if err := db.AutoMigrate(&model.Host{}); err != nil {
		logger.Fatal("数据库迁移失败", zap.Error(err))
	}

	// 查询所有主机
	var hosts []model.Host
	if err := db.Find(&hosts).Error; err != nil {
		logger.Fatal("查询主机失败", zap.Error(err))
	}

	logger.Info("找到主机", zap.Int("count", len(hosts)))

	// 统计
	stats := struct {
		Total         int
		Encrypted     int
		Decrypted     int
		Updated       int
		Skipped       int
		AlreadyEnc     int
	}{}

	// 处理每个主机
	for _, host := range hosts {
		stats.Total++
		needsUpdate := false

		// 检查密码
		if host.Password != "" {
			isEncrypted := crypto.IsEncrypted(host.Password)
			if isEncrypted {
				stats.AlreadyEnc++
				if *force {
					// 强制重新加密
					if encrypted, err := encryptor.Encrypt(host.Password); err == nil {
						host.Password = encrypted
						needsUpdate = true
						logger.Info("重新加密密码",
							zap.String("host", host.Name),
							zap.String("field", "password"))
					}
				}
			} else {
				stats.Decrypted++
				// 尝试解密（如果已经是旧格式加密）
				decrypted, err := encryptor.Decrypt(host.Password)
				if err == nil {
					// 已经是旧密钥加密的，需要重新加密
					stats.Encrypted++
					if encrypted, err := encryptor.Encrypt(decrypted); err == nil {
						host.Password = encrypted
						needsUpdate = true
						logger.Info("重新加密密码（密钥更换）",
							zap.String("host", host.Name),
							zap.String("field", "password"))
					}
				} else {
					// 明文，需要加密
					stats.Encrypted++
					if encrypted, err := encryptor.Encrypt(host.Password); err == nil {
						host.Password = encrypted
						needsUpdate = true
						logger.Info("加密密码",
							zap.String("host", host.Name),
							zap.String("field", "password"))
					}
				}
			}
		}

		// 检查私钥
		if host.KeyContent != "" {
			isEncrypted := crypto.IsEncrypted(host.KeyContent)
			if isEncrypted {
				if *force {
					// 强制重新加密
					if encrypted, err := encryptor.Encrypt(host.KeyContent); err == nil {
						host.KeyContent = encrypted
						needsUpdate = true
						logger.Info("重新加密私钥",
							zap.String("host", host.Name),
							zap.String("field", "key_content"))
					}
				}
			} else {
				// 尝试解密（如果已经是旧格式加密）
				decrypted, err := encryptor.Decrypt(host.KeyContent)
				if err == nil {
					// 已经是旧密钥加密的，需要重新加密
					if encrypted, err := encryptor.Encrypt(decrypted); err == nil {
						host.KeyContent = encrypted
						needsUpdate = true
						logger.Info("重新加密私钥（密钥更换）",
							zap.String("host", host.Name),
							zap.String("field", "key_content"))
					}
				} else {
					// 明文，需要加密
					if encrypted, err := encryptor.Encrypt(host.KeyContent); err == nil {
						host.KeyContent = encrypted
						needsUpdate = true
						logger.Info("加密私钥",
							zap.String("host", host.Name),
							zap.String("field", "key_content"))
					}
				}
			}
		}

		// 更新数据库
		if needsUpdate {
			if *dryRun {
				stats.Updated++
				logger.Info("【模拟运行】将更新主机",
					zap.String("host", host.Name),
					zap.Bool("has_password", host.Password != ""),
					zap.Bool("has_key", host.KeyContent != ""))
			} else {
				if err := db.Save(&host).Error; err != nil {
					logger.Error("更新主机失败",
						zap.String("host", host.Name),
						zap.Error(err))
				} else {
					stats.Updated++
					logger.Info("已更新主机",
						zap.String("host", host.Name))
				}
			}
		} else {
			stats.Skipped++
		}
	}

	// 打印统计
	fmt.Println("\n========== 迁移统计 ==========")
	fmt.Printf("总主机数:        %d\n", stats.Total)
	fmt.Printf("已加密字段数:    %d\n", stats.AlreadyEnc)
	fmt.Printf("明文字段数:      %d\n", stats.Decrypted)
	fmt.Printf("需加密字段数:    %d\n", stats.Encrypted)
	fmt.Printf("已更新主机数:    %d\n", stats.Updated)
	fmt.Printf("跳过主机数:      %d\n", stats.Skipped)

	if *dryRun {
		fmt.Println("\n【注意】这是模拟运行，未实际修改数据")
		fmt.Println("如需实际执行，请移除 -dry-run 参数")
	}

	fmt.Println("==============================")
}
