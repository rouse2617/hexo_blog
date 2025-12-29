package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 错误码类型
type ErrorCode int

// 标准错误码定义
const (
	// 成功
	CodeSuccess ErrorCode = 0

	// 客户端错误 1000-1999
	CodeParamError    ErrorCode = 1001 // 参数错误
	CodeValidationError ErrorCode = 1002 // 验证失败
	CodeNotFound      ErrorCode = 1003 // 资源不存在
	CodeConflict      ErrorCode = 1004 // 资源冲突
	CodeUnauthorized  ErrorCode = 1005 // 未授权
	CodeForbidden     ErrorCode = 1006 // 禁止访问
	CodeRateLimit     ErrorCode = 1007 // 请求过于频繁

	// SSH 相关错误 2000-2999
	CodeSSHError      ErrorCode = 2001 // SSH 连接失败
	CodeSSHTimeout    ErrorCode = 2002 // SSH 超时
	CodeSSHAuthFailed ErrorCode = 2003 // SSH 认证失败
	CodeExecError     ErrorCode = 2004 // 命令执行失败

	// LLM 相关错误 3000-3999
	CodeLLMError      ErrorCode = 3001 // LLM 调用失败
	CodeLLMTimeout    ErrorCode = 3002 // LLM 超时
	CodeLLMRateLimit  ErrorCode = 3003 // LLM 请求过于频繁

	// 工具相关错误 4000-4999
	CodeToolNotFound  ErrorCode = 4001 // 工具不存在
	CodeToolDisabled  ErrorCode = 4002 // 工具已禁用
	CodeToolExecute   ErrorCode = 4003 // 工具执行失败

	// 数据库错误 5000-5999
	CodeDBError       ErrorCode = 5001 // 数据库错误
	CodeDBConnError   ErrorCode = 5002 // 数据库连接错误

	// 内部错误 9000-9999
	CodeInternalError ErrorCode = 9000 // 内部错误
	CodeUnknownError  ErrorCode = 9999 // 未知错误
)

// AppError 应用错误类型
type AppError struct {
	Code    ErrorCode   `json:"code"`              // 错误码
	Message string      `json:"message"`           // 错误消息（给用户看）
	Detail  string      `json:"detail,omitempty"`  // 详细错误信息（开发调试用）
	HTTPStatus int      `json:"-"`                 // HTTP 状态码
	Err     error       `json:"-"`                 // 原始错误（用于日志记录）
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
	}
}

// Wrap 包装已有错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
		Err:        err,
	}
}

// Wrapf 包装已有错误（支持格式化）
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: getHTTPStatus(code),
		Err:        err,
	}
}

// WithDetail 添加详细错误信息
func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail
	return e
}

// WithHTTPStatus 设置 HTTP 状态码
func (e *AppError) WithHTTPStatus(status int) *AppError {
	e.HTTPStatus = status
	return e
}

// getHTTPStatus 根据错误码获取对应的 HTTP 状态码
func getHTTPStatus(code ErrorCode) int {
	switch {
	case code == CodeSuccess:
		return http.StatusOK
	case code >= 1000 && code < 2000:
		// 客户端错误
		switch code {
		case CodeUnauthorized:
			return http.StatusUnauthorized
		case CodeForbidden:
			return http.StatusForbidden
		case CodeNotFound:
			return http.StatusNotFound
		case CodeConflict:
			return http.StatusConflict
		default:
			return http.StatusBadRequest
		}
	case code >= 2000 && code < 9000:
		// 服务端业务错误，返回 200，通过 code 字段区分
		return http.StatusOK
	case code >= 9000:
		// 内部错误
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// 预定义的常用错误
var (
	ErrParamError    = New(CodeParamError, "参数错误")
	ErrValidationError = New(CodeValidationError, "验证失败")
	ErrNotFound      = New(CodeNotFound, "资源不存在")
	ErrConflict      = New(CodeConflict, "资源冲突")
	ErrUnauthorized  = New(CodeUnauthorized, "未授权")
	ErrForbidden     = New(CodeForbidden, "禁止访问")

	ErrSSHError      = New(CodeSSHError, "SSH 连接失败")
	ErrSSHTimeout    = New(CodeSSHTimeout, "SSH 连接超时")
	ErrSSHAuthFailed = New(CodeSSHAuthFailed, "SSH 认证失败")
	ErrExecError     = New(CodeExecError, "命令执行失败")

	ErrLLMError      = New(CodeLLMError, "LLM 调用失败")
	ErrLLMTimeout    = New(CodeLLMTimeout, "LLM 调用超时")

	ErrToolNotFound  = New(CodeToolNotFound, "工具不存在")
	ErrToolDisabled  = New(CodeToolDisabled, "工具已禁用")

	ErrDBError       = New(CodeDBError, "数据库错误")
	ErrInternalError = New(CodeInternalError, "内部错误")
)

// IsAppError 判断是否为 AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetCode 获取错误码
func GetCode(err error) ErrorCode {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return CodeUnknownError
}

// GetMessage 获取错误消息
func GetMessage(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Message
	}
	return "未知错误"
}
