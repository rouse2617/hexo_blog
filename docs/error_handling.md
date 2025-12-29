# 统一错误处理机制使用指南

## 概述

本项目实现了统一的错误处理机制，提供标准化的错误码定义、错误类型和响应格式。

## 核心组件

### 1. 错误类型定义 (`pkg/errors/errors.go`)

#### 错误码分类

```go
// 成功
CodeSuccess = 0

// 客户端错误 1000-1999
CodeParamError      = 1001 // 参数错误
CodeValidationError = 1002 // 验证失败
CodeNotFound        = 1003 // 资源不存在
CodeConflict        = 1004 // 资源冲突
CodeUnauthorized    = 1005 // 未授权
CodeForbidden       = 1006 // 禁止访问

// SSH 相关错误 2000-2999
CodeSSHError      = 2001 // SSH 连接失败
CodeSSHTimeout    = 2002 // SSH 超时
CodeSSHAuthFailed = 2003 // SSH 认证失败
CodeExecError     = 2004 // 命令执行失败

// LLM 相关错误 3000-3999
CodeLLMError     = 3001 // LLM 调用失败
CodeLLMTimeout   = 3002 // LLM 超时

// 工具相关错误 4000-4999
CodeToolNotFound = 4001 // 工具不存在
CodeToolDisabled = 4002 // 工具已禁用
CodeToolExecute  = 4003 // 工具执行失败

// 数据库错误 5000-5999
CodeDBError     = 5001 // 数据库错误
CodeDBConnError = 5002 // 数据库连接错误

// 内部错误 9000-9999
CodeInternalError = 9000 // 内部错误
CodeUnknownError  = 9999 // 未知错误
```

#### AppError 结构

```go
type AppError struct {
    Code       ErrorCode // 错误码
    Message    string    // 错误消息（给用户看）
    Detail     string    // 详细错误信息（开发调试用）
    HTTPStatus int       // HTTP 状态码
    Err        error     // 原始错误（用于日志记录）
}
```

### 2. 响应处理 (`internal/api/handler/response.go`)

#### 响应结构

```go
// 成功响应
type Response struct {
    Code    int         // 错误码（0表示成功）
    Message string      // 响应消息
    Data    interface{} // 响应数据
}

// 错误响应
type ErrorResponse struct {
    Code    int    // 错误码
    Message string // 错误消息
    Detail  string // 详细错误信息（仅开发环境）
}
```

## 使用方法

### 基本用法

#### 1. 创建错误

```go
import "ai-ops/pkg/errors"

// 方式1: 使用预定义错误
err := errors.ErrParamError

// 方式2: 创建新错误
err := errors.New(errors.CodeParamError, "用户名不能为空")

// 方式3: 包装底层错误（保留原始错误用于日志）
err := errors.Wrap(dbErr, errors.CodeDBError, "保存用户失败")

// 方式4: 包装底层错误（支持格式化）
err := errors.Wrapf(dbErr, errors.CodeDBError, "保存 %s 失败", username)

// 方式5: 添加详细错误信息
err := errors.ErrParamError.WithDetail("field 'email' must be valid")

// 方式6: 自定义 HTTP 状态码
err := errors.New(errors.CodeParamError, "invalid").
    WithHTTPStatus(http.StatusUnprocessableEntity)
```

#### 2. Handler 中返回错误

```go
import (
    "ai-ops/internal/api/handler"
    "ai-ops/pkg/errors"
    "github.com/gin-gonic/gin"
)

func (h *UserHandler) CreateUser(c *gin.Context) {
    var req UserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        handler.HandleError(c, errors.Wrap(err, errors.CodeParamError, "参数解析失败"))
        return
    }

    // 参数验证
    if req.Name == "" {
        handler.HandleError(c, errors.New(errors.CodeParamError, "用户名不能为空"))
        return
    }

    // 业务逻辑
    user, err := h.userRepo.Create(req)
    if err != nil {
        handler.HandleError(c, errors.Wrap(err, errors.CodeDBError, "创建用户失败"))
        return
    }

    // 成功响应
    handler.Success(c, user)
}
```

#### 3. 判断错误类型

```go
// 判断是否为 AppError
if errors.IsAppError(err) {
    appErr := err.(*errors.AppError)
    fmt.Println("Error code:", appErr.Code)
    fmt.Println("Error message:", appErr.Message)
}

// 获取错误码
code := errors.GetCode(err)
if code == errors.CodeNotFound {
    // 处理"不存在"的情况
}

// 获取错误消息
message := errors.GetMessage(err)
```

### 高级用法

#### 1. 全局错误处理中间件

在 `router.go` 中注册：

```go
import "ai-ops/internal/api"

func SetupRouter() *gin.Engine {
    r := gin.Default()

    // 注册全局错误处理中间件
    r.Use(api.ErrorHandlerMiddleware())

    // ... 其他路由配置

    return r
}
```

中间件功能：
- 自动捕获 panic
- 统一错误响应格式
- 自动记录错误日志

#### 2. 自定义业务错误

```go
// 在 pkg/errors/ 中定义业务错误
var (
    ErrUserNotFound = errors.New(errors.CodeNotFound, "用户不存在").
                      WithDetail("请检查用户ID是否正确")

    ErrInvalidPassword = errors.New(errors.CodeValidationError, "密码格式错误").
                         WithDetail("密码必须包含字母和数字，长度8-20位")
)

// 使用
if user == nil {
    handler.HandleError(c, ErrUserNotFound)
    return
}
```

#### 3. 错误链追踪

