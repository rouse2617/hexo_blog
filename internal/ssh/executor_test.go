package ssh

import (
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	// 测试默认配置
	pool := NewPool(Config{})

	if pool.config.DefaultTimeout != 30*time.Second {
		t.Errorf("DefaultTimeout 默认值期望 30s, 实际 %v", pool.config.DefaultTimeout)
	}
	if pool.config.ConnectTimeout != 10*time.Second {
		t.Errorf("ConnectTimeout 默认值期望 10s, 实际 %v", pool.config.ConnectTimeout)
	}
	if pool.config.MaxConnections != 100 {
		t.Errorf("MaxConnections 默认值期望 100, 实际 %d", pool.config.MaxConnections)
	}
	if pool.config.MaxConcurrent != 20 {
		t.Errorf("MaxConcurrent 默认值期望 20, 实际 %d", pool.config.MaxConcurrent)
	}

	// 测试自定义配置
	pool2 := NewPool(Config{
		DefaultTimeout: 60 * time.Second,
		MaxConcurrent:  50,
	})

	if pool2.config.DefaultTimeout != 60*time.Second {
		t.Errorf("DefaultTimeout 期望 60s, 实际 %v", pool2.config.DefaultTimeout)
	}
	if pool2.config.MaxConcurrent != 50 {
		t.Errorf("MaxConcurrent 期望 50, 实际 %d", pool2.config.MaxConcurrent)
	}
}

func TestAddRemoveHost(t *testing.T) {
	pool := NewPool(Config{})

	// 添加主机
	pool.AddHost(HostInfo{
		Name:     "test-host",
		Host:     "192.168.1.100",
		Port:     22,
		User:     "root",
		AuthType: "password",
		Password: "test123",
	})

	if pool.HostCount() != 1 {
		t.Errorf("添加后主机数期望 1, 实际 %d", pool.HostCount())
	}

	// 获取主机
	info, ok := pool.GetHost("test-host")
	if !ok {
		t.Error("获取主机失败")
	}
	if info.Host != "192.168.1.100" {
		t.Errorf("Host 期望 192.168.1.100, 实际 %s", info.Host)
	}
	if info.User != "root" {
		t.Errorf("User 期望 root, 实际 %s", info.User)
	}

	// 移除主机
	pool.RemoveHost("test-host")
	if pool.HostCount() != 0 {
		t.Errorf("移除后主机数期望 0, 实际 %d", pool.HostCount())
	}

	// 获取不存在的主机
	_, ok = pool.GetHost("test-host")
	if ok {
		t.Error("移除后仍能获取主机")
	}
}

func TestDefaultPort(t *testing.T) {
	pool := NewPool(Config{})

	// 添加主机时不指定端口
	pool.AddHost(HostInfo{
		Name: "test-host",
		Host: "192.168.1.100",
		User: "root",
	})

	info, _ := pool.GetHost("test-host")
	if info.Port != 22 {
		t.Errorf("默认端口期望 22, 实际 %d", info.Port)
	}
}

func TestGetConnectionHostNotFound(t *testing.T) {
	pool := NewPool(Config{})

	_, err := pool.getConnection("nonexistent")
	if err == nil {
		t.Error("期望主机不存在错误，但没有返回错误")
	}
}

func TestExecHostNotFound(t *testing.T) {
	pool := NewPool(Config{})

	_, err := pool.Exec("nonexistent", "ls")
	if err == nil {
		t.Error("期望主机不存在错误，但没有返回错误")
	}
}

func TestBatchExecEmpty(t *testing.T) {
	pool := NewPool(Config{})

	results := pool.BatchExec([]string{}, "ls")
	if len(results) != 0 {
		t.Errorf("空主机列表期望返回空结果, 实际 %d", len(results))
	}
}

func TestClose(t *testing.T) {
	pool := NewPool(Config{})

	pool.AddHost(HostInfo{
		Name: "test-host",
		Host: "192.168.1.100",
		User: "root",
	})

	// Close 不应该 panic
	pool.Close()

	if pool.ConnectionCount() != 0 {
		t.Errorf("关闭后连接数期望 0, 实际 %d", pool.ConnectionCount())
	}
}

func TestMultipleHosts(t *testing.T) {
	pool := NewPool(Config{})

	// 添加多个主机
	hosts := []HostInfo{
		{Name: "host1", Host: "192.168.1.1", User: "root"},
		{Name: "host2", Host: "192.168.1.2", User: "admin"},
		{Name: "host3", Host: "192.168.1.3", User: "user"},
	}

	for _, h := range hosts {
		pool.AddHost(h)
	}

	if pool.HostCount() != 3 {
		t.Errorf("主机数期望 3, 实际 %d", pool.HostCount())
	}

	// 验证每个主机
	for _, h := range hosts {
		info, ok := pool.GetHost(h.Name)
		if !ok {
			t.Errorf("获取主机 %s 失败", h.Name)
		}
		if info.Host != h.Host {
			t.Errorf("主机 %s 的 Host 期望 %s, 实际 %s", h.Name, h.Host, info.Host)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	pool := NewPool(Config{})

	// 并发添加主机
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			pool.AddHost(HostInfo{
				Name: "host-" + string(rune('a'+idx)),
				Host: "192.168.1.1",
				User: "root",
			})
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	if pool.HostCount() != 10 {
		t.Errorf("并发添加后主机数期望 10, 实际 %d", pool.HostCount())
	}
}

func TestExecResultStruct(t *testing.T) {
	result := ExecResult{
		Host:    "test-host",
		Output:  "hello world",
		Error:   nil,
		Elapsed: 100 * time.Millisecond,
	}

	if result.Host != "test-host" {
		t.Errorf("Host 期望 test-host, 实际 %s", result.Host)
	}
	if result.Output != "hello world" {
		t.Errorf("Output 期望 hello world, 实际 %s", result.Output)
	}
	if result.Error != nil {
		t.Errorf("Error 期望 nil, 实际 %v", result.Error)
	}
}
