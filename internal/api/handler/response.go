package handler

import (
	"net/http"

	"ai-ops/pkg/errors"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse 错误响应结构（带详细错误信息）
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"` // 详细错误信息（仅开发环境）
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    int(errors.CodeSuccess),
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    int(errors.CodeSuccess),
		Message: message,
		Data:    data,
	})
}

// HandleError 统一错误处理
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// 如果是 AppError，使用定义的错误码和消息
	if appErr, ok := err.(*errors.AppError); ok {
		status := appErr.HTTPStatus
		if status == 0 {
			status = http.StatusInternalServerError
		}

		// 开发环境返回详细错误信息
		// 生产环境可以隐藏 detail 字段
		c.JSON(status, ErrorResponse{
			Code:    int(appErr.Code),
			Message: appErr.Message,
			Detail:  appErr.Detail,
		})
		return
	}

	// 普通错误，作为内部错误处理
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    int(errors.CodeInternalError),
		Message: "内部错误",
		Detail:  err.Error(),
	})
}

// Error 错误响应（兼容旧代码）
// Deprecated: 请使用 HandleError 代替
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ParamError 参数错误
// Deprecated: 请使用 errors.ErrParamError 代替
func ParamError(c *gin.Context, message string) {
	Error(c, int(errors.CodeParamError), message)
}

// NotFound 资源不存在
// Deprecated: 请使用 errors.ErrNotFound 代替
func NotFound(c *gin.Context, message string) {
	Error(c, int(errors.CodeNotFound), message)
}

// SSHError SSH 错误
// Deprecated: 请使用 errors.ErrSSHError 代替
func SSHError(c *gin.Context, message string) {
	Error(c, int(errors.CodeSSHError), message)
}

// ExecError 执行错误
// Deprecated: 请使用 errors.ErrExecError 代替
func ExecError(c *gin.Context, message string) {
	Error(c, int(errors.CodeExecError), message)
}

// LLMError LLM 错误
// Deprecated: 请使用 errors.ErrLLMError 代替
func LLMError(c *gin.Context, message string) {
	Error(c, int(errors.CodeLLMError), message)
}

// InternalError 内部错误
// Deprecated: 请使用 errors.ErrInternalError 代替
func InternalError(c *gin.Context, message string) {
	Error(c, int(errors.CodeInternalError), message)
}
