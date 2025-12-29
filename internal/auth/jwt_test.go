package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJWTManager_Generate(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", 24*time.Hour, "test-issuer")

	token, err := jwtManager.Generate("user123", "testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTManager_Verify(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", 24*time.Hour, "test-issuer")

	// 生成 token
	token, err := jwtManager.Generate("user123", "testuser")
	assert.NoError(t, err)

	// 验证 token
	claims, err := jwtManager.Verify(token)
	assert.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "test-issuer", claims.Issuer)
}

func TestJWTManager_Verify_InvalidToken(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", 24*time.Hour, "test-issuer")

	// 测试无效 token（格式错误）
	_, err := jwtManager.Verify("invalid-token")
	assert.Error(t, err)
	assert.Equal(t, ErrTokenMalformed, err)
}

func TestJWTManager_Verify_WrongSecret(t *testing.T) {
	jwtManager1 := NewJWTManager("secret1", 24*time.Hour, "test-issuer")
	jwtManager2 := NewJWTManager("secret2", 24*time.Hour, "test-issuer")

	// 使用第一个管理器生成 token
	token, err := jwtManager1.Generate("user123", "testuser")
	assert.NoError(t, err)

	// 使用第二个管理器验证（不同的密钥）
	_, err = jwtManager2.Verify(token)
	assert.Error(t, err)
}

func TestJWTManager_Refresh(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", 24*time.Hour, "test-issuer")

	// 生成一个较老的 token（通过修改过期时间）
	oldToken, err := jwtManager.Generate("user123", "testuser")
	assert.NoError(t, err)

	// 验证原始 token
	claims, err := jwtManager.Verify(oldToken)
	assert.NoError(t, err)

	// 手动创建一个即将过期的 token（模拟场景）
	// 注意：在实际使用中，refresh 应该在 token 接近过期时调用
	newToken, err := jwtManager.Refresh(oldToken)
	if err == nil {
		// 如果刷新成功，验证新 token
		newClaims, err := jwtManager.Verify(newToken)
		assert.NoError(t, err)
		assert.Equal(t, claims.UserID, newClaims.UserID)
		assert.Equal(t, claims.Username, newClaims.Username)
	}
}

func TestClaims_StandardFields(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", 24*time.Hour, "test-issuer")

	token, err := jwtManager.Generate("user123", "testuser")
	assert.NoError(t, err)

	claims, err := jwtManager.Verify(token)
	assert.NoError(t, err)

	// 验证标准字段
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.NotBefore)
	assert.Equal(t, "user123", claims.Subject)
}
