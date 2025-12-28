package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"ai-ops/internal/ssh"
)

func main() {
	fmt.Println("=== SSH 认证快速测试 ===\n")

	// 创建 SSH 连接池
	config := ssh.Config{
		DefaultTimeout:    15 * time.Second,
		ConnectTimeout:    10 * time.Second,
		MaxConnections:    10,
		MaxConcurrent:     5,
		KeepaliveInterval: 30 * time.Second,
	}
	pool := ssh.NewPool(config)

	// 从环境变量或使用默认值
	host := os.Getenv("TEST_HOST")
	if host == "" {
		host = "127.0.0.1" // 默认本地测试
	}

	user := os.Getenv("TEST_USER")
	if user == "" {
		user = os.Getenv("USER")
		if user == "" {
			user = "root"
		}
	}

	port := 22
	fmt.Printf("测试主机: %s@%s:%d\n\n", user, host, port)

	// 测试 1: Auto 模式
	fmt.Println("[Test 1] Auto 自动检测模式")
	pool.AddHost(ssh.HostInfo{
		Name:     "test-auto",
		Host:     host,
		Port:     port,
		User:     user,
		AuthType: "auto",
	})

	output, err := pool.Exec("test-auto", "uname -a")
	if err != nil {
		log.Printf("  ✗ 失败: %v\n", err)
	} else {
		log.Printf("  ✓ 成功\n  输出: %s\n", output)
	}

	// 测试 2: 检查连接池状态
	fmt.Println("\n[Test 2] 连接池状态")
	log.Printf("  连接数: %d, 主机数: %d\n", pool.ConnectionCount(), pool.HostCount())

	// 清理
	pool.Close()
	fmt.Println("\n=== 测试完成 ===")
}
