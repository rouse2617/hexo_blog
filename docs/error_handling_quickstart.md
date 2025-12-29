# 统一错误处理快速开始

## 1. 基本用法（5分钟上手）

### 创建错误

```go
import "ai-ops/pkg/errors"

// 预定义错误
err := errors.ErrParamError

// 自定义消息
err := errors.New(errors.CodeParamError, "用户名不能为空")

// 包装底层错误（保留原始错误用于日志）
err := errors.Wrap(dbErr, errors.CodeDBError, "保存用户失败")

// 添加详细错误信息
err := errors.ErrParamError.WithDetail("field 'email' is required")
```

### Handler 中使用

```go
import (
    "ai-ops/internal/api/handler"
    "ai-ops/pkg/errors"
)

func (h *Handler) GetUser(c *gin.Context) {
    id := c.Param("id")

    // 参数验证
    if id == "" {
        handler.HandleError(c, errors.ErrParamError.WithDetail("id 不能为空"))
        return
    }

    // 查询数据
    user, err := h.repo.GetByID(id)
    if err != nil {
        handler.HandleError(c, errors.Wrap(err, errors.CodeDBError, "查询失败"))
        return
    }

    // 成功响应
    handler.Success(c, user)
}
```

## 2. 常用错误码速查

| 错误码 | 常量名 | 使用场景 | 示例 |
|-------|--------|---------|------|
| 1001 | CodeParamError | 参数错误 | id 为空、格式错误 |
| 1003 | CodeNotFound | 资源不存在 | 查询数据库无结果 |
| 1004 | CodeConflict | 资源冲突 | 重复创建、版本冲突 |
| 2001 | CodeSSHError | SSH 连接失败 | SSH 无法连接 |
| 2004 | CodeExecError | 命令执行失败 | SSH 命令执行出错 |
| 3001 | CodeLLMError | LLM 调用失败 | AI 接口调用失败 |
| 5001 | CodeDBError | 数据库错误 | 数据库操作失败 |
| 9000 | CodeInternalError | 内部错误 | 未预期的错误 |

## 3. 迁移旧代码

### 旧代码
```go
func (h *Handler) GetUser(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        handler.ParamError(c, "id 不能为空")
        return
    }

    user, err := h.repo.GetByID(id)
    if err != nil {
        handler.InternalError(c, "查询失败: "+err.Error())
        return
    }

    handler.Success(c, user)
}
```

### 新代码（推荐）
```go
func (h *Handler) GetUser(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        handler.HandleError(c, errors.ErrParamError.WithDetail("id 不能为空"))
        return
    }

    user, err := h.repo.GetByID(id)
    if err != nil {
        handler.HandleError(c, errors.Wrap(err, errors.CodeDBError, "查询失败"))
        return
    }

    handler.Success(c, user)
}
```

## 4. 判断错误类型

```go
user, err := h.repo.GetByID(id)
if err != nil {
    // 方式1: 判断错误码
    switch errors.GetCode(err) {
    case errors.CodeNotFound:
        handler.HandleError(c, errors.ErrNotFound)
    case errors.CodeDBError:
        handler.HandleError(c, errors.ErrInternalError)
    default:
        handler.HandleError(c, errors.ErrInternalError)
    }
    return
}

// 方式2: 判断是否为 AppError
if errors.IsAppError(err) {
    appErr := err.(*errors.AppError)
    fmt.Printf("Error code: %d, message: %s\n", appErr.Code, appErr.Message)
}
```

## 5. 响应格式

### 成功响应
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "123",
    "name": "张三"
  }
}
```

### 错误响应
```json
{
  "code": 1001,
  "message": "参数错误",
  "detail": "id 不能为空"
}
```

## 6. 完整示例

```go
package handler

import (
    "ai-ops/internal/api/handler"
    "ai-ops/pkg/errors"
    "github.com/gin-gonic/gin"
)

func (h *UserHandler) CreateUser(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        handler.HandleError(c, errors.Wrap(err, errors.CodeParamError, "参数解析失败"))
        return
    }

    // 验证参数
    if req.Name == "" {
        handler.HandleError(c, errors.New(errors.CodeParamError, "用户名不能为空"))
        return
    }

    if req.Email == "" {
        handler.HandleError(c, errors.ErrParamError.WithDetail("email 不能为空"))
        return
    }

    // 检查邮箱是否已存在
    if exists, _ := h.repo.ExistsByEmail(req.Email); exists {
        handler.HandleError(c, errors.ErrConflict.WithDetail("邮箱已被注册"))
        return
    }

    // 创建用户
    user, err := h.repo.Create(req)
    if err != nil {
        handler.HandleError(c, errors.Wrap(err, errors.CodeDBError, "创建用户失败"))
        return
    }

    handler.SuccessWithMessage(c, "创建成功", user)
}
```

## 7. 注册全局错误中间件（推荐）

在 `router.go` 中：

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

## 8. 常见问题

### Q: 旧的 `ParamError()` 等函数还能用吗？
A: 可以，但标记为 Deprecated。建议新代码使用 `HandleError(c, errors.*)`

### Q: 如何在 Service 层使用统一错误？
A: Service 层返回 `*errors.AppError`，Handler 层直接处理

```go
// Service 层
func (s *UserService) GetUser(id string) (*User, error) {
    user, err := s.repo.GetByID(id)
    if err != nil {
        return nil, errors.ErrNotFound.WithDetail("用户不存在")
    }
    return user, nil
}

// Handler 层
func (h *Handler) GetUser(c *gin.Context) {
    user, err := h.service.GetUser(c.Param("id"))
    if err != nil {
        handler.HandleError(c, err)
        return
    }
    handler.Success(c, user)
}
```

### Q: 错误码如何选择？
A: 参考"常用错误码速查"表，根据错误类型选择对应的错误码

## 9. 更多资源

- 完整文档：`docs/error_handling.md`
- 代码示例：`internal/api/handler/errors_example.go`
- 测试代码：`pkg/errors/errors_test.go`

---

**快速开始，就这么简单！** 🚀
