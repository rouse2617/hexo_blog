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
	"ai-ops/internal/model"
	"ai-ops/internal/repository"
	"ai-ops/internal/security"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"
	"ai-ops/internal/tool/builtin"
	"ai-ops/pkg/logger"
	"time"

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

	// 2.1 初始化数据库
	db, err := repository.InitDB(cfg.Database.DSN)
	if err != nil {
		logger.Fatal("数据库初始化失败", zap.Error(err))
	}
	logger.Info("数据库初始化完成", zap.String("dsn", cfg.Database.DSN))

	// 创建 Repository 实例
	hostRepo := repository.NewHostRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	configRepo := repository.NewConfigRepository(db)

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

	// 3.2 初始化命令白名单策略与审计日志（默认只读）
	policyStore := security.NewPolicyStore("./data/command_policy.json", security.DefaultCommandPolicy())
	auditLogger := security.NewAuditLogger("./data/command_audit.jsonl")
	ssh.SetCommandPolicyStore(policyStore, auditLogger)

	// 3.1 从数据库加载主机到SSH Pool
	dbHosts, _ := hostRepo.List(repository.HostFilter{})
	for _, dbHost := range dbHosts {
		sshPool.AddHost(ssh.HostInfo{
			Name:       dbHost.Name,
			Host:       dbHost.IP,
			Port:       dbHost.Port,
			User:       dbHost.User,
			Group:      dbHost.Group,
			Tags:       dbHost.Tags,
			AuthType:   dbHost.AuthType,
			Password:   dbHost.Password,
			KeyPath:    dbHost.KeyPath,
			KeyContent: dbHost.KeyContent,
		})
		logger.Debug("从数据库加载主机", zap.String("name", dbHost.Name), zap.String("host", dbHost.IP))
	}
	if len(dbHosts) > 0 {
		logger.Info("从数据库加载主机", zap.Int("count", len(dbHosts)))
	}

	// 3.1 加载配置文件中的预定义主机（如果不存在于数据库则添加）
	for _, h := range cfg.Hosts {
		// 使用默认值填充
		port := h.Port
		if port == 0 {
			port = 22
		}
		user := h.User
		if user == "" {
			user = cfg.SSH.DefaultUser
		}
		authType := h.AuthType
		if authType == "" {
			if h.Password != "" {
				authType = "password"
			} else {
				authType = "key"
			}
		}
		keyPath := h.KeyPath
		if keyPath == "" && authType == "key" {
			keyPath = cfg.SSH.DefaultKeyPath
		}

		// 检查是否已存在于数据库
		_, err := hostRepo.GetByName(h.Name)
		if err != nil {
			// 不存在，添加到数据库
			hostModel := &model.Host{
				ID:        h.Name,
				Name:      h.Name,
				IP:        h.Host,
				Port:      port,
				User:      user,
				Group:     h.Group,
				Tags:      h.Tags,
				AuthType:  authType,
				Password:  h.Password,
				KeyPath:   keyPath,
				Status:    model.HostStatusUnknown,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := hostRepo.Create(hostModel); err != nil {
				logger.Warn("保存配置主机到数据库失败", zap.String("name", h.Name), zap.Error(err))
			}
		}

		// 添加到SSH Pool
		sshPool.AddHost(ssh.HostInfo{
			Name:     h.Name,
			Host:     h.Host,
			Port:     port,
			User:     user,
			Group:    h.Group,
			Tags:     h.Tags,
			AuthType: authType,
			Password: h.Password,
			KeyPath:  keyPath,
		})
		logger.Debug("加载主机配置", zap.String("name", h.Name), zap.String("host", h.Host))
	}
	if len(cfg.Hosts) > 0 {
		logger.Info("从配置文件加载主机", zap.Int("count", len(cfg.Hosts)))
	}

	// 4. 初始化 Tool 注册中心
	toolRegistry := tool.NewRegistry()

	// 注册内置工具
	getHostsFunc := func(group string) []builtin.HostBasicInfo {
		poolHosts := sshPool.ListHosts()
		hosts := make([]builtin.HostBasicInfo, 0, len(poolHosts))
		for _, h := range poolHosts {
			if group != "" && h.Group != group {
				continue
			}
			hosts = append(hosts, builtin.HostBasicInfo{
				Name:   h.Name,
				Host:   h.Host,
				Port:   h.Port,
				User:   h.User,
				Group:  h.Group,
				Status: "unknown",
			})
		}
		return hosts
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
		PolicyStore:  policyStore,
		AuditLogger:  auditLogger,
		HostRepo:     hostRepo,
		SessionRepo:  sessionRepo,
		GroupRepo:    groupRepo,
		ConfigRepo:   configRepo,
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
