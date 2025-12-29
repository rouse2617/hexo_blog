# 入侵检测工具快速参考

## 快速开始

```bash
# 通过 Agent 调用
agent.ExecuteTool("detect_intrusion", map[string]interface{}{
    "host": "192.168.1.100",
})
```

## 检测能力矩阵

| 检测类别 | 检测项 | 风险级别 | 说明 |
|---------|--------|---------|------|
| **登录审计** | 异常时间登录 | Medium | 凌晨 2-6 点登录 |
| | 暴力破解尝试 | High | 失败登录 ≥10 次 |
| | root 用户登录 | High | 监控特权登录 |
| **命令审计** | 危险命令执行 | Critical | rm -rf /、mkfs、dd 等 |
| | Fork bomb | Critical | `:(){:|:&};:` |
| | 可疑脚本下载 | Critical | wget\|sh、curl\|sh |
| **文件完整性** | 异常 SUID 文件 | Critical | bash/sh/cat/vim 等 |
| | 配置文件修改 | Medium | 24 小时内变更 |
| | 临时目录可执行 | High | /tmp 下可执行文件 |
| **后门检测** | 异常端口监听 | High | 非 22/80/443 等 |
| | 可疑定时任务 | Critical | 包含 base64、\|sh 等 |
| | 恶意自启动 | Critical | miner、xmrig 等 |
| **挖矿检测** | 已知挖矿进程 | Critical | xmrig、cpuminer 等 |
| | 高 CPU 低内存 | High | CPU>80%, 内存<5% |

## 评分规则

```
初始分数: 100
Critical: -25 分
High:     -10 分
Medium:   -5 分
最低分数: 0 分
```

## 结果解读

### 优秀 (90-100分)
- 系统安全状况良好
- 少量低风险问题
- 建议：继续保持

### 良好 (70-89分)
- 存在一些中风险问题
- 需要关注和处理
- 建议：定期检查

### 一般 (50-69分)
- 存在较多安全问题
- 有高风险项
- 建议：立即加固

### 危险 (0-49分)
- 严重安全威胁
- 可能已被入侵
- 建议：立即响应

## 应急响应流程

### Critical 级别
1. ⚠️ **立即隔离主机**（断网）
2. 🛑 **终止恶意进程**
   ```bash
   killall -9 [恶意进程名]
   ```
3. 🗑️ **删除恶意文件**
   ```bash
   rm -f [恶意文件路径]
   ```
4. 🔍 **全面审计**（检查其他主机）
5. 🔐 **修改所有密码**
6. 📋 **记录事件详情**

### High 级别
1. 🔒 **加固安全配置**
2. 🚫 **封禁攻击 IP**
   ```bash
   iptables -A INPUT -s [攻击IP] -j DROP
   ```
3. 📊 **持续监控**
4. 📝 **记录处理过程**

### Medium 级别
1. ✅ **确认是否误报**
2. 📧 **通知相关人员**
3. 🔧 **优化配置**
4. 📅 **安排复查**

## 常见处置命令

### 终止进程
```bash
# 按名称终止
killall -9 xmrig

# 按 PID 终止
kill -9 [PID]

# 查找进程
ps aux | grep [进程名]
```

### 删除文件
```bash
# 删除单个文件
rm -f [文件路径]

# 删除目录
rm -rf [目录路径]

# 查找文件
find / -name [文件名]
```

### 禁用服务
```bash
# 停止服务
systemctl stop [服务名]

# 禁用自启动
systemctl disable [服务名]

# 查看服务状态
systemctl status [服务名]
```

### 封禁 IP
```bash
# 使用 iptables
iptables -A INPUT -s [IP] -j DROP
iptables -L -n  # 查看

# 使用 fail2ban
fail2ban-client set [jail] banip [IP]
```

### 删除定时任务
```bash
# 编辑 crontab
crontab -e

# 删除特定行
crontab -l | grep -v '[恶意命令]' | crontab -

# 查看系统定时任务
cat /etc/crontab
```

## 安全加固清单

- [ ] 禁用 root 远程登录
- [ ] 安装并配置 fail2ban
- [ ] 启用防火墙（ufw/iptables）
- [ ] 配置 SSH 密钥认证
- [ ] 禁用密码登录（可选）
- [ ] 更改默认 SSH 端口
- [ ] 安装系统更新
- [ ] 配置日志审计
- [ ] 启用入侵检测系统（IDS）
- [ ] 定期备份重要数据

## 监控建议

### 每日检查
- 系统负载
- 登录记录
- 失败登录
- 网络连接

### 每周检查
- 完整安全审计
- 配置变更
- 用户权限
- 系统更新

### 每月检查
- 安全策略审查
- 访问日志审计
- 备份验证
- 渗透测试

## 误报处理

### 常见误报
1. **高 CPU 进程**：合法的计算任务
2. **异常端口**：自定义服务端口
3. **配置修改**：合法的运维操作
4. **定时任务**：合法的自动化脚本

### 处理方法
1. 确认业务背景
2. 检查操作记录
3. 验证文件签名
4. 咨询相关人员

## 工具集成

### 与监控系统集成
```go
// 定期执行检测
ticker := time.NewTicker(24 * time.Hour)
for range ticker.C {
    result := agent.ExecuteTool("detect_intrusion", params)
    if score < 70 {
        alert.Send("安全评分低于 70 分")
    }
}
```

### 与日志系统集成
```go
// 记录检测结果
logger.Info("安全审计完成",
    "host", host,
    "score", score,
    "findings", len(findings),
)
```

### 与告警系统集成
```go
// 发送告警
if score < 50 {
    alert.Critical("严重安全威胁",
        "host", host,
        "score", score,
    )
}
```

## 最佳实践

1. **分层检测**：结合多种检测手段
2. **基线对比**：建立正常行为基线
3. **及时响应**：Critical 级别立即处理
4. **持续优化**：根据误报调整规则
5. **定期演练**：测试应急响应流程
6. **文档记录**：完整记录处理过程
7. **事后分析**：总结经验教训

## 故障排除

### 问题：部分检测失败
**原因**：权限不足或命令不存在
**解决**：
```bash
# 检查权限
sudo -l

# 安装缺失工具
apt-get install util-linux net-tools
```

### 问题：误报过多
**原因**：检测规则过于严格
**解决**：调整检测阈值，添加白名单

### 问题：执行缓慢
**原因**：网络延迟或主机负载高
**解决**：优化 SSH 连接，设置超时

## 参考资料

- [OWASP 安全最佳实践](https://owasp.org/)
- [CIS 基准](https://www.cisecurity.org/cis-benchmarks/)
- [Linux 安全加固指南](https://lasr.net/)

## 更新日志

### v1.0.0 (2025-12-30)
- ✅ 初始版本
- ✅ 支持 5 大类检测
- ✅ 安全评分系统
- ✅ 详细处置建议
