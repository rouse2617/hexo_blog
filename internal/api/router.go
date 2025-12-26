package api

import (
	"ai-ops/internal/agent"
	"ai-ops/internal/api/handler"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"

	"github.com/gin-gonic/gin"
)

// RouterConfig 路由配置
type RouterConfig struct {
	Agent        *agent.Agent
	ToolRegistry *tool.Registry
	SSHPool      *ssh.Pool
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
	chatHandler := handler.NewChatHandler(cfg.Agent)
	hostHandler := handler.NewHostHandler(cfg.SSHPool)
	toolHandler := handler.NewToolHandler(cfg.ToolRegistry, cfg.SSHPool)
	systemHandler := handler.NewSystemHandler(cfg.Version)

	// API 路由组
	api := r.Group("/api")
	{
		// 健康检查
		api.GET("/health", systemHandler.Health)

		// 对话 API
		chat := api.Group("/chat")
		{
			chat.POST("", chatHandler.Chat)
			chat.POST("/stream", chatHandler.StreamChat)
			chat.GET("/sessions", chatHandler.GetSessions)
			chat.GET("/history", chatHandler.GetHistory)
			chat.DELETE("/sessions/:id", chatHandler.DeleteSession)
		}

		// 主机管理 API
		hosts := api.Group("/hosts")
		{
			hosts.GET("", hostHandler.ListHosts)
			hosts.POST("", hostHandler.CreateHost)
			hosts.GET("/:id", hostHandler.GetHost)
			hosts.PUT("/:id", hostHandler.UpdateHost)
			hosts.DELETE("/:id", hostHandler.DeleteHost)
			hosts.POST("/:id/test", hostHandler.TestConnection)
			hosts.POST("/import", hostHandler.ImportHosts)
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
			tools.GET("/:name", toolHandler.GetTool)
			tools.POST("/:name/execute", toolHandler.ExecuteTool)
		}

		// 系统 API
		system := api.Group("/system")
		{
			system.GET("/info", systemHandler.GetSystemInfo)
			system.GET("/config", systemHandler.GetConfig)
			system.PUT("/config", systemHandler.UpdateConfig)
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
