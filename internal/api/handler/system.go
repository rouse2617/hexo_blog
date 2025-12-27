package handler

import (
	"net/http"
	"runtime"
	"strconv"
	"time"

	"ai-ops/internal/security"

	"github.com/gin-gonic/gin"
)

// SystemHandler 系统处理器
type SystemHandler struct {
	startTime time.Time
	version   string
	policy    *security.PolicyStore
	audit     *security.AuditLogger
}

// NewSystemHandler 创建系统处理器
func NewSystemHandler(version string, policy *security.PolicyStore, audit *security.AuditLogger) *SystemHandler {
	return &SystemHandler{
		startTime: time.Now(),
		version:   version,
		policy:    policy,
		audit:     audit,
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
			"alloc":       memStats.Alloc / 1024 / 1024,      // MB
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

// GetCommandPolicy 获取命令白名单策略
// GET /api/system/command-policy
func (h *SystemHandler) GetCommandPolicy(c *gin.Context) {
	if h.policy == nil {
		InternalError(c, "command policy store 未初始化")
		return
	}
	Success(c, h.policy.Get())
}

// UpdateCommandPolicy 更新命令白名单策略
// PUT /api/system/command-policy
func (h *SystemHandler) UpdateCommandPolicy(c *gin.Context) {
	if h.policy == nil {
		InternalError(c, "command policy store 未初始化")
		return
	}
	var req security.CommandPolicy
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}
	if err := h.policy.Set(req); err != nil {
		c.JSON(http.StatusOK, Response{Code: CodeInternalError, Message: "保存失败: " + err.Error()})
		return
	}
	SuccessWithMessage(c, "保存成功", h.policy.Get())
}

// GetCommandAudit 获取命令审计日志（最近 N 条）
// GET /api/system/audit/commands?limit=200
func (h *SystemHandler) GetCommandAudit(c *gin.Context) {
	if h.audit == nil {
		InternalError(c, "audit logger 未初始化")
		return
	}
	limitStr := c.Query("limit")
	limit := 200
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	events, err := h.audit.ReadLastN(limit)
	if err != nil {
		InternalError(c, "读取审计失败: "+err.Error())
		return
	}
	Success(c, gin.H{"events": events})
}
