package ssh

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-ops/internal/security"
	"ai-ops/pkg/logger"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var commandPolicyStore *security.PolicyStore
var commandAuditLogger *security.AuditLogger

// SetCommandPolicyStore sets global command policy store and audit logger used by SSH execution.
// This is a process-wide guardrail to ensure only whitelisted (read-only) commands can be executed.
func SetCommandPolicyStore(store *security.PolicyStore, audit *security.AuditLogger) {
	commandPolicyStore = store
	commandAuditLogger = audit
}

// HostInfo 主机信息
type HostInfo struct {
	Name       string   // 主机名称
	Host       string   // IP 或域名
	Port       int      // SSH 端口
	User       string   // 用户名
	Group      string   // 分组
	Tags       []string // 标签
	AuthType   string   // 认证类型: auto / password / key / key_content
	Password   string   // 密码
	KeyPath    string   // 密钥路径
	KeyContent string   // 密钥内容（直接传入）
}

// ExecResult 执行结果
type ExecResult struct {
	Host    string        // 主机
	Output  string        // 标准输出
	Error   error         // 错误
	Elapsed time.Duration // 执行耗时
}

// Config SSH 配置
type Config struct {
	DefaultTimeout    time.Duration // 默认执行超时
	ConnectTimeout    time.Duration // 连接超时
	MaxConnections    int           // 最大连接数
	MaxConcurrent     int           // 最大并发执行数
	KeepaliveInterval time.Duration // 心跳间隔
}

// Executor SSH 执行器接口
type Executor interface {
	// Exec 执行命令
	Exec(host string, cmd string) (string, error)

	// ExecWithTimeout 带超时执行
	ExecWithTimeout(host string, cmd string, timeout time.Duration) (string, error)

	// BatchExec 批量执行
	BatchExec(hosts []string, cmd string) []ExecResult

	// AddHost 添加主机
	AddHost(info HostInfo)

	// RemoveHost 移除主机
	RemoveHost(name string)

	// Close 关闭所有连接
	Close()
}

// Pool SSH 连接池
type Pool struct {
	connections map[string]*ssh.Client
	hosts       map[string]HostInfo
	config      Config
	mu          sync.RWMutex
}

// NewPool 创建连接池
func NewPool(cfg Config) *Pool {
	// 设置默认值
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.MaxConnections == 0 {
		cfg.MaxConnections = 100
	}
	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = 20
	}

	return &Pool{
		connections: make(map[string]*ssh.Client),
		hosts:       make(map[string]HostInfo),
		config:      cfg,
	}
}

// AddHost 添加主机信息
func (p *Pool) AddHost(info HostInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 设置默认端口
	if info.Port == 0 {
		info.Port = 22
	}
	p.hosts[info.Name] = info
}

// RemoveHost 移除主机
func (p *Pool) RemoveHost(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 关闭连接
	if conn, ok := p.connections[name]; ok {
		conn.Close()
		delete(p.connections, name)
	}
	delete(p.hosts, name)
}

// GetHost 获取主机信息
func (p *Pool) GetHost(name string) (HostInfo, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	info, ok := p.hosts[name]
	return info, ok
}

// getConnection 获取或创建连接
func (p *Pool) getConnection(name string) (*ssh.Client, error) {
	p.mu.RLock()
	conn, connExists := p.connections[name]
	hostInfo, hostExists := p.hosts[name]
	p.mu.RUnlock()

	if !hostExists {
		return nil, fmt.Errorf("主机不存在: %s", name)
	}

	// 检查现有连接是否有效
	if connExists && p.isAlive(conn) {
		return conn, nil
	}

	// 创建新连接
	return p.createConnection(name, hostInfo)
}

// createConnection 创建 SSH 连接
func (p *Pool) createConnection(name string, info HostInfo) (*ssh.Client, error) {
	// 如果是 auto 模式，尝试所有认证方式
	if info.AuthType == "auto" || info.AuthType == "" {
		client, err := p.tryAllAuthMethods(name, info)
		if err != nil {
			return nil, err
		}
		return client, nil
	}

	// 显式指定的认证方式
	authMethods, err := p.getAuthMethods(info)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            info.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证
		Timeout:         p.config.ConnectTimeout,
	}

	addr := fmt.Sprintf("%s:%d", info.Host, info.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}

	// 保存连接
	p.mu.Lock()
	if oldConn, ok := p.connections[name]; ok {
		oldConn.Close()
	}
	p.connections[name] = client
	p.mu.Unlock()

	return client, nil
}

