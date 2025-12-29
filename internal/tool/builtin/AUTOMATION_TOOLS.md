# 自动化工具使用指南

本文档介绍新增的两个自动化工具：自动故障恢复工具和告警管理工具。

## 1. 自动故障恢复工具 (auto_recover)

### 功能说明
根据诊断结果自动执行故障恢复操作，支持多种常见故障场景的自动修复。

### 支持的恢复操作

#### 1.1 磁盘满恢复 (disk_full)
清理系统日志、临时文件、包管理器缓存等释放磁盘空间。

**操作内容：**
- 清理 systemd 日志（保留7天）
- 清理 7 天前的临时文件
- 清理包管理器缓存（yum/apt）
- 终止占用已删除文件的进程

**使用示例：**
```json
{
  "host": "server-01",
  "issue_type": "disk_full",
  "confirm": true
}
```

#### 1.2 内存不足恢复 (oom)
清理系统缓存释放内存空间。

**操作内容：**
- 清理 page cache
- 清理 dentry/inode cache
- 清理所有缓存

**使用示例：**
```json
{
  "host": "server-01",
  "issue_type": "oom",
  "confirm": true
}
```

#### 1.3 服务停止恢复 (service_down)
重启已停止的服务并设置开机自启。

**操作内容：**
- 启动指定服务
- 启用开机自启
- 返回服务状态

**使用示例：**
```json
{
  "host": "server-01",
  "issue_type": "service_down",
  "service_name": "nginx",
  "confirm": true
}
```

#### 1.4 网络问题恢复 (network_issue)
重启网络服务恢复网络连接。

**操作内容：**
- 重启网络服务（network/NetworkManager）
- 刷新 DNS 缓存

**使用示例：**
```json
{
  "host": "server-01",
  "issue_type": "network_issue",
  "confirm": true
}
```

#### 1.5 高 CPU 使用率处理 (high_cpu)
列出高 CPU 进程供分析（出于安全考虑不自动终止进程）。

**返回内容：**
- CPU 使用率前 10 的进程列表

**使用示例：**
```json
{
  "host": "server-01",
  "issue_type": "high_cpu",
  "confirm": true
}
```

### 参数说明

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| host | string | 是 | 目标主机名称 |
| issue_type | string | 是 | 问题类型（disk_full/oom/service_down/network_issue/high_cpu） |
| confirm | boolean | 是 | 是否确认执行（必须为 true） |
| service_name | string | 否 | 服务名称（仅 service_down 类型需要） |

### 安全机制
- 所有操作前必须设置 `confirm=true` 确认执行
- 每个操作都记录详细日志
- 关键操作（如终止进程）需要人工确认
- 返回每个操作的执行状态

### 返回结果示例

```json
{
  "success": true,
  "data": {
    "host": "server-01",
    "issue_type": "disk_full",
    "operations": [
      {
        "action": "清理 systemd 日志（保留7天）",
        "cmd": "journalctl --vacuum-time=7d",
        "output": "Vacuuming done, freed 1.2G of archived journals",
        "status": "success"
      }
    ],
    "final_state": "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1        50G   20G   28G  42% /",
    "status": "completed"
  },
  "message": "磁盘空间恢复完成"
}
```

---

## 2. 告警管理工具 (alert_manager)

### 功能说明
创建、删除、查询告警规则，支持多种资源类型的监控告警。

### 支持的告警类型

| 类型 | 说明 | 示例阈值 |
|------|------|----------|
| cpu | CPU 使用率告警 | 80 (%) |
| memory | 内存使用率告警 | 85 (%) |
| disk | 磁盘使用率告警 | 90 (%) |
| process | 进程退出告警 | - |
| log | 日志关键字告警 | - |
| service | 服务停止告警 | - |

### 告警级别

- **critical**: 严重，需要立即处理
- **high**: 高级，1 小时内处理
- **medium**: 中级，当天处理
- **low**: 低级，计划处理

### 操作说明

#### 2.1 创建告警规则

**示例：创建 CPU 使用率告警**
```json
{
  "action": "create",
  "name": "cpu-high",
  "type": "cpu",
  "threshold": 80.0,
  "level": "high"
}
```

**示例：创建磁盘空间告警**
```json
{
  "action": "create",
  "name": "disk-full",
  "type": "disk",
  "threshold": 85.0,
  "level": "critical"
}
```

#### 2.2 删除告警规则

```json
{
  "action": "delete",
  "name": "cpu-high"
}
```

#### 2.3 列出告警规则

```json
{
  "action": "list"
}
```

