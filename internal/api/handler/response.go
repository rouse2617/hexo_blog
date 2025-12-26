package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错误码定义
const (
	CodeSuccess       = 0    // 成功
	CodeParamError    = 1001 // 参数错误
	CodeNotFound      = 1002 // 资源不存在
	CodeSSHError      = 2001 // SSH 连接失败
	CodeExecError     = 2002 // 命令执行失败
	CodeLLMError      = 3001 // LLM 调用失败
	CodeInternalError = 5000 // 内部错误
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ParamError 参数错误
func ParamError(c *gin.Context, message string) {
	Error(c, CodeParamError, message)
}

// NotFound 资源不存在
func NotFound(c *gin.Context, message string) {
	Error(c, CodeNotFound, message)
}

// SSHError SSH 错误
func SSHError(c *gin.Context, message string) {
	Error(c, CodeSSHError, message)
}

// ExecError 执行错误
func ExecError(c *gin.Context, message string) {
	Error(c, CodeExecError, message)
}

// LLMError LLM 错误
func LLMError(c *gin.Context, message string) {
	Error(c, CodeLLMError, message)
}

// InternalError 内部错误
func InternalError(c *gin.Context, message string) {
	Error(c, CodeInternalError, message)
}