// tryAllAuthMethods 尝试所有可用的认证方式（auto 模式）
func (p *Pool) tryAllAuthMethods(name string, info HostInfo) (*ssh.Client, error) {
	// 定义尝试顺序：SSH Agent -> 密码 -> 私钥文件
	attempts := []struct {
		name string
		fn   func() (*ssh.Client, error)
	}{
		{"SSH Agent", func() (*ssh.Client, error) {
			return p.trySSHAgent(name, info)
		}},
		{"私钥文件", func() (*ssh.Client, error) {
			return p.tryKeyFiles(name, info)
		}},
		{"密码", func() (*ssh.Client, error) {
			if info.Password == "" {
				return nil, fmt.Errorf("未配置密码")
			}
			authMethods := []ssh.AuthMethod{ssh.Password(info.Password)}
			return p.connectWithAuth(name, info, authMethods)
		}},
	}

	var lastErr error
	for _, attempt := range attempts {
		client, err := attempt.fn()
		if err == nil {
			logger.Info("认证成功", zap.String("host", info.Host), zap.String("method", attempt.name))
			return client, nil
		}
		lastErr = err
		logger.Debug("认证方式失败", zap.String("host", info.Host), zap.String("method", attempt.name), zap.Error(err))
	}

	return nil, fmt.Errorf("所有认证方式均失败: %w", lastErr)
}

// trySSHAgent 尝试使用 SSH Agent 认证
func (p *Pool) trySSHAgent(name string, info HostInfo) (*ssh.Client, error) {
	// 尝试连接到 SSH Agent
	sshAgentSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAgentSock == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK 环境变量未设置")
	}

	conn, err := net.Dial("unix", sshAgentSock)
	if err != nil {
		return nil, fmt.Errorf("连接 SSH Agent 失败: %w", err)
	}

	agentClient := agent.NewClient(conn)
	signers, err := agentClient.Signers()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("从 Agent 获取签名者失败: %w", err)
	}

	if len(signers) == 0 {
		conn.Close()
		return nil, fmt.Errorf("SSH Agent 中没有可用的密钥")
	}

	// 使用第一个可用的签名者
	authMethods := []ssh.AuthMethod{ssh.PublicKeys(signers[0])}
	client, err := p.connectWithAuth(name, info, authMethods)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// 保存连接以便后续关闭
	return client, nil
}

// tryKeyFiles 尝试使用常见的私钥文件
func (p *Pool) tryKeyFiles(name string, info HostInfo) (*ssh.Client, error) {
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return nil, fmt.Errorf("无法确定用户主目录")
	}

	// 定义要尝试的私钥文件列表（按优先级）
	keyFiles := []string{}

	// 如果指定了 KeyPath，优先使用
	if info.KeyPath != "" {
		keyPath := info.KeyPath
		if strings.HasPrefix(keyPath, "~") {
			keyPath = strings.Replace(keyPath, "~", home, 1)
		}
		keyFiles = append(keyFiles, keyPath)
	}

	// 默认私钥文件列表（按常见优先级）
	defaultKeys := []string{
		"id_rsa",      // 最常见的 RSA 密钥
		"id_ed25519",  // Ed25519 密钥（更安全，推荐）
		"id_ecdsa",    // ECDSA 密钥
		"id_ecdsa_sk", // 支持 U2F/FIDO2 的 ECDSA 密钥
		"id_ed25519_sk", // 支持 U2F/FIDO2 的 Ed25519 密钥
		"id_dsa",      // 旧的 DSA 密钥（不推荐）
	}

	sshDir := filepath.Join(home, ".ssh")
	for _, key := range defaultKeys {
		keyFiles = append(keyFiles, filepath.Join(sshDir, key))
	}

	// 尝试每个私钥文件
	var lastErr error
	for _, keyPath := range keyFiles {
		if _, err := os.Stat(keyPath); err != nil {
			continue // 文件不存在，跳过
		}

		key, err := os.ReadFile(keyPath)
		if err != nil {
			lastErr = err
			continue
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			// 尝试解析带密码的私钥
			if _, ok := err.(*ssh.PassphraseMissingError); ok {
				logger.Debug("私钥需要密码", zap.String("key", keyPath))
				lastErr = fmt.Errorf("私钥 %s 需要密码，请使用 key_content 并提供密码", keyPath)
				continue
			}
			lastErr = err
			continue
		}

		authMethods := []ssh.AuthMethod{ssh.PublicKeys(signer)}
		client, err := p.connectWithAuth(name, info, authMethods)
		if err == nil {
			logger.Info("使用私钥认证成功", zap.String("host", info.Host), zap.String("key", keyPath))
			return client, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("尝试所有私钥文件均失败: %w", lastErr)
	}
	return nil, fmt.Errorf("没有找到可用的私钥文件")
}

