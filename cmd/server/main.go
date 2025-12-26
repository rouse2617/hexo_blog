package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"ai-ops/internal/config"
	"ai-ops/pkg/logger"
)

func main() {
	fmt.Println("AI-Ops Server Starting...")
	fmt.Println("Version: 0.1.0")

	// 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	err = logger.Init(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
	})
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 使用日志输出
	logger.Info("配置加载成功",
		zap.String("addr", cfg.Server.Addr),
		zap.String("mode", cfg.Server.Mode),
		zap.String("llm_model", cfg.LLM.Model),
		zap.String("database", cfg.Database.Driver),
	)

	logger.Debug("详细配置信息",
		zap.Duration("llm_timeout", cfg.LLM.Timeout),
		zap.Duration("ssh_timeout", cfg.SSH.DefaultTimeout),
		zap.Int("max_loops", cfg.Agent.MaxLoops),
	)

	logger.Info("AI-Ops 服务启动完成")
}
