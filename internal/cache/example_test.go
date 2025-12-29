package cache_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"ai-ops/internal/cache"
	"ai-ops/internal/model"

	"gorm.io/gorm"
)

// Example_cacheUsage 展示缓存的基本用法
func Example_cacheUsage() {
	// 1. 创建缓存实例
	cacheConfig := cache.DefaultCacheConfig()
	cacheConfig.DefaultTTL = 5 * time.Minute
	cacheConfig.MaxItems = 1000
	memoryCache := cache.NewMemoryCache(cacheConfig)
	defer memoryCache.Close()

	// 2. 使用缓存
	ctx := context.Background()

	// 设置缓存
	err := memoryCache.Set(ctx, "user:123", &model.Host{
		ID:   "123",
		Name: "Test Server",
		IP:   "192.168.1.1",
		Port: 22,
		User: "root",
	}, 10*time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	// 获取缓存
	var host model.Host
	err = memoryCache.Get(ctx, "user:123", &host)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Host: %s (%s)\n", host.Name, host.IP)

	// 3. 查看缓存指标
	metrics := memoryCache.GetMetrics()
	fmt.Printf("Hit Rate: %.2f%%\n", metrics.HitRate*100)
}

// Example_serviceWithCache 展示如何在 Service 层使用缓存
func Example_serviceWithCache() {
	// 这个示例展示了如何在真实项目中集成缓存

	// 1. 初始化缓存
	cacheConfig := cache.DefaultCacheConfig()
	cacheConfig.DefaultTTL = 5 * time.Minute
	cacheConfig.CleanupInterval = 1 * time.Minute
	cacheConfig.EnableMetrics = true
	memoryCache := cache.NewMemoryCache(cacheConfig)
	defer memoryCache.Close()

	// 2. 初始化 Repository（假设已有数据库连接）
	// var db *gorm.DB
	// hostRepo := repository.NewHostRepository(db)

	// 3. 创建带缓存的服务层
	// hostService := service.NewHostService(nil, hostRepo, nil, memoryCache)

	// 4. 使用服务（自动享受缓存加速）
	// filter := repository.HostFilter{Group: "web"}
	// hosts, err := hostService.ListHosts(filter)
	// if err != nil {
	//     log.Fatal(err)
	// }

	// fmt.Printf("Found %d hosts\n", len(hosts))

	// 5. 查看缓存指标
	// metrics := hostService.GetCacheMetrics()
	// fmt.Printf("Cache Hit Rate: %.2f%%\n", metrics.HitRate*100)
}

// Example_cacheInvalidation 展示缓存失效策略
func Example_cacheInvalidation() {
	ctx := context.Background()
	memoryCache := cache.NewMemoryCache(cache.DefaultCacheConfig())
	defer memoryCache.Close()

	keyGen := cache.NewCacheKey("host")

	// 1. 设置多个相关缓存
	_ = memoryCache.Set(ctx, keyGen.Build("list", "web"), []string{"host1", "host2"}, 5*time.Minute)
	_ = memoryCache.Set(ctx, keyGen.Build("list", "db"), []string{"host3"}, 5*time.Minute)
	_ = memoryCache.Set(ctx, keyGen.Build("id", "host1"), "host1详情", 10*time.Minute)

	// 2. 当更新主机时，清除相关缓存
	// 按模式删除所有主机列表缓存
	_ = memoryCache.DeleteByPattern(ctx, keyGen.Pattern("list"))

	// 按模式删除特定主机的缓存
	_ = memoryCache.DeleteByPattern(ctx, keyGen.Build("*", "host1"))

	// 3. 或者清除所有缓存
	_ = memoryCache.Clear(ctx)
}

// Example_cacheWithTTL 展示不同场景的 TTL 设置
func Example_cacheWithTTL() {
	ctx := context.Background()
	memoryCache := cache.NewMemoryCache(cache.DefaultCacheConfig())
	defer memoryCache.Close()

	// 不同数据类型使用不同的 TTL

	// 热数据：短 TTL（1分钟）
	_ = memoryCache.Set(ctx, "metrics:cpu", 45.2, 1*time.Minute)

	// 温数据：中等 TTL（5分钟）
	_ = memoryCache.Set(ctx, "host:list", []string{"host1", "host2"}, 5*time.Minute)

	// 冷数据：长 TTL（1小时）
	_ = memoryCache.Set(ctx, "config:app", map[string]string{"env": "prod"}, 1*time.Hour)

	// 永久数据：无 TTL
	_ = memoryCache.Set(ctx, "version", "1.0.0", 0)
}

// MockHostRepository 模拟主机仓库（用于演示）
type MockHostRepository struct {
	db map[string]*model.Host
}

func NewMockHostRepository() *MockHostRepository {
	return &MockHostRepository{
		db: make(map[string]*model.Host),
	}
}

func (r *MockHostRepository) Create(host *model.Host) error {
	r.db[host.ID] = host
	return nil
}

func (r *MockHostRepository) Update(host *model.Host) error {
	r.db[host.ID] = host
	return nil
}

func (r *MockHostRepository) Delete(id string) error {
	delete(r.db, id)
	return nil
}

func (r *MockHostRepository) GetByID(id string) (*model.Host, error) {
	host, ok := r.db[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return host, nil
}

func (r *MockHostRepository) GetByName(name string) (*model.Host, error) {
	for _, host := range r.db {
		if host.Name == name {
			return host, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// HostFilter 主机过滤条件
type HostFilter struct {
	Group   string
	Keyword string
	Status  string
	Tags    []string
}

func (r *MockHostRepository) List(filter HostFilter) ([]*model.Host, error) {
	hosts := make([]*model.Host, 0, len(r.db))
	for _, host := range r.db {
		hosts = append(hosts, host)
	}
	return hosts, nil
}

func (r *MockHostRepository) UpdateStatus(id string, status string) error {
	host, ok := r.db[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	host.Status = status
	return nil
}

// Example_fullWorkflow 展示完整的缓存工作流程
func Example_fullWorkflow() {
	ctx := context.Background()

	// 1. 初始化缓存
	cacheConfig := cache.DefaultCacheConfig()
	cacheConfig.EnableMetrics = true
	memoryCache := cache.NewMemoryCache(cacheConfig)
	defer memoryCache.Close()

	// 2. 初始化 Mock Repository
	mockRepo := NewMockHostRepository()
	mockRepo.Create(&model.Host{
		ID:     "host1",
		Name:   "Web Server 1",
		IP:     "192.168.1.10",
		Port:   22,
		User:   "admin",
		Status: "online",
	})

	// 3. 第一次请求（缓存未命中）
	cacheKey := cache.NewCacheKey("host").Build("id", "host1")
	var host model.Host
	err := memoryCache.Get(ctx, cacheKey, &host)
	if err != nil {
		// 从数据库加载
		hostFromDB, _ := mockRepo.GetByID("host1")
		host = *hostFromDB

		// 写入缓存
		_ = memoryCache.Set(ctx, cacheKey, host, 10*time.Minute)
		fmt.Println("从数据库加载")
	}

	// 4. 第二次请求（缓存命中）
	err = memoryCache.Get(ctx, cacheKey, &host)
	if err == nil {
		fmt.Println("从缓存加载")
	}

	// 5. 更新数据时清除缓存
	host.Name = "Web Server 1 (Updated)"
	mockRepo.Update(&host)
	_ = memoryCache.Delete(ctx, cacheKey)

	// 6. 查看缓存指标
	metrics := memoryCache.GetMetrics()
	fmt.Printf("缓存命中率: %.2f%%\n", metrics.HitRate*100)
	fmt.Printf("缓存大小: %d\n", metrics.Size)
}
