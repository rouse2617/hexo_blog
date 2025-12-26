# SSH 执行器设计

## 职责

SSH 执行器是 Agent 与远程节点交互的基础设施层。

```
Tool 调用 → SSH 执行器 → 远程节点执行命令 → 返回结果
```

## 核心功能

```
┌─────────────────────────────────────────────────────────┐
│                    SSH 执行器                            │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │                 连接池管理                        │   │
│  │  - 复用连接，避免频繁建立 SSH 连接                 │   │
│  │  - 连接健康检查                                   │   │
│  │  - 自动重连                                       │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │                 命令执行                          │   │
│  │  - 单节点执行                                     │   │
│  │  - 批量并发执行                                   │   │
│  │  - 超时控制                                       │   │
│  │  - 输出捕获                                       │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │                 文件传输                          │   │
│  │  - 上传脚本到远程节点                             │   │
│  │  - 下载文件                                       │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## 接口定义

```go
// SSHExecutor SSH 执行器接口
type SSHExecutor interface {
    // 单节点执行
    Exec(host string, cmd string) (output string, err error)

    // 带超时的执行
    ExecWithTimeout(host string, cmd string, timeout time.Duration) (string, error)

    // 批量执行 (并发)
    BatchExec(hosts []string, cmd string) []ExecResult

    // 上传文件
    Upload(host string, localPath string, remotePath string) error

    // 下载文件
    Download(host string, remotePath string, localPath string) error

    // 关闭连接
    Close()
}

// ExecResult 执行结果
type ExecResult struct {
    Host    string
    Output  string
    Error   error
    Elapsed time.Duration
}
```

## 实现

### 连接池

```go
// SSHPool SSH 连接池
type SSHPool struct {
    connections map[string]*ssh.Client
    hostMgr     *HostManager
    mu          sync.RWMutex
    config      *SSHConfig
}

type SSHConfig struct {
    DefaultTimeout  time.Duration
    MaxConnections  int
    ConnectTimeout  time.Duration
    KeepaliveInterval time.Duration
}

// getConnection 获取或创建连接
func (p *SSHPool) getConnection(host string) (*ssh.Client, error) {
    p.mu.RLock()
    if conn, ok := p.connections[host]; ok {
        p.mu.RUnlock()
        // 检查连接是否有效
        if p.isAlive(conn) {
            return conn, nil
        }
        // 连接失效，需要重建
    } else {
        p.mu.RUnlock()
    }

    // 创建新连接
    return p.createConnection(host)
}

// createConnection 创建 SSH 连接
func (p *SSHPool) createConnection(host string) (*ssh.Client, error) {
    hostInfo, err := p.hostMgr.GetHost(host)
    if err != nil {
        return nil, fmt.Errorf("host not found: %s", host)
    }

    var authMethods []ssh.AuthMethod

    switch hostInfo.AuthType {
    case "password":
        authMethods = append(authMethods, ssh.Password(hostInfo.Password))
    case "key":
        key, err := os.ReadFile(hostInfo.KeyPath)
        if err != nil {
            return nil, err
        }
        signer, err := ssh.ParsePrivateKey(key)
        if err != nil {
            return nil, err
        }
        authMethods = append(authMethods, ssh.PublicKeys(signer))
    }

    config := &ssh.ClientConfig{
        User:            hostInfo.User,
        Auth:            authMethods,
        HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证
        Timeout:         p.config.ConnectTimeout,
    }

    addr := fmt.Sprintf("%s:%d", hostInfo.IP, hostInfo.Port)
    client, err := ssh.Dial("tcp", addr, config)
    if err != nil {
        return nil, err
    }

    p.mu.Lock()
    p.connections[host] = client
    p.mu.Unlock()

    return client, nil
}
```

### 命令执行

```go
// Exec 执行命令
func (e *SSHExecutorImpl) Exec(host string, cmd string) (string, error) {
    return e.ExecWithTimeout(host, cmd, e.config.DefaultTimeout)
}