// connectWithAuth 使用指定的认证方法创建连接
func (p *Pool) connectWithAuth(name string, info HostInfo, authMethods []ssh.AuthMethod) (*ssh.Client, error) {
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("没有可用的认证方法")
	}

	config := &ssh.ClientConfig{
		User:            info.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         p.config.ConnectTimeout,
	}

	addr := fmt.Sprintf("%s:%d", info.Host, info.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}

	// 保存连接
	p.mu.Lock()
	if oldConn, ok := p.connections[name]; ok {
		oldConn.Close()
	}
	p.connections[name] = client
	p.mu.Unlock()

	return client, nil
}

// getAuthMethods 根据认证类型获取认证方法
func (p *Pool) getAuthMethods(info HostInfo) ([]ssh.AuthMethod, error) {
	switch info.AuthType {
	case "password":
		if info.Password == "" {
			return nil, fmt.Errorf("密码认证需要提供密码")
		}
		return []ssh.AuthMethod{ssh.Password(info.Password)}, nil

	case "key_content", "key":
		var keyBytes []byte
		var err error

		if info.AuthType == "key_content" {
			// 直接使用传入的私钥内容
			if info.KeyContent == "" {
				return nil, fmt.Errorf("私钥内容为空")
			}
			keyBytes = []byte(info.KeyContent)
		} else {
			// 从文件读取私钥
			keyPath := info.KeyPath
			home, homeErr := os.UserHomeDir()

			if keyPath == "" {
				if homeErr != nil || home == "" {
					return nil, fmt.Errorf("未配置 key_path 且无法确定用户主目录")
				}
				keyPath = filepath.Join(home, ".ssh", "id_rsa")
			}

			// 展开 ~ 路径
			if strings.HasPrefix(keyPath, "~") {
				if homeErr != nil || home == "" {
					return nil, fmt.Errorf("展开密钥路径失败（无法确定用户主目录）")
				}
				keyPath = strings.Replace(keyPath, "~", home, 1)
			}

			if _, err := os.Stat(keyPath); err != nil {
				return nil, fmt.Errorf("读取密钥文件失败: %w", err)
			}

			keyBytes, err = os.ReadFile(keyPath)
			if err != nil {
				return nil, fmt.Errorf("读取密钥文件失败: %w", err)
			}
		}

		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil

	case "agent":
		// SSH Agent 模式需要单独处理
		return nil, fmt.Errorf("agent 模式请使用 auto 模式，系统会自动检测并使用 SSH Agent")

	default:
		return nil, fmt.Errorf("不支持的认证类型: %s", info.AuthType)
	}
}

// isAlive 检查连接是否有效
func (p *Pool) isAlive(conn *ssh.Client) bool {
	if conn == nil {
		return false
	}

	// 尝试创建一个会话来检查连接
	session, err := conn.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}

// Exec 执行命令
func (p *Pool) Exec(host string, cmd string) (string, error) {
	return p.ExecWithTimeout(host, cmd, p.config.DefaultTimeout)
}

