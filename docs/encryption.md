# 敏感数据加密存储

## 概述

系统实现了 AES-256-GCM 加密算法来保护敏感数据（SSH 密码和私钥）的存储。加密数据在写入数据库前自动加密，读取时自动解密。

## 功能特性

- **AES-256-GCM 加密**: 使用业界标准的 AES-256-GCM 算法
- **透明加解密**: 在 Repository 层自动处理，业务层无感知
- **环境变量支持**: 支持通过环境变量配置加密密钥
- **向后兼容**: 自动检测数据是否已加密，支持迁移

## 架构设计

### 1. 加密层 (internal/crypto)

`Encryptor` 提供 AES 加解密功能：

```go
// 创建加密器（需要 32 字节密钥）
encryptor, err := crypto.NewEncryptor(key)

// 从 Base64 编码的密钥创建
encryptor, err := crypto.NewEncryptorFromBase64Key(base64Key)

// 从环境变量创建
encryptor, err := crypto.NewEncryptorFromEnv()
```

### 2. Repository 层 (internal/repository/host)

Repository 层自动处理加解密：

- **Create/Update**: 保存前自动加密 `Password` 和 `KeyContent`
- **GetByID/GetByName/List**: 读取后自动解密

```go
// 创建带加密功能的 Repository
hostRepo := repository.NewHostRepositoryWithEncryption(db, encryptor)
```

### 3. 配置层 (internal/config)

支持环境变量覆盖敏感配置：

```bash
# LLM API Key
export LLM_API_KEY="your-api-key"
export OPENAI_API_KEY="your-api-key"  # 备选

# LLM 配置
export LLM_ENDPOINT="http://localhost:11434/v1"
export LLM_MODEL="qwen2.5:14b"

# 数据库
export DATABASE_DSN="./data/aiops.db"

# 服务器
export SERVER_ADDR=":8080"
export SERVER_MODE="release"

# JWT Secret
export JWT_SECRET="your-jwt-secret"
```

## 使用指南

### 1. 生成加密密钥

```bash
# 使用 Go 代码生成
key, _ := crypto.GenerateKey()
keyBase64, _ := crypto.GenerateKeyBase64()
println(keyBase64)

# 或使用 OpenSSL
openssl rand -base64 32
```

### 2. 配置加密密钥

#### 方式 1: 环境变量（推荐）

```bash
# 使用 Base64 编码的密钥（推荐）
export ENCRYPTION_BASE64_KEY="your-base64-encoded-32-byte-key"

# 或使用原始密钥（会自动处理长度）
export ENCRYPTION_KEY="your-32-byte-encryption-key"
```

#### 方式 2: 代码配置

```go
// main.go
func main() {
    // 从环境变量加载加密密钥
    encryptor, err := crypto.NewEncryptorFromEnv()
    if err != nil {
        logger.Warn("未配置加密密钥，敏感数据将以明文存储", zap.Error(err))
        encryptor = nil
    }

    // 创建带加密功能的 Repository
    hostRepo := repository.NewHostRepositoryWithEncryption(db, encryptor)
}
```

### 3. 初始化应用

更新 `cmd/server/main.go`：

```go
package main

import (
    "ai-ops/internal/crypto"
    "ai-ops/internal/repository"
    // ... 其他导入
)

func main() {
    // ... 初始化数据库等

    // 初始化加密器
    encryptor, err := crypto.NewEncryptorFromEnv()
    if err != nil {
        logger.Warn("未配置加密密钥，敏感数据将以明文存储",
            zap.String("hint", "设置 ENCRYPTION_BASE64_KEY 环境变量启用加密"),
            zap.Error(err))
        // 不使用加密
        hostRepo = repository.NewHostRepository(db)
    } else {
        logger.Info("加密功能已启用", zap.String("algorithm", "AES-256-GCM"))
        hostRepo = repository.NewHostRepositoryWithEncryption(db, encryptor)
    }

    // ... 其他初始化
}
```

## 数据迁移

### 从明文迁移到加密存储

如果数据库中已有明文数据，可以运行迁移脚本：

```go
// migration/encrypt_hosts.go
package main

func migrateHosts(db *gorm.DB, encryptor *crypto.Encryptor) error {
    var hosts []model.Host
    db.Find(&hosts)

    for _, host := range hosts {
        needsUpdate := false

        // 检查并加密密码
        if host.Password != "" && !crypto.IsEncrypted(host.Password) {
            encrypted, _ := encryptor.Encrypt(host.Password)
            host.Password = encrypted
            needsUpdate = true
        }

        // 检查并加密私钥
        if host.KeyContent != "" && !crypto.IsEncrypted(host.KeyContent) {
            encrypted, _ := encryptor.Encrypt(host.KeyContent)
            host.KeyContent = encrypted
            needsUpdate = true
        }

        if needsUpdate {
            db.Save(&host)
        }
    }

    return nil
}
```

