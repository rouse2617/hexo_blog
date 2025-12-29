package middleware

import (
	"errors"
	"net/http"
	"strings"

	"ai-ops/internal/auth"

	"github.com/gin-gonic/gin"
)

const (
	// UserIDKey 用户 ID 上下文键
	UserIDKey = "user_id"
	// UsernameKey 用户名 上下文键
	UsernameKey = "username"
)

// AuthMiddleware JWT 认证中间件
func AuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "未提供认证令牌",
			})
			c.Abort()
			return
		}

		// 解析 Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "认证令牌格式错误",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 验证 token
		claims, err := jwtManager.Verify(tokenString)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrTokenExpired):
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "认证令牌已过期",
				})
			case errors.Is(err, auth.ErrTokenMalformed):
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "认证令牌格式错误",
				})
			default:
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "认证令牌无效",
				})
			}
			c.Abort()
			return
		}

		// 将用户信息存储到上下文
		c.Set(UserIDKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)

		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件（不强制要求 token）
func OptionalAuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]

		claims, err := jwtManager.Verify(tokenString)
		if err == nil {
			c.Set(UserIDKey, claims.UserID)
			c.Set(UsernameKey, claims.Username)
		}

		c.Next()
	}
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get(UsernameKey)
	if !exists {
		return "", false
	}
	name, ok := username.(string)
	return name, ok
}