// ExecWithTimeout 带超时执行命令
func (p *Pool) ExecWithTimeout(host string, cmd string, timeout time.Duration) (string, error) {
	start := time.Now()
	if commandPolicyStore != nil {
		allowed, reason := commandPolicyStore.Check(cmd)
		if !allowed {
			if commandAuditLogger != nil {
				_ = commandAuditLogger.Append(security.CommandAuditEvent{
					Time:    time.Now(),
					Host:    host,
					Command: cmd,
					Allowed: false,
					Reason:  reason,
				})
			}
			logger.Warn("SSH command blocked by whitelist policy",
				zap.String("host", host),
				zap.String("command", cmd),
				zap.String("reason", reason),
			)
			return "", fmt.Errorf("命令被白名单策略拦截: %s", reason)
		}
	}

	conn, err := p.getConnection(host)
	if err != nil {
		if commandAuditLogger != nil {
			_ = commandAuditLogger.Append(security.CommandAuditEvent{
				Time:      time.Now(),
				Host:      host,
				Command:   cmd,
				Allowed:   true,
				Reason:    "connection_failed",
				ElapsedMs: time.Since(start).Milliseconds(),
				Error:     err.Error(),
			})
		}
		return "", err
	}

	session, err := conn.NewSession()
	if err != nil {
		// 连接可能已断开，尝试重新连接
		p.mu.Lock()
		delete(p.connections, host)
		p.mu.Unlock()

		conn, err = p.getConnection(host)
		if err != nil {
			return "", fmt.Errorf("重新连接失败: %w", err)
		}

		session, err = conn.NewSession()
		if err != nil {
			return "", fmt.Errorf("创建会话失败: %w", err)
		}
	}
	defer session.Close()

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 捕获输出
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// 执行命令
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case err := <-done:
		if err != nil {
			// 返回 stderr 内容作为错误信息
			errMsg := stderr.String()
			if errMsg == "" {
				errMsg = err.Error()
			}
			if commandAuditLogger != nil {
				_ = commandAuditLogger.Append(security.CommandAuditEvent{
					Time:      time.Now(),
					Host:      host,
					Command:   cmd,
					Allowed:   true,
					Reason:    "executed",
					ElapsedMs: time.Since(start).Milliseconds(),
					Error:     errMsg,
				})
			}
			return stdout.String(), fmt.Errorf("执行失败: %s", errMsg)
		}
		if commandAuditLogger != nil {
			_ = commandAuditLogger.Append(security.CommandAuditEvent{
				Time:      time.Now(),
				Host:      host,
				Command:   cmd,
				Allowed:   true,
				Reason:    "executed",
				ElapsedMs: time.Since(start).Milliseconds(),
			})
		}
		return stdout.String(), nil
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		if commandAuditLogger != nil {
			_ = commandAuditLogger.Append(security.CommandAuditEvent{
				Time:      time.Now(),
				Host:      host,
				Command:   cmd,
				Allowed:   true,
				Reason:    "timeout",
				ElapsedMs: time.Since(start).Milliseconds(),
				Error:     fmt.Sprintf("timeout: %v", timeout),
			})
		}
		return "", fmt.Errorf("执行超时: %v", timeout)
	}
}

// BatchExec 批量并发执行
func (p *Pool) BatchExec(hosts []string, cmd string) []ExecResult {
	results := make([]ExecResult, len(hosts))
	var wg sync.WaitGroup

	// 控制并发数
	semaphore := make(chan struct{}, p.config.MaxConcurrent)

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()

			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			start := time.Now()
			output, err := p.Exec(h, cmd)
			elapsed := time.Since(start)

			results[idx] = ExecResult{
				Host:    h,
				Output:  output,
				Error:   err,
				Elapsed: elapsed,
			}
		}(i, host)
	}

	wg.Wait()
	return results
}

// Close 关闭所有连接
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for name, conn := range p.connections {
		conn.Close()
		delete(p.connections, name)
	}
}

// ConnectionCount 获取当前连接数
func (p *Pool) ConnectionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// HostCount 获取主机数量
func (p *Pool) HostCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.hosts)
}

// ListHosts 获取所有主机列表
func (p *Pool) ListHosts() []HostInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	hosts := make([]HostInfo, 0, len(p.hosts))
	for _, info := range p.hosts {
		hosts = append(hosts, info)
	}
	return hosts
}
