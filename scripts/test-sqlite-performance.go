package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Host 测试模型
type Host struct {
	ID      string `gorm:"primaryKey"`
	Name    string `gorm:"uniqueIndex;not null"`
	IP      string `gorm:"not null"`
	Group   string `gorm:"index"`
	Status  string `gorm:"index"`
	Created int64  `gorm:"index"`
}

func main() {
	// 测试数据库路径
	dbPath := "test_performance.db"

	// 初始化数据库（带优化）
	fmt.Println("========================================")
	fmt.Println("SQLite WAL 模式性能测试")
	fmt.Println("========================================\n")

	db, err := initDB(dbPath)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 清理旧数据
	db.Exec("DROP TABLE IF EXISTS hosts")

	// 创建表
	if err := db.AutoMigrate(&Host{}); err != nil {
		log.Fatalf("迁移失败: %v", err)
	}

	// 创建测试索引
	createIndexes(db)

	// 显示当前配置
	showDBConfig(db)

	// 测试 1：插入性能
	fmt.Println("\n========================================")
	fmt.Println("测试 1：插入性能")
	fmt.Println("========================================")
	testInsert(db, 1000)

	// 测试 2：并发读取
	fmt.Println("\n========================================")
	fmt.Println("测试 2：并发读取性能")
	fmt.Println("========================================")
	testConcurrentRead(db, 100)

	// 测试 3：索引查询
	fmt.Println("\n========================================")
	fmt.Println("测试 3：索引查询性能")
	fmt.Println("========================================")
	testIndexedQueries(db)

	// 测试 4：写入与读取并发
	fmt.Println("\n========================================")
	fmt.Println("测试 4：读写并发测试")
	fmt.Println("========================================")
	testConcurrentReadWrite(db)

	fmt.Println("\n========================================")
	fmt.Println("性能测试完成！")
	fmt.Println("========================================")
}

// initDB 初始化数据库并配置 WAL 模式
func initDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	// 配置 WAL 模式和性能优化
	pragmaStatements := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-2000",
		"PRAGMA wal_autocheckpoint=1000",
		"PRAGMA busy_timeout=5000",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA mmap_size=268435456",
		"PRAGMA foreign_keys=ON",
	}

	for _, pragma := range pragmaStatements {
		if _, err := sqlDB.Exec(pragma); err != nil {
			log.Printf("执行 PRAGMA 失败: %s, 错误: %v", pragma, err)
		} else {
			log.Printf("✓ %s", pragma)
		}
	}

	return db, nil
}

// createIndexes 创建性能索引
func createIndexes(db *gorm.DB) {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_hosts_group_status ON hosts(`group`, status)",
		"CREATE INDEX IF NOT EXISTS idx_hosts_created ON hosts(created DESC)",
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Printf("创建索引失败: %s, 错误: %v", idx, err)
		} else {
			log.Printf("✓ 索引创建成功: %s", idx)
		}
	}
}

// showDBConfig 显示数据库配置
func showDBConfig(db *gorm.DB) {
	fmt.Println("\n当前数据库配置：")
	fmt.Println("========================================")

	var journalMode string
	db.Raw("PRAGMA journal_mode").Scan(&journalMode)
	fmt.Printf("Journal Mode: %s (WAL 模式)\n", journalMode)

	var synchronous string
	db.Raw("PRAGMA synchronous").Scan(&synchronous)
	fmt.Printf("Synchronous: %s\n", synchronous)

	var cacheSize int
	db.Raw("PRAGMA cache_size").Scan(&cacheSize)
	fmt.Printf("Cache Size: %d KB\n", -cacheSize)

	var mmapSize int
	db.Raw("PRAGMA mmap_size").Scan(&mmapSize)
	fmt.Printf("MMAP Size: %d bytes (%.2f MB)\n", mmapSize, float64(mmapSize)/1024/1024)
}

