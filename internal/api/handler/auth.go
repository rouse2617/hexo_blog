package handler

import (
	"net/http"
	"time"

	"ai-ops/internal/auth"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	jwtManager *auth.JWTManager
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		jwtManager: jwtManager,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
}

// RefreshTokenRequest 刷新 token 请求
type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// TODO: 实际生产环境中，这里应该：
	// 1. 从数据库查询用户
	// 2. 使用 bcrypt 验证密码
	// 3. 检查用户状态是否正常

	// 示例：简单硬编码验证（仅用于演示）
	// 生产环境应该从数据库验证
	if !h.validateCredentials(req.Username, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户名或密码错误",
		})
		return
	}

	// 生成 JWT token
	token, err := h.jwtManager.Generate(req.Username, req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "生成令牌失败",
		})
		return
	}

	// 计算过期时间（24小时后）
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登录成功",
		"data": LoginResponse{
			Token:     token,
			ExpiresAt: expiresAt,
			UserID:    req.Username,
			Username:  req.Username,
		},
	})
}

// RefreshToken 刷新 token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 刷新 token
	newToken, err := h.jwtManager.Refresh(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "令牌刷新失败: " + err.Error(),
		})
		return
	}

	// 计算过期时间
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "令牌刷新成功",
		"data": LoginResponse{
			Token:     newToken,
			ExpiresAt: expiresAt,
		},
	})
}

// ValidateToken 验证 token
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未提供认证令牌",
		})
		return
	}

	// 移除 "Bearer " 前缀
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	claims, err := h.jwtManager.Verify(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "令牌无效: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "令牌有效",
		"data": gin.H{
			"user_id":  claims.UserID,
			"username": claims.Username,
			"expires":  claims.ExpiresAt.Unix(),
		},
	})
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT 是无状态的，登出主要在前端删除 token
	// 如果需要实现强制失效，可以使用 Redis 黑名单
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登出成功",
	})
}

// validateCredentials 验证用户凭据
// TODO: 生产环境中应该从数据库验证，使用 bcrypt
func (h *AuthHandler) validateCredentials(username, password string) bool {
	// 示例：硬编码的演示账户
	// 生产环境必须从数据库验证并使用 bcrypt
	validUsers := map[string]string{
		"admin":     "admin123",
		"operator":  "operator123",
	}

	storedPassword, exists := validUsers[username]
	if !exists {
		return false
	}

	return password == storedPassword
}
