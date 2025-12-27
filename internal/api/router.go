package api

import (
	"ai-ops/internal/agent"
	"ai-ops/internal/api/handler"
	"ai-ops/internal/llm"
	"ai-ops/internal/repository"
	"ai-ops/internal/security"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"

	"github.com/gin-gonic/gin"
)

// RouterConfig 路由配置
type RouterConfig struct {
	Agent        *agent.Agent
	ToolRegistry *tool.Registry
	SSHPool      *ssh.Pool
	PolicyStore  *security.PolicyStore
	AuditLogger  *security.AuditLogger
	HostRepo     repository.HostRepository
	SessionRepo  repository.SessionRepository
	GroupRepo    repository.GroupRepository
	ConfigRepo   repository.ConfigRepository
	AnalysisRepo repository.AnalysisRepository
	LLMClient    *llm.OpenAIClient
	Version      string
	Mode         string // debug / release
}

// NewRouter 创建路由
func NewRouter(cfg RouterConfig) *gin.Engine {
	// 设置运行模式
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// 中间件
	r.Use(RecoveryMiddleware())
	r.Use(LoggerMiddleware())
	r.Use(CORSMiddleware())

	// 创建 handlers
	chatHandler := handler.NewChatHandler(cfg.Agent, cfg.SessionRepo)
	hostHandler := handler.NewHostHandler(cfg.SSHPool, cfg.HostRepo, cfg.GroupRepo)
	toolHandler := handler.NewToolHandler(cfg.ToolRegistry, cfg.SSHPool, cfg.ConfigRepo)
	systemHandler := handler.NewSystemHandler(cfg.Version, cfg.PolicyStore, cfg.AuditLogger, cfg.ConfigRepo)
	operationsHandler := handler.NewOperationsHandler(cfg.ToolRegistry, cfg.SSHPool)
	analysisHandler := handler.NewAnalysisHandler(cfg.LLMClient, cfg.AnalysisRepo)

	// API 路由组
	api := r.Group("/api")
	{
		// 健康检查
		api.GET("/health", systemHandler.Health)

		// 对话 API
		chat := api.Group("/chat")
		{
			chat.POST("", chatHandler.Chat)
			chat.POST("/send", chatHandler.Chat) // 兼容前端 /chat/send 调用
			chat.POST("/stream", chatHandler.StreamChat)
			chat.GET("/sessions", chatHandler.GetSessions)
			chat.POST("/sessions", chatHandler.CreateSession)
			chat.GET("/history/:session_id", chatHandler.GetHistory)
			chat.DELETE("/sessions/:id", chatHandler.DeleteSession)
		}

		// 主机管理 API
		hosts := api.Group("/hosts")
		{
			hosts.GET("", hostHandler.ListHosts)
			hosts.GET("/all", hostHandler.GetAllHosts)
			hosts.POST("", hostHandler.CreateHost)
			hosts.POST("/batch-delete", hostHandler.BatchDeleteHosts)
			hosts.POST("/import", hostHandler.ImportHosts)
			hosts.GET("/:id", hostHandler.GetHost)
			hosts.PUT("/:id", hostHandler.UpdateHost)
			hosts.DELETE("/:id", hostHandler.DeleteHost)
			hosts.POST("/:id/test", hostHandler.TestConnection)
		}

		// 分组 API
		groups := api.Group("/groups")
		{
			groups.GET("", hostHandler.GetGroups)
			groups.POST("", hostHandler.CreateGroup)
			groups.DELETE("/:name", hostHandler.DeleteGroup)
		}

		// 工具 API
		tools := api.Group("/tools")
		{
			tools.GET("", toolHandler.ListTools)
			tools.GET("/builtin", toolHandler.ListBuiltinTools)
			tools.GET("/script", toolHandler.ListScriptTools)
			tools.GET("/:name", toolHandler.GetTool)
			tools.PUT("/:name/toggle", toolHandler.ToggleTool)
			tools.POST("/:name/execute", toolHandler.ExecuteTool)
		}

		// 系统 API
		system := api.Group("/system")
		{
			system.GET("/info", systemHandler.GetSystemInfo)
			system.GET("/config", systemHandler.GetConfig)
			system.PUT("/config", systemHandler.UpdateConfig)
			system.GET("/command-policy", systemHandler.GetCommandPolicy)
			system.PUT("/command-policy", systemHandler.UpdateCommandPolicy)
			system.GET("/audit/commands", systemHandler.GetCommandAudit)
		}

		// 批量操作 API
		operations := api.Group("/operations")
		{
			operations.POST("/batch-execute", operationsHandler.BatchExecute)
		}

		// AI分析 API
		analysis := api.Group("/analysis")
		{
			analysis.POST("/analyze", analysisHandler.Analyze)
			analysis.GET("/history", analysisHandler.GetHistory)
			analysis.GET("/:id", analysisHandler.GetAnalysis)
			analysis.DELETE("/:id", analysisHandler.DeleteAnalysis)
		}
	}

	// 静态文件服务（前端）
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	return r
}
