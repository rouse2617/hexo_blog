package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrTokenInvalid token 无效
	ErrTokenInvalid = errors.New("token 无效")
	// ErrTokenExpired token 已过期
	ErrTokenExpired = errors.New("token 已过期")
	// ErrTokenMalformed token 格式错误
	ErrTokenMalformed = errors.New("token 格式错误")
)

// Claims JWT 声明
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTManager JWT 管理器
type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
	issuer        string
}

// NewJWTManager 创建 JWT 管理器
func NewJWTManager(secretKey string, tokenDuration time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
		issuer:        issuer,
	}
}

// Generate 生成 JWT token
func (m *JWTManager) Generate(userID, username string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.tokenDuration)

	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Verify 验证 JWT token
func (m *JWTManager) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(m.secretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, ErrTokenMalformed
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// Refresh 刷新 token
func (m *JWTManager) Refresh(tokenString string) (string, error) {
	claims, err := m.Verify(tokenString)
	if err != nil {
		return "", err
	}

	// 检查是否在刷新时间窗口内（例如原过期时间的 30 分钟内）
	if time.Until(claims.ExpiresAt.Time) > 30*time.Minute {
		return "", errors.New("token 还未到刷新时间")
	}

	// 生成新 token
	return m.Generate(claims.UserID, claims.Username)
}
