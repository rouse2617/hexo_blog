package handler

import (
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// SystemHandler 系统处理器
type SystemHandler struct {
	startTime time.Time
	version   string
}

// NewSystemHandler 创建系统处理器
func NewSystemHandler(version string) *SystemHandler {
	return &SystemHandler{
		startTime: time.Now(),
		version:   version,
	}
}

// Health 健康检查
// GET /api/health
func (h *SystemHandler) Health(c *gin.Context) {
	uptime := time.Since(h.startTime)

	Success(c, gin.H{
		"status":  "ok",
		"version": h.version,
		"uptime":  uptime.String(),
		"time":    time.Now().Format(time.RFC3339),
	})
}

// GetSystemInfo 获取系统信息
// GET /api/system/info
func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	Success(c, gin.H{
		"version":    h.version,
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"cpus":       runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
		"memory": gin.H{
			"alloc":       memStats.Alloc / 1024 / 1024,       // MB
			"total_alloc": memStats.TotalAlloc / 1024 / 1024, // MB
			"sys":         memStats.Sys / 1024 / 1024,        // MB
		},
		"uptime":     time.Since(h.startTime).String(),
		"start_time": h.startTime.Format(time.RFC3339),
	})
}

// GetConfig 获取系统配置
// GET /api/system/config
func (h *SystemHandler) GetConfig(c *gin.Context) {
	// 返回非敏感配置
	// TODO: 从配置管理器获取
	Success(c, gin.H{
		"server": gin.H{
			"mode": "debug",
		},
		"llm": gin.H{
			"model": "qwen2.5:14b",
		},
		"agent": gin.H{
			"max_loops": 10,
		},
	})
}

// UpdateConfig 更新系统配置
// PUT /api/system/config
func (h *SystemHandler) UpdateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// TODO: 实现配置更新逻辑
	SuccessWithMessage(c, "配置更新功能开发中", nil)
}
