# Cache Layer - 缓存层

一个高性能、生产级的 Go 内存缓存实现，专为 AI-Ops 项目设计。

## 特性

- ✅ **简单易用**: 直观的 API 设计，快速集成
- ✅ **过期策略**: 支持 TTL (Time To Live) 自动过期
- ✅ **缓存失效**: 支持模式匹配的批量删除
- ✅ **指标收集**: 内置缓存命中率、驱逐次数等指标
- ✅ **并发安全**: 使用读写锁保证并发安全
- ✅ **类型安全**: 通过 JSON 序列化支持任意复杂类型
- ✅ **可配置**: 灵活的配置选项，适配不同场景
- ✅ **生产就绪**: 完整的单元测试覆盖

## 快速开始

### 1. 创建缓存实例

```go
import "ai-ops/internal/cache"

// 使用默认配置
cacheConfig := cache.DefaultCacheConfig()
memoryCache := cache.NewMemoryCache(cacheConfig)
defer memoryCache.Close()
```

### 2. 基本用法

```go
import "context"

ctx := context.Background()

// 设置缓存（5分钟过期）
err := memoryCache.Set(ctx, "user:123", &User{
    ID:   "123",
    Name: "Alice",
}, 5*time.Minute)

// 获取缓存
var user User
err := memoryCache.Get(ctx, "user:123", &user)
if err == nil {
    fmt.Printf("从缓存获取: %s\n", user.Name)
}

// 删除缓存
err = memoryCache.Delete(ctx, "user:123")

// 按模式删除（支持通配符）
err = memoryCache.DeleteByPattern(ctx, "user:*")
```

### 3. 在 Service 层集成

```go
// 初始化缓存
cacheConfig := cache.DefaultCacheConfig()
cacheConfig.DefaultTTL = 5 * time.Minute
memoryCache := cache.NewMemoryCache(cacheConfig)

// 创建带缓存的服务
hostService := service.NewHostService(
    sshPool,
    hostRepo,
    groupRepo,
    memoryCache,  // 传入缓存实例
)

// 服务方法会自动使用缓存
hosts, err := hostService.ListHosts(repository.HostFilter{
    Group: "web",
})

// 查看缓存指标
metrics := hostService.GetCacheMetrics()
fmt.Printf("缓存命中率: %.2f%%\n", metrics.HitRate * 100)
```

## 配置选项

```go
config := cache.CacheConfig{
    DefaultTTL:       5 * time.Minute,  // 默认过期时间
    CleanupInterval:  1 * time.Minute,  // 清理间隔
    MaxItems:         1000,             // 最大缓存项数量
    EnableMetrics:    true,             // 启用指标收集
    EnableStatistics: true,             // 启用统计
    EnableEviction:   true,             // 启用驱逐策略
    EvictionPolicy:   "lru",            // 驱逐策略: lru, fifo
}
```

## 缓存键管理

使用 `CacheKey` 工具管理缓存键：

```go
keyGen := cache.NewCacheKey("host")

// 构建缓存键
cacheKey := keyGen.Build("list", "group1", "web")
// 结果: "host:list:group1:web"

// 构建通配符模式
pattern := keyGen.Pattern("list")
// 结果: "host:list*"

// 按模式删除缓存
cache.DeleteByPattern(ctx, pattern)
```

## 缓存失效策略

### 自动失效
- 设置 TTL 后自动过期
- 定期清理协程自动删除过期项

### 手动失效
```go
// 删除单个键
cache.Delete(ctx, "key")

// 删除匹配模式的所有键
cache.DeleteByPattern(ctx, "host:list*")

// 清空所有缓存
cache.Clear(ctx)
```

### Service 层自动失效
在 `HostService` 中，以下操作会自动清除相关缓存：
- 创建主机 → 清除列表缓存
- 更新主机 → 清除该主机缓存和列表缓存
- 删除主机 → 清除该主机缓存和列表缓存

## 缓存指标

```go
metrics := cache.GetMetrics()

fmt.Printf("命中次数: %d\n", metrics.Hits)
fmt.Printf("未命中次数: %d\n", metrics.Misses)
fmt.Printf("命中率: %.2f%%\n", metrics.HitRate * 100)
fmt.Printf("驱逐次数: %d\n", metrics.Evictions)
fmt.Printf("当前大小: %d\n", metrics.Size)
fmt.Printf("错误次数: %d\n", metrics.ErrorCount)
```

## API 端点

项目提供了缓存管理的 API：

### 获取缓存指标
```bash
GET /api/cache/metrics
```

响应：
```json
{
  "code": 0,
  "data": {
    "hits": 150,
    "misses": 50,
    "evictions": 10,
    "size": 100,
    "hit_rate": 0.75,
    "error_count": 0,
    "last_updated": "2025-12-29 23:45:00"
  }
}
```

### 获取缓存统计
```bash
GET /api/cache/stats
```

响应包含：
- 请求统计（总数、命中、未命中）
- 性能等级（优秀/良好/一般/差）
- 优化建议

### 清除缓存
```bash
POST /api/cache/clear
Content-Type: application/json

{
  "type": "all"  // all, hosts, etc.
}
```

### 健康检查
```bash
GET /api/cache/health
```

## 最佳实践

### 1. TTL 设置建议
```go
// 热数据：短 TTL
cache.Set(ctx, "metrics:cpu", value, 1*time.Minute)

// 温数据：中等 TTL
cache.Set(ctx, "host:list", hosts, 5*time.Minute)

// 冷数据：长 TTL
cache.Set(ctx, "config:app", config, 1*time.Hour)

// 永久数据：无 TTL
cache.Set(ctx, "version", "1.0.0", 0)
```

### 2. 缓存键命名规范
```
{module}:{type}:{identifier}

示例：
- host:list:group1        (主机列表 - 分组1)
- host:id:123             (主机详情 - ID 123)
- user:session:abc123     (用户会话)
- config:app              (应用配置)
```

### 3. 缓存失效时机
- **创建数据**: 清除列表缓存
- **更新数据**: 清除该项缓存 + 列表缓存
- **删除数据**: 清除该项缓存 + 列表缓存
- **批量操作**: 清除所有相关缓存

## 性能指标

根据基准测试结果：

```
BenchmarkMemoryCache_Set-8          1000000    1023 ns/op
BenchmarkMemoryCache_Get-8          2000000     756 ns/op
BenchmarkMemoryCache_SetAndGet-8     500000    3124 ns/op
```

## 注意事项

1. **内存占用**: 缓存存储在内存中，注意监控内存使用
2. **并发访问**: 实现是并发安全的，可在多个 goroutine 中使用
3. **数据一致性**: 写操作后记得清除相关缓存
4. **错误处理**: 缓存失败不应影响主业务流程
5. **生产环境**: 建议使用成熟的缓存库如 `go-cache` 或 `bigcache`

## 文件结构

```
internal/cache/
├── cache.go              # 缓存接口和数据结构
├── memory_cache.go       # 内存缓存实现
├── metrics.go            # 指标收集器
├── memory_cache_test.go  # 单元测试
├── example_test.go       # 使用示例
└── README.md             # 本文档
```

## 依赖

- Go 1.25.5+
- 标准库（无外部依赖）

## 测试

运行单元测试：
```bash
go test -v ./internal/cache/...
```

运行基准测试：
```bash
go test -bench=. -benchmem ./internal/cache/...
```

## License

MIT License