### 参数说明

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| action | string | 是 | 操作类型（create/delete/list） |
| name | string | 创建时必填 | 告警规则名称 |
| type | string | 创建时必填 | 告警类型 |
| threshold | number | 创建时必填 | 阈值 |
| level | string | 否 | 告警级别（默认 medium） |
| hosts | array | 否 | 监控主机列表 |

### 返回结果示例

#### 创建告警
```json
{
  "success": true,
  "data": {
    "alert": {
      "name": "cpu-high",
      "type": "cpu",
      "threshold": 80.0,
      "level": "high",
      "enabled": true
    },
    "message": "告警规则 cpu-high 创建成功"
  },
  "message": "告警规则 cpu-high 创建成功"
}
```

#### 列出告警
```json
{
  "success": true,
  "data": {
    "alerts": [
      {
        "name": "cpu-high",
        "type": "cpu",
        "threshold": 80.0,
        "level": "high",
        "enabled": true
      },
      {
        "name": "disk-full",
        "type": "disk",
        "threshold": 85.0,
        "level": "critical",
        "enabled": true
      }
    ],
    "count": 2
  },
  "message": "共 2 条告警规则"
}
```

---

## 3. 在 AI 对话中的使用示例

### 场景 1：磁盘空间自动恢复

```
用户: server-01 的磁盘满了，帮我清理一下
AI: 我来帮您清理 server-01 的磁盘空间。

[调用 auto_recover 工具]

✅ 已完成磁盘空间恢复：
• 清理 systemd 日志（保留7天）- 成功
• 清理 7 天前的临时文件 - 成功
• 清理包管理器缓存 - 成功
• 终止占用已删除文件的进程 - 成功

当前磁盘状态：/dev/sda1 50G 20G 28G 42% /
释放了约 2.5GB 空间。
```

### 场景 2：服务自动恢复

```
用户: server-02 的 Nginx 服务停止了
AI: 我来帮您重启 Nginx 服务。

[调用 auto_recover 工具]

✅ 服务恢复完成：
• 启动 nginx 服务 - 成功
• 启用开机自启 - 成功

服务状态：active (running)
```

### 场景 3：创建告警规则

```
用户: 帮我设置一个 CPU 告警，超过 80% 就告警
AI: 我来帮您创建 CPU 使用率告警规则。

[调用 alert_manager 工具]

✅ 告警规则创建成功：
• 规则名称：cpu-high
• 监控类型：CPU 使用率
• 告警阈值：80%
• 告警级别：高级

当 CPU 使用率超过 80% 时会触发告警。
```

---

## 4. 注意事项

### 4.1 安全考虑
- **确认机制**：所有自动恢复操作都需要设置 `confirm=true`
- **日志记录**：所有操作都会记录详细日志
- **人工审核**：高危操作（如终止进程）需要人工确认

### 4.2 最佳实践
- 在生产环境使用前先在测试环境验证
- 定期检查自动恢复操作的日志
- 为关键服务设置合理的告警阈值
- 建议先使用诊断工具确认问题再执行恢复

### 4.3 限制说明
- **high_cpu** 恢复只列出进程，不自动终止
- **service_down** 恢复需要指定服务名称
- 告警规则当前是内存实现，重启后会丢失（需要持久化存储）

---

## 5. 集成到系统

工具已自动注册到工具注册表，无需额外配置。在 AI 对话中可以直接使用：

```go
import "ai-ops/internal/tool/builtin"

// 工具已在 register.go 中自动注册
builtin.RegisterAllEnhanced(registry, getHostsFunc)
```

工具名称：
- `auto_recover` - 自动故障恢复
- `alert_manager` - 告警管理

---

## 6. 测试

运行测试验证工具功能：

```bash
go test -v ./internal/tool/builtin -run TestAutoRecoveryTool
go test -v ./internal/tool/builtin -run TestAlertManagerTool
```

---

## 7. 扩展开发

### 7.1 添加新的恢复类型

在 `auto_recovery.go` 的 `Execute` 方法中添加新的 case：

```go
case "new_issue_type":
    return t.recoverNewIssue(ctx, host)
```

然后实现对应的恢复方法：

```go
func (t *AutoRecoveryTool) recoverNewIssue(ctx *tool.Context, host string) (*tool.Result, error) {
    // 实现恢复逻辑
}
```

### 7.2 持久化告警规则

当前告警规则是内存实现，可以扩展为持久化存储：

```go
// 保存到数据库
func (t *AlertManagerTool) saveToDB(alert Alert) error {
    // 实现
}

// 从数据库加载
func (t *AlertManagerTool) loadFromDB() ([]Alert, error) {
    // 实现
}
```

---

**文档版本**: 1.0
**更新时间**: 2025-12-30
**作者**: AI-Ops Team