```go
// 底层错误
dbErr := database.ErrDuplicateKey

// 中间层包装
serviceErr := errors.Wrap(dbErr, errors.CodeConflict, "用户名已存在")

// Handler 层再次包装（可选）
handler.HandleError(c, errors.Wrap(serviceErr, errors.CodeConflict, "创建用户失败"))

// 日志记录时可以获取完整错误链
log.Printf("Error: %+v", serviceErr)
// Output: [1004] 创建用户失败: [1004] 用户名已存在: duplicate key
```

### 迁移指南

#### 旧代码（兼容方式）

```go
// 旧代码仍然可以工作
func (h *UserHandler) GetUser(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        handler.ParamError(c, "id 不能为空")
        return
    }

    user, err := h.userRepo.GetByID(id)
    if err != nil {
        handler.InternalError(c, "获取用户失败: "+err.Error())
        return
    }

    handler.Success(c, user)
}
```

#### 新代码（推荐方式）

```go
func (h *UserHandler) GetUser(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        handler.HandleError(c, errors.ErrParamError.WithDetail("id 不能为空"))
        return
    }

    user, err := h.userRepo.GetByID(id)
    if err != nil {
        handler.HandleError(c, errors.Wrap(err, errors.CodeDBError, "获取用户失败"))
        return
    }

    handler.Success(c, user)
}
```

## 最佳实践

### 1. 错误处理原则

- **保留原始错误**：使用 `errors.Wrap` 包装底层错误，便于日志追踪
- **明确错误类型**：使用合适的错误码，不要全部使用 `CodeInternalError`
- **友好的错误消息**：`Message` 面向用户，`Detail` 面向开发者
- **一致的错误处理**：使用 `handler.HandleError` 统一返回错误

### 2. 错误码选择指南

```go
// 参数验证失败 → CodeParamError
if req.Name == "" {
    return errors.ErrParamError.WithDetail("name 不能为空")
}

// 资源不存在 → CodeNotFound
if user == nil {
    return errors.ErrNotFound.WithDetail("用户不存在")
}

// 资源冲突 → CodeConflict
if exists {
    return errors.ErrConflict.WithDetail("用户名已存在")
}

// 数据库错误 → CodeDBError
if err != nil {
    return errors.Wrap(err, errors.CodeDBError, "保存失败")
}

// SSH 相关错误 → CodeSSHError, CodeExecError
if err := ssh.Exec(cmd); err != nil {
    return errors.Wrap(err, errors.CodeExecError, "命令执行失败")
}

// 未预期错误 → CodeInternalError
if err != nil {
    return errors.Wrap(err, errors.CodeInternalError, "处理失败")
}
```

### 3. 日志记录

```go
import "ai-ops/pkg/logger"

// Handler 中记录错误
func (h *UserHandler) DeleteUser(c *gin.Context) {
    id := c.Param("id")

    if err := h.userRepo.Delete(id); err != nil {
        // 记录完整错误信息（包括堆栈）
        logger.Errorf("Failed to delete user %s: %+v", id, err)

        // 返回简化的错误给客户端
        handler.HandleError(c, errors.Wrap(err, errors.CodeDBError, "删除用户失败"))
        return
    }

    handler.SuccessWithMessage(c, "删除成功", nil)
}
```

### 4. 分层错误处理

```go
// Repository 层：返回底层错误
func (r *UserRepo) GetByID(id string) (*User, error) {
    var user User
    err := r.db.Where("id = ?", id).First(&user).Error
    if err != nil {
        // 直接返回数据库错误，不包装
        return nil, err
    }
    return &user, nil
}

// Service 层：包装业务错误
func (s *UserService) GetUser(id string) (*User, error) {
    user, err := s.repo.GetByID(id)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            // 返回业务错误
            return nil, errors.ErrNotFound.WithDetail("用户不存在")
        }
        // 返回数据库错误
        return nil, errors.Wrap(err, errors.CodeDBError, "查询用户失败")
    }
    return user, nil
}

// Handler 层：转换为 HTTP 响应
func (h *UserHandler) GetUser(c *gin.Context) {
    id := c.Param("id")

    user, err := h.service.GetUser(id)
    if err != nil {
        handler.HandleError(c, err)
        return
    }

    handler.Success(c, user)
}
```

## 测试

```go
func TestCreateUser(t *testing.T) {
    tests := []struct {
        name       string
        request    UserRequest
        wantErr    bool
        wantErrCode errors.ErrorCode
    }{
        {
            name: "success",
            request: UserRequest{Name: "test"},
            wantErr: false,
        },
        {
            name: "empty name",
            request: UserRequest{Name: ""},
            wantErr: true,
            wantErrCode: errors.CodeParamError,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := service.CreateUser(tt.request)
            if tt.wantErr {
                if err == nil {
                    t.Fatal("expected error, got nil")
                }
                if errors.GetCode(err) != tt.wantErrCode {
                    t.Errorf("expected code %d, got %d", tt.wantErrCode, errors.GetCode(err))
                }
            } else {
                if err != nil {
                    t.Fatalf("unexpected error: %v", err)
                }
            }
        })
    }
}
```

## API 响应示例

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "123",
    "name": "test"
  }
}
```

### 错误响应

```json
{
  "code": 1001,
  "message": "参数错误",
  "detail": "field 'name' is required"
}
```

### 资源不存在

```json
{
  "code": 1003,
  "message": "资源不存在",
  "detail": "用户 123 不存在"
}
```

## 参考文档

- Go 错误处理最佳实践：https://go.dev/doc/go1.13_error
- Gin 框架错误处理：https://gin-gonic.com/docs/examples/error-binding/
- 项目示例：`internal/api/handler/errors_example.go`
