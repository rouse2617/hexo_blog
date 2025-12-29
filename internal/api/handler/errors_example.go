//go:build ignore
// +build ignore

// 这是一个示例文件，展示如何使用新的错误处理机制
// 此文件不会被编译到实际项目中

package handler

import (
	"ai-ops/pkg/errors"

	"github.com/gin-gonic/gin"
)

// ========================================
// 示例 1: 基本错误处理
// ========================================
func ExampleBasicErrorHandling(c *gin.Context) {
	// 场景 1: 参数验证失败
	id := c.Param("id")
	if id == "" {
		// 使用预定义错误
		HandleError(c, errors.ErrParamError.WithDetail("id 不能为空"))
		return
	}

	// 场景 2: 资源不存在
	user, err := getUserByID(id)
	if err != nil {
		// 使用 Wrap 保留原始错误
		HandleError(c, errors.Wrap(err, errors.CodeNotFound, "用户不存在"))
		return
	}

	// 场景 3: 业务逻辑错误
	if user.Status != "active" {
		// 创建新错误
		HandleError(c, errors.New(errors.CodeValidationError, "用户未激活"))
		return
	}

	// 成功响应
	Success(c, user)
}

// ========================================
// 示例 2: 数据库操作错误处理
// ========================================
func ExampleDatabaseErrorHandling(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, errors.Wrap(err, errors.CodeParamError, "请求参数解析失败"))
		return
	}

	// 参数验证
	if req.Name == "" {
		HandleError(c, errors.New(errors.CodeParamError, "用户名不能为空"))
		return
	}

	// 创建用户
	user, err := createUserInDB(req)
	if err != nil {
		// 包装数据库错误
		HandleError(c, errors.Wrap(err, errors.CodeDBError, "创建用户失败"))
		return
	}

	Success(c, user)
}

// ========================================
// 示例 3: SSH 操作错误处理
// ========================================
func ExampleSSHErrorHandling(c *gin.Context) {
	hostID := c.Param("host_id")

	// 检查主机是否存在
	host, err := getHostByID(hostID)
	if err != nil {
		if errors.IsAppError(err) && errors.GetCode(err) == errors.CodeNotFound {
			HandleError(c, errors.ErrNotFound.WithDetail("主机不存在"))
		} else {
			HandleError(c, errors.Wrap(err, errors.CodeDBError, "查询主机失败"))
		}
		return
	}

	// 执行 SSH 命令
	output, err := executeSSHCommand(hostID, "uptime")
	if err != nil {
		HandleError(c, errors.Wrap(err, errors.CodeExecError, "命令执行失败"))
		return
	}

	Success(c, gin.H{
		"output": output,
	})
}

