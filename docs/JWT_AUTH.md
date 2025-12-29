# JWT 认证授权文档

## 概述

本项目已集成完整的 JWT (JSON Web Token) 认证授权机制，用于保护 API 接口。

## 特性

- 基于 JWT 的无状态认证
- 支持配置化启用/禁用认证（默认禁用以保持向后兼容）
- Token 生成、验证、刷新
- 路由级权限控制
- Bearer Token 认证方式

## 配置

### 1. 修改 `config.yaml`

```yaml
# 认证配置
auth:
  enabled: true                # 启用认证
  secret: "your-secret-key"    # JWT 密钥（建议使用环境变量）
  token_duration: 24h          # Token 有效期
  issuer: "ai-ops"             # Token 签发者
```

### 2. 使用环境变量（推荐）

```bash
export JWT_SECRET="your-production-secret-key"
```

环境变量的优先级高于配置文件。

## API 接口

### 公开接口（无需认证）

- `GET /api/health` - 健康检查
- `POST /api/auth/login` - 用户登录
- `POST /api/auth/refresh` - 刷新 Token
- `POST /api/auth/logout` - 用户登出
- `GET /api/auth/validate` - 验证 Token

### 受保护接口（需要认证）

所有 `/api/*` 下的接口（除公开接口外）都需要认证，包括：

- `/api/chat/*` - 对话相关
- `/api/hosts/*` - 主机管理
- `/api/groups/*` - 分组管理
- `/api/tools/*` - 工具管理
- `/api/system/*` - 系统配置
- `/api/operations/*` - 批量操作
- `/api/analysis/*` - AI 分析

## 使用示例

### 1. 用户登录

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'
```

响应示例：

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": 1735689600,
    "user_id": "admin",
    "username": "admin"
  }
}
```

### 2. 使用 Token 访问受保护接口

```bash
curl -X GET http://localhost:8080/api/hosts \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 3. 刷新 Token

```bash
curl -X POST http://localhost:8080/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

### 4. 验证 Token

```bash
curl -X GET http://localhost:8080/api/auth/validate \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## 默认用户账户

⚠️ **重要提示**：当前版本使用硬编码的演示账户，仅用于开发测试。

| 用户名   | 密码         | 说明     |
|---------|-------------|---------|
| admin   | admin123    | 管理员   |
| operator| operator123 | 操作员   |

**生产环境部署前，必须实现数据库用户验证！**

## 安全建议

### 1. 生产环境配置

```yaml
auth:
  enabled: true
  secret: ""  # 留空，使用环境变量 JWT_SECRET
  token_duration: 8h  # 缩短有效期
  issuer: "ai-ops"
```

### 2. 生成安全的密钥

```bash
# 使用 openssl 生成随机密钥
openssl rand -base64 32

# 或使用 Python
python -c "import secrets; print(secrets.token_urlsafe(32))"
```

### 3. 设置环境变量

```bash
export JWT_SECRET="生成的安全密钥"
```

### 4. HTTPS 部署

生产环境必须使用 HTTPS，防止 Token 被拦截。

## 实现数据库用户验证

当前 `internal/api/handler/auth.go` 中的 `validateCredentials` 方法是硬编码的演示实现。

生产环境需要：

1. 创建用户表（User Model）
2. 实现密码哈希（使用 bcrypt）
3. 实现用户查询逻辑
4. 更新 `validateCredentials` 方法

示例实现：

```go
import "golang.org/x/crypto/bcrypt"

func (h *AuthHandler) validateCredentials(username, password string) bool {
    // 从数据库查询用户
    user, err := h.userRepo.GetByUsername(username)
    if err != nil {
        return false
    }

    // 验证密码
    err = bcrypt.CompareHashAndPassword(
        []byte(user.PasswordHash),
        []byte(password),
    )

    return err == nil
}
```

## 架构说明

### 核心组件

1. **JWT Manager** (`internal/auth/jwt.go`)
   - Token 生成
   - Token 验证
   - Token 刷新

2. **认证中间件** (`internal/api/middleware/auth.go`)
   - 拦截请求
   - 验证 Token
   - 注入用户信息到上下文

3. **认证处理器** (`internal/api/handler/auth.go`)
   - 登录处理
   - Token 刷新
   - Token 验证

### 路由结构

```
/api
├── /health (公开)
├── /auth
│   ├── /login (公开)
│   ├── /refresh (公开)
│   ├── /logout (公开)
│   └── /validate (公开)
├── /chat (需认证)
├── /hosts (需认证)
├── /groups (需认证)
├── /tools (需认证)
├── /system (需认证)
├── /operations (需认证)
└── /analysis (需认证)
```

## 禁用认证

如需临时禁用认证（如内网环境），设置：

```yaml
auth:
  enabled: false
```

所有接口将变为公开访问，无需 Token。

## 前端集成

前端需要：

1. 登录后保存 Token 到 localStorage
2. 每次请求在 Header 中添加 `Authorization: Bearer <token>`
3. 处理 401 错误，引导用户重新登录
4. 在 Token 过期前自动刷新

示例代码：

```javascript
// 登录
const login = async (username, password) => {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password })
  });
  const data = await response.json();
  localStorage.setItem('token', data.data.token);
};

// 带认证的请求
const fetchWithAuth = async (url) => {
  const token = localStorage.getItem('token');
  return fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
};
```

## 故障排查

### 1. Token 无效

- 检查 Token 是否过期
- 检查 JWT_SECRET 是否一致
- 检查 Token 格式是否正确

### 2. 401 Unauthorized

- 确认认证已启用
- 检查 Header 格式：`Authorization: Bearer <token>`
- 验证 Token 是否有效

### 3. 路由不生效

- 确认配置文件加载正确
- 检查 `auth.enabled` 设置
- 查看日志中的认证状态

## 测试

运行 JWT 认证测试：

```bash
go test ./internal/auth/... -v
```

## 后续改进

- [ ] 实现数据库用户管理
- [ ] 添加角色和权限控制（RBAC）
- [ ] Token 黑名单（Redis）
- [ ] 多因素认证（MFA）
- [ ] OAuth 2.0 / OpenID Connect
- [ ] 限流和防暴力破解
- [ ] 会话管理
- [ ] 审计日志增强

## 许可证

MIT License
