package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ai-ops/internal/agent"
	"ai-ops/internal/api"
	"ai-ops/internal/config"
	"ai-ops/internal/llm"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"
	"ai-ops/internal/tool/builtin"
	"ai-ops/pkg/logger"

	"go.uber.org/zap"
)

const Version = "0.1.0"

func main() {
	fmt.Println("AI-Ops Server Starting...")
	fmt.Printf("Version: %s\n", Version)

	// 1. 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
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

	logger.Info("配置加载成功",
		zap.String("addr", cfg.Server.Addr),
		zap.String("mode", cfg.Server.Mode),
		zap.String("llm_model", cfg.LLM.Model),
	)

	// 3. 初始化 SSH 连接池
	sshPool := ssh.NewPool(ssh.Config{
		DefaultTimeout:    cfg.SSH.DefaultTimeout,
		ConnectTimeout:    cfg.SSH.ConnectTimeout,
		MaxConnections:    cfg.SSH.MaxConnections,
		MaxConcurrent:     cfg.SSH.MaxConcurrent,
		KeepaliveInterval: cfg.SSH.KeepaliveInterval,
	})
	defer sshPool.Close()
	logger.Info("SSH 连接池初始化完成")

	// 4. 初始化 Tool 注册中心
	toolRegistry := tool.NewRegistry()

	// 注册内置工具
	getHostsFunc := func(group string) []builtin.HostBasicInfo {
		// TODO: 从数据库获取主机列表
		return []builtin.HostBasicInfo{}
	}
	if err := builtin.RegisterAll(toolRegistry, getHostsFunc); err != nil {
		logger.Fatal("注册内置工具失败", zap.Error(err))
	}
	logger.Info("Tool 注册中心初始化完成", zap.Int("tools", toolRegistry.Count()))

	// 5. 初始化 LLM 客户端
	llmClient := llm.NewOpenAIClient(llm.OpenAIConfig{
		Endpoint:  cfg.LLM.Endpoint,
		Model:     cfg.LLM.Model,
		APIKey:    cfg.LLM.APIKey,
		Timeout:   cfg.LLM.Timeout,
		MaxTokens: cfg.LLM.MaxTokens,
	})
	logger.Info("LLM 客户端初始化完成",
		zap.String("endpoint", cfg.LLM.Endpoint),
		zap.String("model", cfg.LLM.Model),
	)

	// 6. 初始化 Agent
	aiAgent := agent.NewAgent(llmClient, toolRegistry, sshPool, agent.Config{
		MaxLoops: cfg.Agent.MaxLoops,
	})
	logger.Info("Agent 初始化完成", zap.Int("max_loops", cfg.Agent.MaxLoops))

	// 7. 初始化 HTTP 路由
	router := api.NewRouter(api.RouterConfig{
		Agent:        aiAgent,
		ToolRegistry: toolRegistry,
		SSHPool:      sshPool,
		Version:      Version,
		Mode:         cfg.Server.Mode,
	})
	logger.Info("HTTP 路由初始化完成")

	// 8. 启动服务
	go func() {
		logger.Info("启动 HTTP 服务", zap.String("addr", cfg.Server.Addr))
		if err := router.Run(cfg.Server.Addr); err != nil {
			logger.Fatal("HTTP 服务启动失败", zap.Error(err))
		}
	}()

	// 9. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务...")
	logger.Info("服务已关闭")
}
