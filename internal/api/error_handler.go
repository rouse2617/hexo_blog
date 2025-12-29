package api

import (
	"log"
	"runtime/debug"

	"ai-ops/internal/api/handler"
	"ai-ops/pkg/errors"
	"ai-ops/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ErrorHandlerMiddleware 全局错误处理中间件
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用 defer recover 捕获 panic
		defer func() {
			if r := recover(); r != nil {
				// 记录 panic 堆栈信息
				log.Printf("PANIC: %v\n%s", r, debug.Stack())

				// 返回内部错误响应
				handler.HandleError(c, errors.ErrInternalError.WithDetail("服务器内部错误"))
				c.Abort()
			}
		}()

		// 执行请求
		c.Next()

		// 检查是否有错误
		if len(c.Errors) > 0 {
			// 获取最后一个错误
			err := c.Errors.Last().Err

			// 记录错误日志
			logger.Errorf("Request error: %s %s - %v", c.Request.Method, c.Request.URL.Path, err)

			// 如果还没有响应，则处理错误
			if !c.Writer.Written() {
				handler.HandleError(c, err)
			}
		}
	}
}

// HandlePanic 处理 panic 的辅助函数
func HandlePanic(c *gin.Context) {
	if r := recover(); r != nil {
		log.Printf("PANIC: %v\n%s", r, debug.Stack())
		handler.HandleError(c, errors.ErrInternalError.WithDetail("服务器内部错误"))
		c.Abort()
	}
}

// WrapError 包装错误并添加到 Gin 的错误列表
func WrapError(c *gin.Context, err error) {
	if err != nil {
		c.Error(err)
	}
}

// WrapErrorf 包装错误并添加到 Gin 的错误列表（支持格式化）
func WrapErrorf(c *gin.Context, format string, args ...interface{}) {
	err := errors.ErrInternalError.WithDetail(format)
	c.Error(err)
}
