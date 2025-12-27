package repository

import (
	"ai-ops/internal/model"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	// 使用临时文件
	tmpFile := "test_" + uuid.New().String() + ".db"
	db, err := gorm.Open(sqlite.Open(tmpFile), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(&model.Host{})
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 返回清理函数
	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile)
	}

	return db, cleanup
}

// TestHostRepository_List_WithTags 测试标签过滤功能
func TestHostRepository_List_WithTags(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewHostRepository(db)

	// 创建测试主机
	host1 := &model.Host{
		ID:        uuid.New().String(),
		Name:      "host1",
		IP:        "192.168.1.1",
		Port:      22,
		User:      "root",
		Tags:      []string{"web", "production"},
		Status:    model.HostStatusOnline,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	host2 := &model.Host{
		ID:        uuid.New().String(),
		Name:      "host2",
		IP:        "192.168.1.2",
		Port:      22,
		User:      "root",
		Tags:      []string{"db", "production"},
		Status:    model.HostStatusOnline,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	host3 := &model.Host{
		ID:        uuid.New().String(),
		Name:      "host3",
		IP:        "192.168.1.3",
		Port:      22,
		User:      "root",
		Tags:      []string{"web", "staging"},
		Status:    model.HostStatusOnline,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	host4 := &model.Host{
		ID:        uuid.New().String(),
		Name:      "host4",
		IP:        "192.168.1.4",
		Port:      22,
		User:      "root",
		Tags:      []string{"cache"},
		Status:    model.HostStatusOnline,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 创建主机
	if err := repo.Create(host1); err != nil {
		t.Fatalf("创建host1失败: %v", err)
	}
	if err := repo.Create(host2); err != nil {
		t.Fatalf("创建host2失败: %v", err)
	}
	if err := repo.Create(host3); err != nil {
		t.Fatalf("创建host3失败: %v", err)
	}
	if err := repo.Create(host4); err != nil {
		t.Fatalf("创建host4失败: %v", err)
	}

	// 测试1: 按单个标签过滤（"web"）
	filter := HostFilter{Tags: []string{"web"}}
	hosts, err := repo.List(filter)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(hosts) != 2 {
		t.Errorf("期望找到2个主机（host1和host3有web标签）, 实际找到 %d 个", len(hosts))
	}
	// 验证找到的主机
	names := make(map[string]bool)
	for _, h := range hosts {
		names[h.Name] = true
	}
	if !names["host1"] || !names["host3"] {
		t.Error("应该找到host1和host3")
	}

	// 测试2: 按多个标签过滤（OR逻辑：匹配任一标签）
	filter = HostFilter{Tags: []string{"db", "cache"}}
	hosts, err = repo.List(filter)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(hosts) != 2 {
		t.Errorf("期望找到2个主机（host2有db标签，host4有cache标签）, 实际找到 %d 个", len(hosts))
	}
	// 验证找到的主机
	names = make(map[string]bool)
	for _, h := range hosts {
		names[h.Name] = true
	}
	if !names["host2"] || !names["host4"] {
		t.Error("应该找到host2和host4")
	}

	// 测试3: 按不存在的标签过滤
	filter = HostFilter{Tags: []string{"nonexistent"}}
	hosts, err = repo.List(filter)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("期望找到0个主机, 实际找到 %d 个", len(hosts))
	}

	// 测试4: 空标签过滤（应该返回所有主机）
	filter = HostFilter{Tags: []string{}}
	hosts, err = repo.List(filter)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(hosts) != 4 {
		t.Errorf("期望找到4个主机, 实际找到 %d 个", len(hosts))
	}

	// 测试5: 结合其他过滤条件（标签 + 状态）
	filter = HostFilter{
		Tags:   []string{"production"},
		Status: model.HostStatusOnline,
	}
	hosts, err = repo.List(filter)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(hosts) != 2 {
		t.Errorf("期望找到2个主机（host1和host2有production标签且在线）, 实际找到 %d 个", len(hosts))
	}
}

// TestHostRepository_List_WithEmptyTags 测试空标签数组的主机
func TestHostRepository_List_WithEmptyTags(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewHostRepository(db)

	// 创建没有标签的主机
	host := &model.Host{
		ID:        uuid.New().String(),
		Name:      "host_no_tags",
		IP:        "192.168.1.1",
		Port:      22,
		User:      "root",
		Tags:      []string{}, // 空标签
		Status:    model.HostStatusOnline,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := repo.Create(host); err != nil {
		t.Fatalf("创建主机失败: %v", err)
	}

	// 按标签过滤时，空标签的主机不应该被匹配
	filter := HostFilter{Tags: []string{"web"}}
	hosts, err := repo.List(filter)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("期望找到0个主机, 实际找到 %d 个", len(hosts))
	}
}








