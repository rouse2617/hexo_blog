# 敏感数据加密存储 - 快速开始

## 概述

本系统使用 AES-256-GCM 算法自动加密 SSH 密码和私钥。加密是透明的，业务代码无需修改。

## 快速启用（3 步）

### 1. 生成加密密钥

```bash
# 使用 OpenSSL 生成 32 字节随机密钥（Base64 编码）
openssl rand -base64 32

# 输出示例：
# dGVzdC1rZXktZm9yLWVuY3J5cHRpb24tcHVycG9zZS1vbmx5...
```

### 2. 设置环境变量

```bash
# Linux/macOS
export ENCRYPTION_BASE64_KEY="your-base64-key-here"

# Windows PowerShell
$env:ENCRYPTION_BASE64_KEY="your-base64-key-here"

# Windows CMD
set ENCRYPTION_BASE64_KEY=your-base64-key-here
```

### 3. 启动应用

```bash
# 应用启动时会自动检测并启用加密
go run cmd/server/main.go

# 或
./ai-ops-server
```

看到以下日志表示加密已启用：

```
INFO    加密功能已启用    {"algorithm": "AES-256-GCM"}
```

## 验证加密

### 方式 1: 检查数据库

```bash
# SQLite
sqlite3 data/aiops.db

# 查看主机表
SELECT id, name, password FROM hosts;

# 加密后的密码示例：
# 密码: "my-password-123"
# 加密后: "2YdHvJ9KxN8PlWmRqT3sY6nF4cB0AeD1gH5iK8jMnP2Q=="
```

### 方式 2: API 验证

```bash
# 创建主机（密码会自动加密）
curl -X POST http://localhost:8080/api/v1/hosts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-server",
    "host": "192.168.1.100",
    "auth_type": "password",
    "password": "my-secret-password"
  }'

# 获取主机（密码会自动解密返回，但实际数据库中已加密）
curl http://localhost:8080/api/v1/hosts/test-server
```

## 环境变量参考

### 加密配置

| 环境变量 | 说明 | 必需 |
|---------|------|------|
| `ENCRYPTION_BASE64_KEY` | Base64 编码的 32 字节密钥 | 推荐 |
| `ENCRYPTION_KEY` | 原始密钥（会自动处理长度） | 备选 |

### LLM 配置

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `LLM_API_KEY` | LLM API 密钥 | `sk-xxx` |
| `OPENAI_API_KEY` | OpenAI API 密钥（备选） | `sk-xxx` |
| `LLM_ENDPOINT` | LLM API 端点 | `http://localhost:11434/v1` |
| `LLM_MODEL` | LLM 模型名称 | `qwen2.5:14b` |

### 其他配置

| 环境变量 | 说明 |
|---------|------|
| `DATABASE_DSN` | 数据库连接字符串 |
| `SERVER_ADDR` | 服务器监听地址 |
| `SERVER_MODE` | 运行模式 (debug/release) |
| `JWT_SECRET` | JWT 签名密钥 |

## 常见问题

### Q: 忘记加密密钥怎么办？

A: 无法解密已加密的数据。需要：
1. 备份数据库
2. 重新生成密钥
3. 运行数据迁移脚本

### Q: 如何从明文迁移到加密？

A: 系统会自动检测并迁移：
1. 设置 `ENCRYPTION_BASE64_KEY`
2. 重启应用
3. 下次读取/写入时自动处理

或手动运行迁移脚本（参见 `docs/encryption.md`）

### Q: 加密会影响性能吗？

A: 影响极小（约 0.1-0.5ms/次），可忽略不计

### Q: 开发环境需要加密吗？

A: 不强制，但建议使用以保持一致性

## 安全提示

⚠️ **重要**:
- 不要将 `ENCRYPTION_BASE64_KEY` 提交到版本控制
- 生产环境必须使用强随机密钥
- 定期备份加密密钥和数据库
- 考虑使用专业密钥管理系统（AWS KMS、Vault）

## 示例配置文件

### .env (开发环境)

```bash
# 加密密钥（开发环境）
ENCRYPTION_BASE64_KEY="dGVzdC1kZXZlbG9wbWVudC1rZXk="

# LLM 配置
LLM_API_KEY="sk-test-api-key"
LLM_ENDPOINT="http://localhost:11434/v1"
LLM_MODEL="qwen2.5:14b"

# 服务器
SERVER_ADDR=":8080"
SERVER_MODE="debug"

# 数据库
DATABASE_DSN="./data/aiops.db"
```

### .env.production (生产环境)

```bash
# ⚠️ 生产环境配置（不要提交到版本控制）

# 加密密钥（必须使用强随机密钥）
ENCRYPTION_BASE64_KEY="$(openssl rand -base64 32)"

# LLM 配置
LLM_API_KEY="sk-production-api-key"

# 服务器
SERVER_ADDR=":8080"
SERVER_MODE="release"

# 数据库
DATABASE_DSN="/var/lib/aiops/aiops.db"

# JWT
JWT_SECRET="strong-random-jwt-secret"
```

## 下一步

- 完整文档: `docs/encryption.md`
- API 文档: `docs/api.md`
- 部署指南: `docs/deployment.md`
