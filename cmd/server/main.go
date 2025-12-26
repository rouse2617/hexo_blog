package main

import (
	"fmt"
	"log"

	"ai-ops/internal/config"
)

func main() {
	fmt.Println("AI-Ops Server Starting...")
	fmt.Println("Version: 0.1.0")

	// 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	fmt.Printf("配置加载成功:\n")
	fmt.Printf("  - 服务地址: %s\n", cfg.Server.Addr)
	fmt.Printf("  - 运行模式: %s\n", cfg.Server.Mode)
	fmt.Printf("  - LLM 模型: %s\n", cfg.LLM.Model)
	fmt.Printf("  - 数据库: %s (%s)\n", cfg.Database.Driver, cfg.Database.DSN)
	fmt.Printf("  - 日志级别: %s\n", cfg.Log.Level)
}
