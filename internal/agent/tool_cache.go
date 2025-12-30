package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// ToolCache 工具调用缓存
type ToolCache struct {
	cache   map[string]*CacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
	cancel  context.CancelFunc // 用于关闭 cleanup goroutine
	stopped sync.Once         // 确保 Close 只执行一次
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Result    string
	Error     string
	CreatedAt time.Time
	HitCount  int
}

// NewToolCache 创建工具缓存
func NewToolCache(ttl time.Duration) *ToolCache {
	if ttl == 0 {
		ttl = 30 * time.Second // 默认 30 秒过期
	}

	ctx, cancel := context.WithCancel(context.Background())

	tc := &ToolCache{
		cache:  make(map[string]*CacheEntry),
		ttl:    ttl,
		cancel: cancel,
	}
	// 启动清理协程
	go tc.cleanupLoop(ctx)
	return tc
}

// Get 获取缓存
func (c *ToolCache) Get(toolName string, params map[string]interface{}) (string, string, bool) {
	key := c.makeKey(toolName, params)

	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if !ok {
		return "", "", false
	}

	// 检查是否过期
	if time.Since(entry.CreatedAt) > c.ttl {
		c.mu.Lock()
		delete(c.cache, key)
		c.mu.Unlock()
		return "", "", false
	}

	// 更新命中计数
	c.mu.Lock()
	entry.HitCount++
	c.mu.Unlock()

	return entry.Result, entry.Error, true
}

// Set 设置缓存
func (c *ToolCache) Set(toolName string, params map[string]interface{}, result, errMsg string) {
	// 只缓存只读工具的结果
	if !isReadOnlyTool(toolName) {
		return
	}

	key := c.makeKey(toolName, params)

	c.mu.Lock()
	c.cache[key] = &CacheEntry{
		Result:    result,
		Error:     errMsg,
		CreatedAt: time.Now(),
		HitCount:  0,
	}
	c.mu.Unlock()
}

// makeKey 生成缓存 key
func (c *ToolCache) makeKey(toolName string, params map[string]interface{}) string {
	data, _ := json.Marshal(map[string]interface{}{
		"tool":   toolName,
		"params": params,
	})
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:16])
}

// cleanupLoop 定期清理过期缓存
func (c *ToolCache) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-ctx.Done():
			// context 取消，退出清理循环
			return
		}
	}
}

// cleanup 清理过期条目
func (c *ToolCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.cache {
		if now.Sub(entry.CreatedAt) > c.ttl {
			delete(c.cache, key)
		}
	}
}

// Clear 清空缓存
func (c *ToolCache) Clear() {
	c.mu.Lock()
	c.cache = make(map[string]*CacheEntry)
	c.mu.Unlock()
}

// Stats 获取缓存统计
func (c *ToolCache) Stats() (size int, totalHits int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	size = len(c.cache)
	for _, entry := range c.cache {
		totalHits += entry.HitCount
	}
	return
}

// Close 关闭缓存，停止清理 goroutine
func (c *ToolCache) Close() {
	c.stopped.Do(func() {
		c.cancel()
		c.Clear()
	})
}

// isReadOnlyTool 判断是否为只读工具（可缓存）
func isReadOnlyTool(toolName string) bool {
	readOnlyTools := map[string]bool{
		"check_cpu":      true,
		"check_memory":   true,
		"check_disk":     true,
		"check_network":  true,
		"check_process":  true,
		"list_hosts":     true,
		"query_log":      true,
		"check_inode":    true,
		"check_io":       true,
		"check_load":     true,
		"check_uptime":   true,
		"check_users":    true,
		"check_services": true,
	}
	return readOnlyTools[toolName]
}
