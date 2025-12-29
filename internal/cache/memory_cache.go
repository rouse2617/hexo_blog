package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MemoryCache 内存缓存实现
type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]*Item
	config   CacheConfig
	metrics  Metrics
	stopChan chan struct{}
	lruList  *LRUList // LRU 淘汰链表
}

// LRUList LRU 链表节点
type LRUList struct {
	head *LRUNode
	tail *LRUNode
	mu   sync.Mutex
}

// LRUNode LRU 链表节点
type LRUNode struct {
	key  string
	prev *LRUNode
	next *LRUNode
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(config CacheConfig) *MemoryCache {
	cache := &MemoryCache{
		items:    make(map[string]*Item),
		config:   config,
		stopChan: make(chan struct{}),
		lruList:  NewLRUList(),
		metrics: Metrics{
			LastUpdated: time.Now(),
		},
	}

	// 启动清理协程
	if config.CleanupInterval > 0 {
		go cache.cleanupLoop()
	}

	return cache
}

// NewLRUList 创建 LRU 链表
func NewLRUList() *LRUList {
	head := &LRUNode{}
	tail := &LRUNode{}
	head.next = tail
	tail.prev = head
	return &LRUList{head: head, tail: tail}
}

// Add 添加节点到链表头部（最新使用）
func (l *LRUList) Add(key string) *LRUNode {
	l.mu.Lock()
	defer l.mu.Unlock()

	node := &LRUNode{key: key}
	node.next = l.head.next
	node.prev = l.head
	l.head.next.prev = node
	l.head.next = node

	return node
}

// MoveToFront 移动节点到链表头部
func (l *LRUList) MoveToFront(node *LRUNode) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node.prev == nil || node.next == nil {
		return
	}

	// 移除节点
	node.prev.next = node.next
	node.next.prev = node.prev

	// 添加到头部
	node.next = l.head.next
	node.prev = l.head
	l.head.next.prev = node
	l.head.next = node
}

// Remove 移除节点
func (l *LRUList) Remove(node *LRUNode) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node.prev == nil || node.next == nil {
		return
	}

	node.prev.next = node.next
	node.next.prev = node.prev
	node.prev = nil
	node.next = nil
}

// RemoveTail 移除链表尾部节点（最久未使用）
func (l *LRUList) RemoveTail() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.tail.prev == l.head {
		return ""
	}

	node := l.tail.prev
	node.prev.next = l.tail
	l.tail.prev = node.prev
	node.prev = nil
	node.next = nil

	return node.key
}

// Get 获取缓存值
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		c.recordMiss()
		return fmt.Errorf("cache key not found: %s", key)
	}

	if item.IsExpired() {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		c.recordMiss()
		c.recordEviction()
		return fmt.Errorf("cache key expired: %s", key)
	}

	// 序列化到目标对象
	data, err := json.Marshal(item.Value)
	if err != nil {
		c.recordError()
		return fmt.Errorf("failed to marshal cached value: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		c.recordError()
		return fmt.Errorf("failed to unmarshal to destination: %w", err)
	}

	c.recordHit()
	return nil
}

// Set 设置缓存值
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	item := &Item{
		Value:      value,
		Expiration: expiration,
		CreatedAt:  time.Now(),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否超过最大缓存项数量
	if c.config.MaxItems > 0 && len(c.items) >= c.config.MaxItems {
		if _, exists := c.items[key]; !exists {
			c.evictLRU()
		}
	}

	c.items[key] = item
	c.updateMetrics()
	return nil
}

// Delete 删除缓存
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.items[key]; exists {
		delete(c.items, key)
		c.updateMetrics()
	}

	return nil
}

// DeleteByPattern 根据模式删除缓存
func (c *MemoryCache) DeleteByPattern(ctx context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	keysToDelete := make([]string, 0)
	for key := range c.items {
		if matchPattern(key, pattern) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(c.items, key)
	}

	c.updateMetrics()
	return nil
}

// Exists 检查缓存是否存在
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		return false, nil
	}

	if item.IsExpired() {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return false, nil
	}

	return true, nil
}

// Clear 清空所有缓存
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*Item)
	c.metrics = Metrics{
		LastUpdated: time.Now(),
	}
	return nil
}

// GetMetrics 获取缓存指标
func (c *MemoryCache) GetMetrics() Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := c.metrics
	metrics.Size = int64(len(c.items))
	metrics.HitRate = c.calculateHitRate()

	return metrics
}

// evictLRU 驱逐最久未使用的缓存项
func (c *MemoryCache) evictLRU() {
	key := c.lruList.RemoveTail()
	if key != "" {
		delete(c.items, key)
		c.recordEviction()
	}
}

// cleanupLoop 定期清理过期缓存
func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopChan:
			return
		}
	}
}

// cleanup 清理过期缓存
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	keysToDelete := make([]string, 0)
	now := time.Now().UnixNano()

	for key, item := range c.items {
		if item.Expiration > 0 && item.Expiration < now {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(c.items, key)
		c.recordEviction()
	}

	c.updateMetrics()
}

// matchPattern 匹配通配符模式
func matchPattern(key, pattern string) bool {
	if !strings.Contains(pattern, "*") {
		return key == pattern
	}

	// 简单的通配符匹配实现
	patternParts := strings.Split(pattern, "*")
	if len(patternParts) == 0 {
		return true
	}

	idx := 0
	for _, part := range patternParts {
		if part == "" {
			continue
		}
		newIdx := strings.Index(key[idx:], part)
		if newIdx == -1 {
			return false
		}
		idx += newIdx + len(part)
	}

	return true
}

// recordHit 记录缓存命中
func (c *MemoryCache) recordHit() {
	if c.config.EnableMetrics {
		c.metrics.Hits++
		c.metrics.LastUpdated = time.Now()
	}
}

// recordMiss 记录缓存未命中
func (c *MemoryCache) recordMiss() {
	if c.config.EnableMetrics {
		c.metrics.Misses++
		c.metrics.LastUpdated = time.Now()
	}
}

// recordEviction 记录缓存驱逐
func (c *MemoryCache) recordEviction() {
	if c.config.EnableMetrics {
		c.metrics.Evictions++
		c.metrics.LastUpdated = time.Now()
	}
}

// recordError 记录错误
func (c *MemoryCache) recordError() {
	if c.config.EnableMetrics {
		c.metrics.ErrorCount++
		c.metrics.LastUpdated = time.Now()
	}
}

// updateMetrics 更新指标
func (c *MemoryCache) updateMetrics() {
	if c.config.EnableMetrics {
		c.metrics.Size = int64(len(c.items))
		c.metrics.LastUpdated = time.Now()
	}
}

// calculateHitRate 计算命中率
func (c *MemoryCache) calculateHitRate() float64 {
	total := c.metrics.Hits + c.metrics.Misses
	if total == 0 {
		return 0
	}
	return float64(c.metrics.Hits) / float64(total)
}

// Stop 停止缓存清理
func (c *MemoryCache) Stop() {
	close(c.stopChan)
}

// Close 关闭缓存
func (c *MemoryCache) Close() error {
	c.Stop()
	return nil
}