// ========================================
// 示例 4: 复杂业务逻辑错误处理
// ========================================
func ExampleComplexBusinessLogic(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
		Action string `json:"action"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, errors.Wrap(err, errors.CodeParamError, "参数解析失败"))
		return
	}

	// 参数验证
	if req.UserID == "" {
		HandleError(c, errors.New(errors.CodeParamError, "user_id 不能为空"))
		return
	}

	if req.Action == "" {
		HandleError(c, errors.New(errors.CodeParamError, "action 不能为空"))
		return
	}

	// 检查用户是否存在
	user, err := getUserByID(req.UserID)
	if err != nil {
		HandleError(c, errors.ErrNotFound.WithDetail("用户不存在"))
		return
	}

	// 业务规则验证
	if !user.IsActive() {
		HandleError(c, errors.New(errors.CodeValidationError, "用户未激活，无法执行操作"))
		return
	}

	// 执行操作
	switch req.Action {
	case "delete":
		if err := deleteUser(user); err != nil {
			HandleError(c, errors.Wrap(err, errors.CodeDBError, "删除用户失败"))
			return
		}
	case "suspend":
		if err := suspendUser(user); err != nil {
			HandleError(c, errors.Wrap(err, errors.CodeDBError, "暂停用户失败"))
			return
		}
	default:
		HandleError(c, errors.New(errors.CodeParamError, "不支持的操作: "+req.Action))
		return
	}

	SuccessWithMessage(c, "操作成功", nil)
}

// ========================================
// 示例 5: 使用错误码判断
// ========================================
func ExampleErrorCodeCheck(c *gin.Context) {
	user, err := getUserByID("123")
	if err != nil {
		// 判断错误类型
		switch errors.GetCode(err) {
		case errors.CodeNotFound:
			HandleError(c, errors.ErrNotFound.WithDetail("用户不存在"))
		case errors.CodeDBError:
			HandleError(c, errors.ErrInternalError.WithDetail("数据库查询失败"))
		default:
			HandleError(c, errors.ErrInternalError)
		}
		return
	}

	Success(c, user)
}

// ========================================
// 示例 6: LLM 调用错误处理
// ========================================
func ExampleLLMErrorHandling(c *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, errors.Wrap(err, errors.CodeParamError, "参数解析失败"))
		return
	}

	// 调用 LLM
	response, err := callLLM(req.Message)
	if err != nil {
		// 判断超时错误
		if errors.IsAppError(err) && errors.GetCode(err) == errors.CodeLLMTimeout {
			HandleError(c, errors.ErrLLMTimeout.WithDetail("LLM 响应超时，请稍后重试"))
			return
		}

		// 其他 LLM 错误
		HandleError(c, errors.Wrap(err, errors.CodeLLMError, "LLM 调用失败"))
		return
	}

	Success(c, response)
}

// ========================================
// 辅助函数（仅用于示例编译）
// ========================================
type CreateUserRequest struct {
	Name string `json:"name"`
}

type User struct {
	ID     string
	Name   string
	Status string
}

func (u *User) IsActive() bool {
	return u.Status == "active"
}

func getUserByID(id string) (*User, error) {
	return nil, errors.ErrNotFound
}

func getHostByID(id string) (interface{}, error) {
	return nil, errors.ErrNotFound
}

func createUserInDB(req CreateUserRequest) (*User, error) {
	return nil, errors.ErrDBError
}

func executeSSHCommand(hostID, cmd string) (string, error) {
	return "", errors.ErrExecError
}

func deleteUser(user *User) error {
	return errors.ErrDBError
}

func suspendUser(user *User) error {
	return errors.ErrDBError
}

func callLLM(message string) (string, error) {
	return "", errors.ErrLLMError
}

// ========================================
// 错误处理最佳实践总结
// ========================================
/*
使用建议：

1. **参数验证错误**（CodeParamError, CodeValidationError）
   - 使用 errors.ErrParamError 或 errors.New 创建
   - 添加 Detail 字段说明具体哪个参数有问题

2. **资源不存在错误**（CodeNotFound）
   - 使用 errors.ErrNotFound
   - 资源查询失败时返回

3. **资源冲突错误**（CodeConflict）
   - 使用 errors.ErrConflict
   - 如：重复创建、版本冲突等

4. **数据库错误**（CodeDBError）
   - 使用 errors.Wrap 包装底层错误
   - 保留原始错误用于日志记录

5. **SSH 相关错误**（CodeSSHError, CodeExecError）
   - 使用对应的错误类型
   - 包装底层 SSH 错误

6. **内部错误**（CodeInternalError）
   - 使用 errors.ErrInternalError
   - 未预期的错误

错误处理模式：

// 模式1: 简单错误
HandleError(c, errors.ErrParamError)

// 模式2: 带详细信息的错误
HandleError(c, errors.ErrParamError.WithDetail("field 'name' is required"))

// 模式3: 包装底层错误
HandleError(c, errors.Wrap(err, errors.CodeDBError, "保存失败"))

// 模式4: 格式化错误消息
HandleError(c, errors.Wrapf(err, errors.CodeDBError, "保存 %s 失败", name))

// 模式5: 判断错误类型
if errors.IsAppError(err) && errors.GetCode(err) == errors.CodeNotFound {
    // 处理"不存在"的情况
}

迁移指南：

旧代码：
    ParamError(c, "参数错误")
    NotFound(c, "资源不存在")
    InternalError(c, "内部错误: "+err.Error())

新代码：
    HandleError(c, errors.ErrParamError)
    HandleError(c, errors.ErrNotFound)
    HandleError(c, errors.Wrap(err, errors.CodeInternalError, "操作失败"))
*/
