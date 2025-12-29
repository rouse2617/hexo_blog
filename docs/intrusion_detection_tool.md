# 入侵检测工具使用指南

## 概述

入侵检测工具（Intrusion Detection Tool）是一个全面的安全审计工具，能够自动检测 Linux 系统中的多种安全威胁和异常行为。

## 工具名称

- **工具标识**: `detect_intrusion`
- **文件位置**: `internal/tool/builtin/intrusion_detection.go`

## 功能特性

### 1. 登录审计
- ✅ 最近登录记录分析
- ✅ 异常时间登录检测（凌晨 2-6 点）
- ✅ 暴力破解尝试检测（失败登录统计）
- ✅ root 用户登录监控

### 2. 命令审计
- ✅ root 历史命令分析
- ✅ 危险命令检测（rm -rf /、mkfs、dd 等）
- ✅ Fork bomb 检测
- ✅ 可疑脚本下载执行检测

### 3. 文件完整性检查
- ✅ SUID/SGID 文件扫描
- ✅ 异常 SUID 文件检测（bash、sh、cat、vim 等）
- ✅ 配置文件变更检测（最近 24 小时）
- ✅ 临时目录可执行文件检测

### 4. 后门检测
- ✅ 异常端口监听检测
- ✅ 可疑定时任务检测
- ✅ 异常启动项检测
- ✅ 已知恶意服务检测

### 5. 挖矿病毒检测
- ✅ 已知挖矿进程检测（xmrig、cpuminer、kinsing 等）
- ✅ 高 CPU 低内存进程检测
- ✅ 矿池连接检测

## 使用方法

### 基本调用

```go
// 通过 AI Agent 调用
response := agent.ExecuteTool("detect_intrusion", map[string]interface{}{
    "host": "192.168.1.100",
})
```

### 参数说明

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| host | string | 是 | 目标主机 IP 地址或主机名 |

## 返回结果

### 结果结构

```json
{
  "host": "192.168.1.100",
  "score": 75,
  "findings": [
    {
      "category": "login",
      "level": "high",
      "item": "暴力破解尝试",
      "detail": "IP 192.168.1.50 失败 25 次",
      "advice": "建议封禁相关 IP，启用 fail2ban"
    },
    {
      "category": "command",
      "level": "critical",
      "item": "危险命令执行",
      "detail": "发现匹配: rm -rf /",
      "advice": "立即检查命令是否合法"
    }
  ],
  "summary": "安全审计发现以下问题：\n- critical级: 1 项\n- high级: 2 项\n- medium级: 3 项"
}
```

### 字段说明

#### score - 安全评分
- **范围**: 0-100
- **评分规则**:
  - 初始分数: 100 分
  - Critical 级别: -25 分
  - High 级别: -10 分
  - Medium 级别: -5 分
  - 最低分数: 0 分

#### findings - 风险项列表
每个风险项包含以下字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| category | string | 风险类别：login/command/file/backdoor/miner |
| level | string | 风险级别：critical/high/medium/low |
| item | string | 风险项目名称 |
| detail | string | 详细证据信息 |
| advice | string | 处置建议 |
| action | string | 可选，建议执行的命令 |

#### summary - 安全摘要
- 无风险时返回："未发现安全问题，系统安全"
- 有风险时返回各级别风险统计

## 风险级别说明

| 级别 | 说明 | 示例 |
|------|------|------|
| **Critical** | 严重威胁，需立即处理 | 挖矿进程、可疑 SUID 文件、后门 |
| **High** | 高风险，应尽快处理 | 暴力破解、root 登录、临时目录可执行文件 |
| **Medium** | 中等风险，需要关注 | 异常时间登录、配置文件修改 |
| **Low** | 低风险，建议检查 | 一般性警告 |

## 使用场景

### 1. 日常安全巡检
```bash
# 每日自动检测所有主机
for host in $(cat hosts.txt); do
    agent.ExecuteTool("detect_intrusion", map[string]interface{}{
        "host": host,
    })
done
```

### 2. 事件响应
```bash
# 系统异常时快速审计
agent.ExecuteTool("detect_intrusion", map[string]interface{}{
    "host": "suspicious-host.com",
})
```

### 3. 新主机入网检查
```bash
# 新服务器上线前的安全检查
agent.ExecuteTool("detect_intrusion", map[string]interface{}{
    "host": "new-server.example.com",
})
```

## 处置建议

### Critical 级别处理
1. **立即终止恶意进程**
   ```bash
   killall -9 xmrig
   ```

2. **删除可疑文件**
   ```bash
   rm -f /tmp/suspicious.exe
   ```

3. **检查系统启动项**
   ```bash
   systemctl disable suspicious-service
   ```

4. **修改所有密码**

5. **检查网络连接**
   ```bash
   netstat -antp | grep ESTABLISHED
   ```

### High 级别处理
1. **启用 fail2ban**
   ```bash
   apt-get install fail2ban
   systemctl enable fail2ban
   ```

2. **封禁攻击 IP**
   ```bash
   iptables -A INPUT -s 192.168.1.50 -j DROP
   ```

3. **审查 root 登录记录**
   ```bash
   last | grep root
   ```

### Medium 级别处理
1. **确认配置变更是否合法**
2. **审查异常时间登录**
3. **加强监控**

## 安全加固建议

### 1. 禁用 root 远程登录
```bash
sed -i 's/PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config
systemctl restart sshd
```

### 2. 启用 fail2ban
```bash
apt-get install fail2ban
systemctl enable fail2ban
systemctl start fail2ban
```

### 3. 定期更新系统
```bash
apt-get update && apt-get upgrade -y
```

### 4. 配置防火墙
```bash
ufw enable
ufw default deny incoming
ufw allow 22/tcp
```

### 5. 启用审计日志
```bash
apt-get install auditd
systemctl enable auditd
systemctl start auditd
```

## 性能考虑

- **执行时间**: 约 10-30 秒（取决于主机性能）
- **网络开销**: 中等（多次 SSH 命令执行）
- **系统负载**: 低（使用系统命令，轻量级检测）

## 注意事项

1. **权限要求**
   - 需要 root 权限执行某些检测
   - 建议使用具有 sudo 权限的账户

2. **误报处理**
   - 某些检测可能产生误报（如合法的高 CPU 进程）
   - 需要人工确认后再采取行动

3. **日志记录**
   - 所有检测结果都会被记录
   - 建议定期审查历史记录

4. **定期扫描**
   - 建议每日或每周定期执行
   - 可配合 cron 任务实现自动化

## 故障排除

### 问题：无法读取日志
- **原因**: 权限不足
- **解决**: 使用具有 sudo 权限的账户

### 问题：某些检测失败
- **原因**: 命令不存在（如 lastb）
- **解决**: 安装相应工具包

### 问题：检测结果为空
- **原因**: 系统过于干净或命令执行失败
- **解决**: 检查 SSH 连接和系统环境

## 扩展开发

如需添加新的检测规则，可以在对应的检测方法中添加逻辑：

```go
// 在 detectMiner 方法中添加新的挖矿进程名
minerProcesses := []string{
    "xmrig",
    "cpuminer",
    "your-new-miner-name", // 添加新的进程名
}
```

## 相关文档

- [工具开发指南](../tool/README.md)
- [Agent 架构说明](../agent/README.md)
- [安全最佳实践](SECURITY_BEST_PRACTICES.md)

## 许可证

本工具是 AI-Ops 系统的一部分，遵循项目许可证。
