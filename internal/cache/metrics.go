package cache

import (
	"encoding/json"
	"time"
)

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// RecordHit 记录命中
	RecordHit(key string)

	// RecordMiss 记录未命中
	RecordMiss(key string)

	// RecordEviction 记录驱逐
	RecordEviction(key string)

	// GetSnapshot 获取快照
	GetSnapshot() MetricsSnapshot

	// Reset 重置指标
	Reset()
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	Timestamp      time.Time              `json:"timestamp"`
	TotalHits      int64                  `json:"total_hits"`
	TotalMisses    int64                  `json:"total_misses"`
	TotalEvictions int64                  `json:"total_evictions"`
	HitRate        float64                `json:"hit_rate"`
	CacheSize      int                    `json:"cache_size"`
	TopKeys        []KeyStats             `json:"top_keys,omitempty"`
	Errors         int64                  `json:"errors"`
	KeyStats       map[string]KeyStats    `json:"key_stats,omitempty"`
}

// KeyStats 键级统计
type KeyStats struct {
	Key         string    `json:"key"`
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	LastAccess  time.Time `json:"last_access"`
	CreatedAt   time.Time `json:"created_at"`
	TTL         int64     `json:"ttl"`
}

// CacheStatistics 缓存统计信息
type CacheStatistics struct {
	metrics        *Metrics
	keyStats       map[string]*keyCounter
	topKeysSize    int
	lastResetTime  time.Time
}

// keyCounter 键计数器
type keyCounter struct {
	hits       int64
	misses     int64
	lastAccess time.Time
}

// NewCacheStatistics 创建缓存统计
func NewCacheStatistics(topKeysSize int) *CacheStatistics {
	return &CacheStatistics{
		metrics:       &Metrics{},
		keyStats:      make(map[string]*keyCounter),
		topKeysSize:   topKeysSize,
		lastResetTime: time.Now(),
	}
}

// GetMetrics 获取指标
func (cs *CacheStatistics) GetMetrics() Metrics {
	return *cs.metrics
}

// RecordHit 记录命中
func (cs *CacheStatistics) RecordHit(key string) {
	cs.metrics.Hits++
	cs.metrics.LastUpdated = time.Now()

	if counter, exists := cs.keyStats[key]; exists {
		counter.hits++
		counter.lastAccess = time.Now()
	} else {
		cs.keyStats[key] = &keyCounter{
			hits:       1,
			lastAccess: time.Now(),
		}
	}
}

// RecordMiss 记录未命中
func (cs *CacheStatistics) RecordMiss(key string) {
	cs.metrics.Misses++
	cs.metrics.LastUpdated = time.Now()

	if counter, exists := cs.keyStats[key]; exists {
		counter.misses++
		counter.lastAccess = time.Now()
	} else {
		cs.keyStats[key] = &keyCounter{
			misses:     1,
			lastAccess: time.Now(),
		}
	}
}

// RecordEviction 记录驱逐
func (cs *CacheStatistics) RecordEviction(key string) {
	cs.metrics.Evictions++
	cs.metrics.LastUpdated = time.Now()
	delete(cs.keyStats, key)
}

// RecordError 记录错误
func (cs *CacheStatistics) RecordError() {
	cs.metrics.ErrorCount++
	cs.metrics.LastUpdated = time.Now()
}

// GetSnapshot 获取快照
func (cs *CacheStatistics) GetSnapshot() MetricsSnapshot {
	snapshot := MetricsSnapshot{
		Timestamp:      time.Now(),
		TotalHits:      cs.metrics.Hits,
		TotalMisses:    cs.metrics.Misses,
		TotalEvictions: cs.metrics.Evictions,
		HitRate:        cs.calculateHitRate(),
		CacheSize:      int(cs.metrics.Size),
		Errors:         cs.metrics.ErrorCount,
	}

	// 获取 Top Keys
	if cs.topKeysSize > 0 {
		snapshot.TopKeys = cs.getTopKeys(cs.topKeysSize)
	}

	// 获取所有键统计
	snapshot.KeyStats = cs.getAllKeyStats()

	return snapshot
}

// getTopKeys 获取访问频率最高的键
func (cs *CacheStatistics) getTopKeys(limit int) []KeyStats {
	stats := make([]KeyStats, 0, len(cs.keyStats))
	for key, counter := range cs.keyStats {
		stats = append(stats, KeyStats{
			Key:        key,
			Hits:       counter.hits,
			Misses:     counter.misses,
			LastAccess: counter.lastAccess,
		})
	}

	// 简单排序（按命中次数）
	for i := 0; i < len(stats)-1; i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[i].Hits < stats[j].Hits {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	if len(stats) > limit {
		stats = stats[:limit]
	}

	return stats
}

// getAllKeyStats 获取所有键统计
func (cs *CacheStatistics) getAllKeyStats() map[string]KeyStats {
	result := make(map[string]KeyStats, len(cs.keyStats))
	for key, counter := range cs.keyStats {
		result[key] = KeyStats{
			Key:        key,
			Hits:       counter.hits,
			Misses:     counter.misses,
			LastAccess: counter.lastAccess,
		}
	}
	return result
}

// calculateHitRate 计算命中率
func (cs *CacheStatistics) calculateHitRate() float64 {
	total := cs.metrics.Hits + cs.metrics.Misses
	if total == 0 {
		return 0
	}
	return float64(cs.metrics.Hits) / float64(total)
}

// Reset 重置统计
func (cs *CacheStatistics) Reset() {
	cs.metrics = &Metrics{}
	cs.keyStats = make(map[string]*keyCounter)
	cs.lastResetTime = time.Now()
}

// ToJSON 转换为 JSON
func (ms *MetricsSnapshot) ToJSON() (string, error) {
	data, err := json.MarshalIndent(ms, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// String 实现 Stringer 接口
func (ms *MetricsSnapshot) String() string {
	json, err := ms.ToJSON()
	if err != nil {
		return "{}"
	}
	return json
}