// testInsert 测试插入性能
func testInsert(db *gorm.DB, count int) {
	start := time.Now()

	// 批量插入
	hosts := make([]Host, count)
	for i := 0; i < count; i++ {
		hosts[i] = Host{
			ID:      fmt.Sprintf("host-%d", i),
			Name:    fmt.Sprintf("host-%d", i),
			IP:      fmt.Sprintf("192.168.1.%d", i%256),
			Group:   fmt.Sprintf("group-%d", i%10),
			Status:  []string{"online", "offline", "unknown"}[i%3],
			Created: time.Now().Unix(),
		}
	}

	if err := db.CreateInBatches(hosts, 100).Error; err != nil {
		log.Printf("批量插入失败: %v", err)
		return
	}

	duration := time.Since(start)
	fmt.Printf("插入 %d 条记录耗时: %v\n", count, duration)
	fmt.Printf("平均每条: %.2f ms\n", float64(duration.Milliseconds())/float64(count))

	var total int64
	db.Model(&Host{}).Count(&total)
	fmt.Printf("当前总记录数: %d\n", total)
}

// testConcurrentRead 测试并发读取
func testConcurrentRead(db *gorm.DB, workers int) {
	start := time.Now()

	done := make(chan bool, workers)

	for i := 0; i < workers; i++ {
		go func(id int) {
			var hosts []Host
			db.Limit(10).Find(&hosts)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < workers; i++ {
		<-done
	}

	duration := time.Since(start)
	fmt.Printf("%d 个并发读取操作耗时: %v\n", workers, duration)
	fmt.Printf("平均每次: %.2f ms\n", float64(duration.Milliseconds())/float64(workers))
}

// testIndexedQueries 测试索引查询
func testIndexedQueries(db *gorm.DB) {
	// 测试 1：状态查询
	start := time.Now()
	var count1 int64
	db.Model(&Host{}).Where("status = ?", "online").Count(&count1)
	fmt.Printf("状态查询 (SELECT COUNT(*) WHERE status='online'): %v\n", time.Since(start))

	// 测试 2：组+状态组合查询
	start = time.Now()
	var hosts []Host
	db.Where("`group` = ? AND status = ?", "group-1", "online").Limit(10).Find(&hosts)
	fmt.Printf("组合查询 (WHERE group='group-1' AND status='online'): %v\n", time.Since(start))

	// 测试 3：排序查询
	start = time.Now()
	db.Order("created DESC").Limit(10).Find(&hosts)
	fmt.Printf("排序查询 (ORDER BY created DESC LIMIT 10): %v\n", time.Since(start))

	// 测试 4：分析查询计划
	fmt.Println("\n查询计划分析：")
	fmt.Println("========================================")
	type QueryPlan struct {
		Detail string
	}

	var plans []QueryPlan
	db.Raw("EXPLAIN QUERY PLAN SELECT * FROM hosts WHERE status = 'online'").Scan(&plans)
	fmt.Println("查询 1: SELECT * FROM hosts WHERE status = 'online'")
	for _, plan := range plans {
		fmt.Printf("  %s\n", plan.Detail)
	}
}

// testConcurrentReadWrite 测试读写并发
func testConcurrentReadWrite(db *gorm.DB) {
	writeDone := make(chan bool)
	readDone := make(chan bool)

	// 启动写入 goroutine
	go func() {
		start := time.Now()
		for i := 0; i < 10; i++ {
			host := Host{
				ID:      fmt.Sprintf("concurrent-%d", i),
				Name:    fmt.Sprintf("concurrent-host-%d", i),
				IP:      "10.0.0.1",
				Group:   "test",
				Status:  "online",
				Created: time.Now().Unix(),
			}
			db.Create(&host)
		}
		fmt.Printf("写入 10 条记录耗时: %v\n", time.Since(start))
		writeDone <- true
	}()

	// 启动读取 goroutine
	go func() {
		start := time.Now()
		for i := 0; i < 10; i++ {
			var hosts []Host
			db.Limit(10).Find(&hosts)
			time.Sleep(10 * time.Millisecond)
		}
		fmt.Printf("读取 10 次耗时: %v\n", time.Since(start))
		readDone <- true
	}()

	// 等待两个操作完成
	<-writeDone
	<-readDone

	fmt.Println("\n✓ 读写并发测试通过（WAL 模式允许读取不阻塞写入）")
}
