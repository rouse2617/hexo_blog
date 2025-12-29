package cache

import (
	"context"
	"time"
)

// Cache 缓存接口
type Cache interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string, dest interface{}) error

	// Set 设置缓存值
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// DeleteByPattern 根据模式删除缓存（支持通配符）
	DeleteByPattern(ctx context.Context, pattern string) error

	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// Clear 清空所有缓存
	Clear(ctx context.Context) error

	// GetMetrics 获取缓存指标
	GetMetrics() Metrics
}

// Item 缓存项
type Item struct {
	Value      interface{}
	Expiration int64
	CreatedAt  time.Time
}

// IsExpired 检查缓存项是否过期
func (item *Item) IsExpired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

// Metrics 缓存指标
type Metrics struct {
	Hits        int64     // 命中次数
	Misses      int64     // 未命中次数
	Evictions   int64     // 驱逐次数
	Size        int64     // 当前缓存项数量
	TotalSize   int64     // 总缓存大小（字节）
	HitRate     float64   // 命中率
	ErrorCount  int64     // 错误次数
	LastUpdated time.Time // 最后更新时间
}

// CacheConfig 缓存配置
type CacheConfig struct {
	DefaultTTL         time.Duration // 默认过期时间
	CleanupInterval    time.Duration // 清理间隔
	MaxItems           int           // 最大缓存项数量
	EnableMetrics      bool          // 是否启用指标收集
	EnableStatistics   bool          // 是否启用统计
	EnableEviction     bool          // 是否启用驱逐策略
	EvictionPolicy     string        // 驱逐策略: lru, fifo, lfu
}

// DefaultCacheConfig 默认缓存配置
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		DefaultTTL:       5 * time.Minute,
		CleanupInterval:  1 * time.Minute,
		MaxItems:         1000,
		EnableMetrics:    true,
		EnableStatistics: true,
		EnableEviction:   true,
		EvictionPolicy:   "lru",
	}
}

// CacheKey 缓存键生成器
type CacheKey struct {
	prefix string
}

// NewCacheKey 创建缓存键生成器
func NewCacheKey(prefix string) *CacheKey {
	return &CacheKey{prefix: prefix}
}

// Build 构建缓存键
func (ck *CacheKey) Build(parts ...string) string {
	key := ck.prefix
	for _, part := range parts {
		if key != "" {
			key += ":"
		}
		key += part
	}
	return key
}

// Pattern 构建通配符模式
func (ck *CacheKey) Pattern(parts ...string) string {
	key := ck.Build(parts...)
	return key + "*"
}
