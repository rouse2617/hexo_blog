package handler

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"ai-ops/internal/repository"
	"ai-ops/internal/security"

	"github.com/gin-gonic/gin"
)

// SystemHandler 系统处理器
type SystemHandler struct {
	startTime  time.Time
	version    string
	policy     *security.PolicyStore
	audit      *security.AuditLogger
	configRepo repository.ConfigRepository
}

// NewSystemHandler 创建系统处理器
func NewSystemHandler(version string, policy *security.PolicyStore, audit *security.AuditLogger, configRepo repository.ConfigRepository) *SystemHandler {
	return &SystemHandler{
		startTime:  time.Now(),
		version:    version,
		policy:     policy,
		audit:      audit,
		configRepo: configRepo,
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
	// 从数据库获取配置
	configs, err := h.configRepo.GetAll()
	if err != nil {
		InternalError(c, "获取配置失败: "+err.Error())
		return
	}

	// 构建配置对象
	result := make(map[string]interface{})
	for key, value := range configs {
		parts := strings.Split(key, ".")
		if len(parts) >= 2 {
			section := parts[0]
			field := strings.Join(parts[1:], ".")
			if result[section] == nil {
				result[section] = make(map[string]interface{})
			}
			result[section].(map[string]interface{})[field] = value
		} else {
			result[key] = value
		}
	}

	// 如果没有配置，返回默认值
	if len(result) == 0 {
		result = map[string]interface{}{
			"server": map[string]interface{}{
				"mode": "debug",
			},
			"llm": map[string]interface{}{
				"model": "qwen2.5:14b",
			},
			"agent": map[string]interface{}{
				"max_loops": "10",
			},
		}
	}

	Success(c, result)
}

// UpdateConfig 更新系统配置
// PUT /api/system/config
func (h *SystemHandler) UpdateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 扁平化配置并保存
	for key, value := range req {
		valueStr := fmt.Sprintf("%v", value)
		
		// 如果是嵌套对象，需要展开
		if nested, ok := value.(map[string]interface{}); ok {
			for nestedKey, nestedValue := range nested {
				configKey := fmt.Sprintf("%s.%s", key, nestedKey)
				if err := h.configRepo.Set(configKey, fmt.Sprintf("%v", nestedValue)); err != nil {
					InternalError(c, "保存配置失败: "+err.Error())
					return
				}
			}
		} else {
			if err := h.configRepo.Set(key, valueStr); err != nil {
				InternalError(c, "保存配置失败: "+err.Error())
				return
			}
		}
	}

	SuccessWithMessage(c, "配置更新成功", nil)
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