## 安全建议

### 1. 密钥管理

- **生产环境**: 使用专业的密钥管理系统（如 AWS KMS、HashiCorp Vault）
- **环境变量**: 不要在 `.env` 文件中提交密钥到版本控制
- **密钥轮换**: 定期轮换加密密钥
- **备份**: 保存加密密钥的安全备份

### 2. 权限控制

```bash
# 限制密钥文件权限
chmod 600 /path/to/secret/key

# 设置环境变量文件权限
chmod 600 .env.production
```

### 3. 审计日志

记录敏感数据访问：

```go
// 在 Repository 层添加审计日志
func (r *hostRepository) GetByID(id string) (*model.Host, error) {
    host, err := // ... 查询逻辑

    // 记录敏感数据访问
    if host.Password != "" || host.KeyContent != "" {
        auditLogger.Log("sensitive_data_access", map[string]interface{}{
            "host_id": id,
            "has_password": host.Password != "",
            "has_key": host.KeyContent != "",
        })
    }

    return host, err
}
```

## 测试

### 单元测试

```bash
# 测试加密器
go test ./internal/crypto -v

# 测试 Repository 加密功能
go test ./internal/repository -v -run TestHostRepository
```

### 集成测试

```bash
# 设置测试密钥
export ENCRYPTION_BASE64_KEY="$(openssl rand -base64 32)"

# 运行集成测试
go test ./... -v
```

## 故障排查

### 问题 1: 解密失败

```
错误: 解密失败: ciphertext decryption failed
```

**原因**: 加密密钥不匹配

**解决方案**:
- 检查 `ENCRYPTION_BASE64_KEY` 或 `ENCRYPTION_KEY` 环境变量
- 确保密钥长度为 32 字节（AES-256）
- 如果更换了密钥，需要重新加密所有数据

### 问题 2: 数据以明文存储

```
警告: 未配置加密密钥，敏感数据将以明文存储
```

**原因**: 未设置加密密钥环境变量

**解决方案**:
```bash
export ENCRYPTION_BASE64_KEY="$(openssl rand -base64 32)"
```

### 问题 3: 部分数据加密，部分数据明文

**原因**: 运行时切换加密配置

**解决方案**:
运行数据迁移脚本，统一所有数据的加密状态

## API 参考

### crypto.Encryptor

```go
type Encryptor struct {
    key []byte
}

// 创建加密器
func NewEncryptor(key []byte) (*Encryptor, error)
func NewEncryptorFromBase64Key(base64Key string) (*Encryptor, error)
func NewEncryptorFromEnv() (*Encryptor, error)

// 加密解密
func (e *Encryptor) Encrypt(plaintext string) (string, error)
func (e *Encryptor) Decrypt(ciphertext string) (string, error)

// 工具函数
func GenerateKey() ([]byte, error)
func GenerateKeyBase64() (string, error)
func IsEncrypted(s string) bool
```

### repository.HostRepository

```go
// 创建 Repository（带加密）
func NewHostRepositoryWithEncryption(
    db *gorm.DB,
    encryptor *crypto.Encryptor,
) HostRepository
```

## 最佳实践

1. **始终启用加密**: 生产环境必须启用数据加密
2. **密钥管理**: 使用专业的密钥管理系统
3. **访问控制**: 限制数据库文件的访问权限
4. **审计日志**: 记录所有敏感数据的访问
5. **密钥轮换**: 定期更换加密密钥
6. **备份加密**: 备份数据时同时备份加密密钥
7. **环境隔离**: 开发、测试、生产使用不同的密钥

## 相关文件

- `internal/crypto/encryptor.go`: AES 加解密实现
- `internal/crypto/encryptor_test.go`: 加密器单元测试
- `internal/repository/host.go`: Repository 加解密集成
- `internal/config/config.go`: 环境变量配置支持
- `internal/service/host_service.go`: 业务逻辑层（支持加密）

## 附录

### AES-256-GCM 算法说明

- **密钥长度**: 256 位 (32 字节)
- **Nonce**: 96 位 (12 字节)，随机生成
- **认证标签**: 128 位 (16 字节)
- **优势**: 同时提供加密和完整性验证

### 性能影响

- **加密开销**: 约 0.1-0.5ms 每次操作
- **内存开销**: 每个加密字段增加约 44 字节（Base64 编码后）
- **建议**: 对于大多数应用，性能影响可忽略不计
