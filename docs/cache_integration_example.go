package main

import (
	"context"
	"log"
	"time"

	"ai-ops/internal/cache"
	"ai-ops/internal/service"
)

// 这是一个示例文件，展示如何在主程序中集成缓存层
// 实际使用时，请根据项目结构调整

func main() {
	// 示例：初始化应用
	// if err := run(); err != nil {
	//     log.Fatal(err)
	// }
}

func run() error {
	// 1. 初始化数据库
	// db, err := gorm.Open(sqlite.Open("aiops.db"), &gorm.Config{})
	// if err != nil {
	//     return fmt.Errorf("failed to connect database: %w", err)
	// }

	// 2. 初始化 Repository
	// hostRepo := repository.NewHostRepository(db)
	// groupRepo := repository.NewGroupRepository(db)

	// 3. 初始化 SSH Pool
	// sshPool := ssh.NewPool()

	// 4. 初始化缓存（新增）
	cacheConfig := cache.DefaultCacheConfig()
	cacheConfig.DefaultTTL = 5 * time.Minute
	cacheConfig.CleanupInterval = 1 * time.Minute
	cacheConfig.MaxItems = 1000
	cacheConfig.EnableMetrics = true
	memoryCache := cache.NewMemoryCache(cacheConfig)
	defer memoryCache.Close()

	log.Println("缓存层已初始化")

	// 5. 初始化 Service 层（传入缓存）
	// hostService := service.NewHostService(
	//     sshPool,
	//     hostRepo,
	//     groupRepo,
	//     memoryCache,  // 传入缓存实例
	// )

	// 6. 初始化 Handler
	// hostHandler := handler.NewHostHandler(sshPool, hostRepo, groupRepo)
	// cacheHandler := handler.NewCacheHandler(hostService)

	// 7. 注册路由
	// router := gin.Default()
	//
	// // 主机相关路由
	// hostGroup := router.Group("/api/hosts")
	// {
	//     hostGroup.GET("", hostHandler.ListHosts)
	//     hostGroup.POST("", hostHandler.CreateHost)
	//     hostGroup.GET("/:id", hostHandler.GetHost)
	//     hostGroup.PUT("/:id", hostHandler.UpdateHost)
	//     hostGroup.DELETE("/:id", hostHandler.DeleteHost)
	// }
	//
	// // 缓存相关路由（新增）
	// cacheGroup := router.Group("/api/cache")
	// {
	//     cacheGroup.GET("/metrics", cacheHandler.GetMetrics)
	//     cacheGroup.GET("/stats", cacheHandler.GetCacheStats)
	//     cacheGroup.POST("/clear", cacheHandler.ClearCache)
	//     cacheGroup.GET("/health", cacheHandler.GetHealth)
	// }

	// 8. 启动服务器
	// return router.Run(":8080")

	return nil
}

// 示例：手动使用缓存
func exampleManualCacheUsage(cache *cache.MemoryCache) {
	ctx := context.Background()

	// 设置缓存
	type UserInfo struct {
		ID   string
		Name string
	}

	user := UserInfo{
		ID:   "123",
		Name: "Alice",
	}

	err := cache.Set(ctx, "user:123", user, 10*time.Minute)
	if err != nil {
		log.Printf("设置缓存失败: %v", err)
	}

	// 获取缓存
	var cachedUser UserInfo
	err = cache.Get(ctx, "user:123", &cachedUser)
	if err == nil {
		log.Printf("从缓存获取用户: %s", cachedUser.Name)
	}

	// 查看缓存指标
	metrics := cache.GetMetrics()
	log.Printf("缓存命中率: %.2f%%", metrics.HitRate*100)
}

// 示例：缓存失效策略
func exampleCacheInvalidation(memoryCache *cache.MemoryCache, hostService *service.HostService) {
	ctx := context.Background()

	// 场景1: 更新主机后清除缓存
	// 1. 更新数据库
	// hostService.UpdateHost("host123", hostInfo)

	// 2. 缓存已自动清除，下次获取会从数据库重新加载

	// 场景2: 手动按模式清除缓存
	keyGen := cache.NewCacheKey("host")

	// 清除所有主机列表缓存
	_ = memoryCache.DeleteByPattern(ctx, keyGen.Pattern("list"))

	// 清除特定主机的所有缓存
	_ = memoryCache.DeleteByPattern(ctx, keyGen.Build("*", "host123"))
}

// 示例：监控缓存性能
func exampleCacheMonitoring(cache *cache.MemoryCache) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		metrics := cache.GetMetrics()

		// 计算命中率
		hitRate := metrics.HitRate * 100

		log.Printf("缓存统计:")
		log.Printf("  命中率: %.2f%%", hitRate)
		log.Printf("  命中次数: %d", metrics.Hits)
		log.Printf("  未命中次数: %d", metrics.Misses)
		log.Printf("  当前大小: %d", metrics.Size)
		log.Printf("  驱逐次数: %d", metrics.Evictions)

		// 根据命中率调整策略
		if hitRate < 50 {
			log.Println("警告: 缓存命中率较低，建议增加 TTL")
		}

		if metrics.Size > 900 {
			log.Println("警告: 缓存接近容量上限")
		}
	}
}

// 示例：在不同场景下的 TTL 设置
func exampleCacheTTL(memoryCache *cache.MemoryCache) {
	ctx := context.Background()

	// 1. 实时数据（1分钟）
	_ = memoryCache.Set(ctx, "metrics:cpu", 45.2, 1*time.Minute)

	// 2. 频繁访问的列表（5分钟）
	_ = memoryCache.Set(ctx, "host:list", []string{"host1", "host2"}, 5*time.Minute)

	// 3. 配置数据（1小时）
	_ = memoryCache.Set(ctx, "config:app", map[string]string{"env": "prod"}, 1*time.Hour)

	// 4. 永久数据（无过期）
	_ = memoryCache.Set(ctx, "version", "1.0.0", 0)

	// 5. 使用默认 TTL
	defaultConfig := cache.DefaultCacheConfig()
	_ = memoryCache.Set(ctx, "default:key", "value", defaultConfig.DefaultTTL)
}

// 示例：缓存预热
func exampleCacheWarmup(memoryCache *cache.MemoryCache, hostService *service.HostService) {
	ctx := context.Background()

	log.Println("开始缓存预热...")

	// 预加载常用数据
	// 1. 加载所有主机列表
	// hosts, _ := hostService.ListHosts(repository.HostFilter{})
	// _ = memoryCache.Set(ctx, "host:all", hosts, 10*time.Minute)

	// 2. 加载常用分组的主机
	// webHosts, _ := hostService.ListHosts(repository.HostFilter{Group: "web"})
	// _ = memoryCache.Set(ctx, "host:list:web", webHosts, 10*time.Minute)

	// 3. 加载配置信息
	// _ = memoryCache.Set(ctx, "config:system", systemConfig, 30*time.Minute)

	log.Println("缓存预热完成")
}
