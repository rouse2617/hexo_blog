package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"ai-ops/internal/ssh"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

func printHeader(title string) {
	fmt.Printf("\n%s%s%s\n", ColorCyan, strings.Repeat("=", 60), ColorReset)
	fmt.Printf("%s[ TEST ] %s%s\n", ColorCyan, title, ColorReset)
	fmt.Printf("%s%s%s\n\n", ColorCyan, strings.Repeat("=", 60), ColorReset)
}

func printSuccess(format string, args ...interface{}) {
	fmt.Printf("%s[✓]%s %s", ColorGreen, ColorReset, fmt.Sprintf(format, args...))
}

func printError(format string, args ...interface{}) {
	fmt.Printf("%s[✗]%s %s", ColorRed, ColorReset, fmt.Sprintf(format, args...))
}

func printInfo(format string, args ...interface{}) {
	fmt.Printf("%s[i]%s %s", ColorBlue, ColorReset, fmt.Sprintf(format, args...))
}

func printWarning(format string, args ...interface{}) {
	fmt.Printf("%s[!]%s %s", ColorYellow, ColorReset, fmt.Sprintf(format, args...))
}

func printSubTest(name string) {
	fmt.Printf("\n%s▶ %s%s\n", ColorPurple, name, ColorReset)
}

func main() {
	printHeader("SSH 认证测试套件")

	// 创建 SSH 连接池
	config := ssh.Config{
		DefaultTimeout:    15 * time.Second,
		ConnectTimeout:    10 * time.Second,
		MaxConnections:    10,
		MaxConcurrent:     5,
		KeepaliveInterval: 30 * time.Second,
	}
	pool := ssh.NewPool(config)

	// 获取测试主机信息
	reader := bufio.NewReader(os.Stdin)

	printInfo("请输入测试主机信息\n\n")

	fmt.Print("主机地址 (IP 或域名): ")
	host, _ := reader.ReadString('\n')
	host = strings.TrimSpace(host)

	fmt.Print("SSH 端口 [22]: ")
	portStr, _ := reader.ReadString('\n')
	portStr = strings.TrimSpace(portStr)
	port := 22
	if portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	fmt.Print("用户名 [root]: ")
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)
	if user == "" {
		user = "root"
	}

	fmt.Print("密码 (可选，按Enter跳过): ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	fmt.Print("私钥路径 (可选，按Enter跳过): ")
	keyPath, _ := reader.ReadString('\n')
	keyPath = strings.TrimSpace(keyPath)

	fmt.Print("私钥内容 (可选，多行内容以 END 结束): ")
	fmt.Println("(输入私钥内容后输入 END 结束)")
	var keyContent strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "END" {
			break
		}
		keyContent.WriteString(line + "\n")
	}

	fmt.Println()
	printHeader("开始测试")

	// 测试计数器
	totalTests := 0
	passedTests := 0

	// 测试结果存储
	type TestResult struct {
		Name   string
		Passed bool
		Error  string
	}
	results := []TestResult{}

	// ========== 测试 1: Auto 模式 (推荐) ==========
	totalTests++
	printSubTest("Test 1: Auto 自动检测模式")
	testName := "Auto 自动检测"
	pool.AddHost(ssh.HostInfo{
		Name:     "test-auto",
		Host:     host,
		Port:     port,
		User:     user,
		AuthType: "auto",
		Password: password,
		KeyPath:  keyPath,
	})

	output, err := pool.Exec("test-auto", "uptime")
	if err != nil {
		printError("测试失败: %v\n", err)
		results = append(results, TestResult{Name: testName, Passed: false, Error: err.Error()})
	} else {
		printSuccess("测试成功\n")
		fmt.Printf("  输出: %s\n", strings.TrimSpace(output))
		results = append(results, TestResult{Name: testName, Passed: true})
		passedTests++
	}

	// ========== 测试 2: SSH Agent ==========
	totalTests++
	printSubTest("Test 2: SSH Agent 认证")
	testName = "SSH Agent"
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		printWarning("SSH_AUTH_SOCK 环境变量未设置，跳过此测试\n")
		results = append(results, TestResult{Name: testName, Passed: false, Error: "SSH Agent not configured"})
	} else {
		printInfo("检测到 SSH Agent: %s\n", os.Getenv("SSH_AUTH_SOCK"))
		pool.AddHost(ssh.HostInfo{
			Name:     "test-agent",
			Host:     host,
			Port:     port,
			User:     user,
			AuthType: "auto", // auto 模式会优先尝试 SSH Agent
		})

		output, err := pool.Exec("test-agent", "uptime")
		if err != nil {
			printError("测试失败: %v\n", err)
			results = append(results, TestResult{Name: testName, Passed: false, Error: err.Error()})
		} else {
			printSuccess("测试成功\n")
			fmt.Printf("  输出: %s\n", strings.TrimSpace(output))
			results = append(results, TestResult{Name: testName, Passed: true})
			passedTests++
		}
	}

	// ========== 测试 3: 密码认证 ==========
	totalTests++
	printSubTest("Test 3: 密码认证")
	testName = "密码认证"
	if password == "" {
		printWarning("未提供密码，跳过此测试\n")
		results = append(results, TestResult{Name: testName, Passed: false, Error: "No password provided"})
	} else {
		pool.AddHost(ssh.HostInfo{
			Name:     "test-password",
			Host:     host,
			Port:     port,
			User:     user,
			AuthType: "password",
			Password: password,
		})

		output, err := pool.Exec("test-password", "uptime")
		if err != nil {
			printError("测试失败: %v\n", err)
			results = append(results, TestResult{Name: testName, Passed: false, Error: err.Error()})
		} else {
			printSuccess("测试成功\n")
			fmt.Printf("  输出: %s\n", strings.TrimSpace(output))
			results = append(results, TestResult{Name: testName, Passed: true})
			passedTests++
		}
	}

	// ========== 测试 4: 私钥文件 ==========
	totalTests++
	printSubTest("Test 4: 私钥文件认证")
	testName = "私钥文件"
	if keyPath == "" {
		printInfo("未指定私钥路径，尝试自动检测常见私钥文件...\n")
		pool.AddHost(ssh.HostInfo{
			Name:     "test-key-auto",
			Host:     host,
			Port:     port,
			User:     user,
			AuthType: "key",
		})

		output, err := pool.Exec("test-key-auto", "uptime")
		if err != nil {
			printError("自动检测失败: %v\n", err)
			results = append(results, TestResult{Name: testName + " (auto)", Passed: false, Error: err.Error()})
		} else {
			printSuccess("自动检测成功\n")
			fmt.Printf("  输出: %s\n", strings.TrimSpace(output))
			results = append(results, TestResult{Name: testName + " (auto)", Passed: true})
			passedTests++
		}
	} else {
		printInfo("使用指定私钥: %s\n", keyPath)
		pool.AddHost(ssh.HostInfo{
			Name:     "test-key-path",
			Host:     host,
			Port:     port,
			User:     user,
			AuthType: "key",
			KeyPath:  keyPath,
		})

		output, err := pool.Exec("test-key-path", "uptime")
		if err != nil {
			printError("测试失败: %v\n", err)
			results = append(results, TestResult{Name: testName, Passed: false, Error: err.Error()})
		} else {
			printSuccess("测试成功\n")
			fmt.Printf("  输出: %s\n", strings.TrimSpace(output))
			results = append(results, TestResult{Name: testName, Passed: true})
			passedTests++
		}
	}

	// ========== 测试 5: 私钥内容 ==========
	totalTests++
	printSubTest("Test 5: 私钥内容认证")
	testName = "私钥内容"
	kc := keyContent.String()
	if kc == "" {
		printWarning("未提供私钥内容，跳过此测试\n")
		results = append(results, TestResult{Name: testName, Passed: false, Error: "No key content provided"})
	} else {
		printInfo("使用提供的私钥内容\n")
		pool.AddHost(ssh.HostInfo{
			Name:       "test-key-content",
			Host:       host,
			Port:       port,
			User:       user,
			AuthType:   "key_content",
			KeyContent: kc,
		})

		output, err := pool.Exec("test-key-content", "uptime")
		if err != nil {
			printError("测试失败: %v\n", err)
			results = append(results, TestResult{Name: testName, Passed: false, Error: err.Error()})
		} else {
			printSuccess("测试成功\n")
			fmt.Printf("  输出: %s\n", strings.TrimSpace(output))
			results = append(results, TestResult{Name: testName, Passed: true})
			passedTests++
		}
	}

	// ========== 测试 6: 批量执行 ==========
	totalTests++
	printSubTest("Test 6: 批量并发执行")
	testName = "批量执行"
	pool.AddHost(ssh.HostInfo{
		Name:     "test-batch-1",
		Host:     host,
		Port:     port,
		User:     user,
		AuthType: "auto",
		Password: password,
		KeyPath:  keyPath,
	})
	pool.AddHost(ssh.HostInfo{
		Name:     "test-batch-2",
		Host:     host,
		Port:     port,
		User:     user,
		AuthType: "auto",
		Password: password,
		KeyPath:  keyPath,
	})

	results2 := pool.BatchExec([]string{"test-batch-1", "test-batch-2"}, "hostname")
	successCount := 0
	for _, result := range results2 {
		if result.Error == nil {
			successCount++
		}
	}
	if successCount == len(results2) {
		printSuccess("批量执行成功 (%d/%d)\n", successCount, len(results2))
		for i, result := range results2 {
			fmt.Printf("  [%d] %s: %s\n", i+1, result.Host, strings.TrimSpace(result.Output))
		}
		results = append(results, TestResult{Name: testName, Passed: true})
		passedTests++
	} else {
		printError("批量执行部分失败 (%d/%d)\n", successCount, len(results2))
		for i, result := range results2 {
			if result.Error != nil {
				fmt.Printf("  [%d] %s: %v\n", i+1, result.Host, result.Error)
			}
		}
		results = append(results, TestResult{Name: testName, Passed: false, Error: "Some hosts failed"})
	}

	// ========== 测试 7: 超时处理 ==========
	totalTests++
	printSubTest("Test 7: 命令超时处理")
	testName = "超时处理"
	pool.AddHost(ssh.HostInfo{
		Name:     "test-timeout",
		Host:     host,
		Port:     port,
		User:     user,
		AuthType: "auto",
		Password: password,
	})

	_, err = pool.ExecWithTimeout("test-timeout", "sleep 10", 2*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "超时") {
			printSuccess("超时检测正常\n")
			results = append(results, TestResult{Name: testName, Passed: true})
			passedTests++
		} else {
			printError("超时测试异常: %v\n", err)
			results = append(results, TestResult{Name: testName, Passed: false, Error: err.Error()})
		}
	} else {
		printError("超时测试失败: 命令应该在2秒后超时\n")
		results = append(results, TestResult{Name: testName, Passed: false, Error: "Timeout not triggered"})
	}

	// ========== 测试 8: 连接状态检查 ==========
	totalTests++
	printSubTest("Test 8: 连接池状态")
	testName = "连接池状态"
	connCount := pool.ConnectionCount()
	hostCount := pool.HostCount()
	printInfo("当前连接数: %d, 主机数: %d\n", connCount, hostCount)
	if connCount > 0 {
		printSuccess("连接池正常工作\n")
		results = append(results, TestResult{Name: testName, Passed: true})
		passedTests++
	} else {
		printWarning("没有活跃连接\n")
		results = append(results, TestResult{Name: testName, Passed: false, Error: "No active connections"})
	}

	// 清理
	pool.Close()

	// ========== 测试报告 ==========
	printHeader("测试报告")
	fmt.Printf("%s总测试数:%s %d\n", ColorWhite, ColorReset, totalTests)
	fmt.Printf("%s通过数:%s %s%d%s\n", ColorGreen, ColorReset, ColorGreen, passedTests, ColorReset)
	fmt.Printf("%s失败数:%s %s%d%s\n\n", ColorRed, ColorReset, ColorRed, totalTests-passedTests, ColorReset)

	fmt.Printf("%s详细结果:%s\n", ColorCyan, ColorReset)
	fmt.Printf("%s%s%s\n", ColorCyan, strings.Repeat("-", 60), ColorReset)
	for i, result := range results {
		status := "✗"
		color := ColorRed
		if result.Passed {
			status = "✓"
			color = ColorGreen
		}
		fmt.Printf("%s[%2d]%s %s%s%s %s\n", color, i+1, ColorReset, color, status, ColorReset, result.Name)
		if result.Error != "" {
			fmt.Printf("      错误: %s\n", result.Error)
		}
	}
	fmt.Printf("%s%s%s\n", ColorCyan, strings.Repeat("-", 60), ColorReset)

	// 建议
	fmt.Printf("\n%s建议:%s\n", ColorYellow, ColorReset)
	if passedTests == totalTests {
		fmt.Printf("  %s所有测试通过！SSH 认证系统工作正常。%s\n", ColorGreen, ColorReset)
		fmt.Printf("  %s推荐在生产环境使用「Auto 自动检测」认证方式。%s\n", ColorCyan, ColorReset)
	} else {
		fmt.Printf("  %s部分测试失败，请检查:%s\n", ColorYellow, ColorReset)
		if os.Getenv("SSH_AUTH_SOCK") == "" {
			fmt.Printf("    • SSH Agent 未配置，请运行: eval $(ssh-agent) && ssh-add\n")
		}
		fmt.Printf("    • 确认主机地址、端口、用户名正确\n")
		fmt.Printf("    • 确认相关认证方式（密码/私钥）配置正确\n")
	}

	fmt.Println()
}
