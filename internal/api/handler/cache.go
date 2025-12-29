package handler

import (
	"net/http"

	"ai-ops/internal/cache"
	"ai-ops/internal/service"

	"github.com/gin-gonic/gin"
)

// CacheHandler 缓存处理器
type CacheHandler struct {
	hostService *service.HostService
}

// NewCacheHandler 创建缓存处理器
func NewCacheHandler(hostService *service.HostService) *CacheHandler {
	return &CacheHandler{
		hostService: hostService,
	}
}

// CacheMetricsResponse 缓存指标响应
type CacheMetricsResponse struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	Evictions   int64   `json:"evictions"`
	Size        int64   `json:"size"`
	HitRate     float64 `json:"hit_rate"`
	ErrorCount  int64   `json:"error_count"`
	LastUpdated string  `json:"last_updated"`
}

// GetMetrics 获取缓存指标
// GET /api/cache/metrics
func (h *CacheHandler) GetMetrics(c *gin.Context) {
	metrics := h.hostService.GetCacheMetrics()

	response := CacheMetricsResponse{
		Hits:        metrics.Hits,
		Misses:      metrics.Misses,
		Evictions:   metrics.Evictions,
		Size:        metrics.Size,
		HitRate:     metrics.HitRate,
		ErrorCount:  metrics.ErrorCount,
		LastUpdated: metrics.LastUpdated.Format("2006-01-02 15:04:05"),
	}

	Success(c, response)
}

// ClearCache 清除缓存
// POST /api/cache/clear
func (h *CacheHandler) ClearCache(c *gin.Context) {
	var req struct {
		Type string `json:"type"` // all, hosts, etc.
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有请求体，默认清除所有缓存
		req.Type = "all"
	}

	var err error
	switch req.Type {
	case "all":
		err = h.hostService.ClearAllCache()
	default:
		err = h.hostService.ClearAllCache()
	}

	if err != nil {
		InternalError(c, "清除缓存失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "缓存已清除", gin.H{
		"type": req.Type,
	})
}

// GetCacheStats 获取缓存统计信息（用于仪表板）
// GET /api/cache/stats
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	metrics := h.hostService.GetCacheMetrics()

	totalRequests := metrics.Hits + metrics.Misses
	hitRate := metrics.HitRate * 100

	// 计算性能等级
	performanceGrade := "优秀"
	if hitRate < 50 {
		performanceGrade = "差"
	} else if hitRate < 70 {
		performanceGrade = "一般"
	} else if hitRate < 85 {
		performanceGrade = "良好"
	}

	stats := gin.H{
		"summary": gin.H{
			"total_requests":   totalRequests,
			"hits":             metrics.Hits,
			"misses":           metrics.Misses,
			"evictions":        metrics.Evictions,
			"current_size":     metrics.Size,
			"hit_rate":         hitRate,
			"hit_rate_percent": hitRate,
			"performance_grade": performanceGrade,
		},
		"performance": gin.H{
			"grade":       performanceGrade,
			"hit_rate":    hitRate,
			"error_count": metrics.ErrorCount,
		},
		"recommendations": h.getRecommendations(metrics),
		"last_updated":    metrics.LastUpdated.Format("2006-01-02 15:04:05"),
	}

	Success(c, stats)
}

// getRecommendations 根据缓存指标生成优化建议
func (h *CacheHandler) getRecommendations(metrics cache.Metrics) []string {
	recommendations := []string{}
	hitRate := metrics.HitRate * 100

	if hitRate < 50 {
		recommendations = append(recommendations, "缓存命中率较低，建议增加缓存过期时间")
		recommendations = append(recommendations, "检查缓存键设计，避免频繁失效")
	}

	if metrics.Evictions > 100 {
		recommendations = append(recommendations, "缓存驱逐次数较多，建议增加缓存容量")
	}

	if metrics.ErrorCount > 10 {
		recommendations = append(recommendations, "缓存错误次数较多，请检查缓存配置")
	}

	if metrics.Size == 0 {
		recommendations = append(recommendations, "缓存为空，系统正在冷启动中")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "缓存运行良好，无需优化")
	}

	return recommendations
}

// CacheInfo 缓存信息（用于健康检查）
type CacheInfo struct {
	Enabled bool    `json:"enabled"`
	Size    int64   `json:"size"`
	HitRate float64 `json:"hit_rate"`
	Healthy bool    `json:"healthy"`
}

// GetHealth 获取缓存健康状态
// GET /api/cache/health
func (h *CacheHandler) GetHealth(c *gin.Context) {
	metrics := h.hostService.GetCacheMetrics()

	info := CacheInfo{
		Enabled: true,
		Size:    metrics.Size,
		HitRate: metrics.HitRate,
		Healthy: metrics.ErrorCount < 10,
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"cache":  info,
	})
}