// ExecWithTimeout 带超时执行
func (e *SSHExecutorImpl) ExecWithTimeout(host string, cmd string, timeout time.Duration) (string, error) {
    conn, err := e.pool.getConnection(host)
    if err != nil {
        return "", fmt.Errorf("连接失败: %w", err)
    }

    session, err := conn.NewSession()
    if err != nil {
        return "", fmt.Errorf("创建会话失败: %w", err)
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
            return stderr.String(), fmt.Errorf("执行失败: %w, stderr: %s", err, stderr.String())
        }
        return stdout.String(), nil
    case <-ctx.Done():
        session.Signal(ssh.SIGKILL)
        return "", fmt.Errorf("执行超时: %v", timeout)
    }
}
```

### 批量执行

```go
// BatchExec 批量并发执行
func (e *SSHExecutorImpl) BatchExec(hosts []string, cmd string) []ExecResult {
    results := make([]ExecResult, len(hosts))
    var wg sync.WaitGroup

    // 控制并发数
    semaphore := make(chan struct{}, e.config.MaxConcurrent)

    for i, host := range hosts {
        wg.Add(1)
        go func(idx int, h string) {
            defer wg.Done()

            semaphore <- struct{}{}        // 获取信号量
            defer func() { <-semaphore }() // 释放信号量

            start := time.Now()
            output, err := e.Exec(h, cmd)
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
```

### 文件传输

```go
// Upload 上传文件 (使用 SFTP)
func (e *SSHExecutorImpl) Upload(host string, localPath string, remotePath string) error {
    conn, err := e.pool.getConnection(host)
    if err != nil {
        return err
    }

    sftpClient, err := sftp.NewClient(conn)
    if err != nil {
        return err
    }
    defer sftpClient.Close()

    // 读取本地文件
    localFile, err := os.Open(localPath)
    if err != nil {
        return err
    }
    defer localFile.Close()

    // 创建远程文件
    remoteFile, err := sftpClient.Create(remotePath)
    if err != nil {
        return err
    }
    defer remoteFile.Close()

    // 复制内容
    _, err = io.Copy(remoteFile, localFile)
    return err
}
```

## 安全考虑

### 1. 凭证管理

```
┌─────────────────────────────────────────────────────────┐
│                    凭证存储方案                          │
│                                                         │
│  方案一: 数据库加密存储                                  │
│  - 密码/密钥加密后存入数据库                             │
│  - 使用时解密                                           │
│                                                         │
│  方案二: 外部密钥管理                                    │
│  - 对接 Vault / 云密钥服务                              │
│  - 运行时获取凭证                                       │
│                                                         │
│  方案三: SSH Agent                                      │
│  - 使用系统 SSH Agent                                   │
│  - 不存储密钥                                           │
│                                                         │
│  MVP 阶段: 方案一 (简单)                                │
│  生产环境: 方案二 (安全)                                │
└─────────────────────────────────────────────────────────┘
```

### 2. 命令安全

```go
// 危险命令检查
var dangerousCommands = []string{
    "rm -rf /",
    "mkfs",
    "dd if=",
    "> /dev/sda",
    "shutdown",
    "reboot",
}

func (e *SSHExecutorImpl) checkCommand(cmd string) error {
    for _, dangerous := range dangerousCommands {
        if strings.Contains(cmd, dangerous) {
            return fmt.Errorf("危险命令被拦截: %s", cmd)
        }
    }
    return nil
}
```

### 3. 审计日志

```go
// 记录所有执行的命令
type AuditLog struct {
    ID        string
    SessionID string
    Host      string
    Command   string
    User      string
    Output    string
    Error     string
    StartTime time.Time
    EndTime   time.Time
}

func (e *SSHExecutorImpl) ExecWithAudit(ctx *Context, host string, cmd string) (string, error) {
    log := &AuditLog{
        ID:        uuid.New().String(),
        SessionID: ctx.SessionID,
        Host:      host,
        Command:   cmd,
        User:      ctx.User,
        StartTime: time.Now(),
    }

    output, err := e.Exec(host, cmd)

    log.Output = output
    log.EndTime = time.Now()
    if err != nil {
        log.Error = err.Error()
    }

    // 异步保存审计日志
    go e.auditStore.Save(log)

    return output, err
}
```

## 配置示例

```yaml
ssh:
  default_timeout: 30s
  connect_timeout: 10s
  max_connections: 100
  max_concurrent: 20
  keepalive_interval: 30s

  # 默认认证方式
  default_auth:
    type: key
    user: root
    key_path: ~/.ssh/id_rsa

  # 危险命令拦截
  security:
    block_dangerous_commands: true
    audit_enabled: true
```

## 错误处理

```go
// SSH 相关错误类型
var (
    ErrHostNotFound     = errors.New("主机不存在")
    ErrConnectionFailed = errors.New("连接失败")
    ErrAuthFailed       = errors.New("认证失败")
    ErrTimeout          = errors.New("执行超时")
    ErrCommandBlocked   = errors.New("命令被拦截")
)

// 错误包装，便于 Agent 理解
func wrapSSHError(host string, err error) error {
    if errors.Is(err, ErrConnectionFailed) {
        return fmt.Errorf("无法连接到 %s，请检查网络或 SSH 服务", host)
    }
    if errors.Is(err, ErrAuthFailed) {
        return fmt.Errorf("认证失败，请检查 %s 的 SSH 凭证配置", host)
    }
    if errors.Is(err, ErrTimeout) {
        return fmt.Errorf("命令执行超时，%s 可能负载过高或命令耗时过长", host)
    }
    return err
}
```
