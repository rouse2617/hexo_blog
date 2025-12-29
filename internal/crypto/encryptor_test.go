package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name     string
		plaintext string
	}{
		{
			name:     "简单字符串",
			plaintext: "hello world",
		},
		{
			name:     "密码",
			plaintext: "MyP@ssw0rd123!",
		},
		{
			name:     "私钥内容",
			plaintext: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyKf7KmFm1Cy2FvJ3O2jVSzYlQYzQJ4Q8zH9K0L1vN2mP3qR5
-----END RSA PRIVATE KEY-----`,
		},
		{
			name:     "空字符串",
			plaintext: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			require.NoError(t, err)

			// 密文不应该与明文相同
			if tt.plaintext != "" {
				assert.NotEqual(t, tt.plaintext, ciphertext)
			}

			// 解密
			decrypted, err := encryptor.Decrypt(ciphertext)
			require.NoError(t, err)

			// 解密后应该与原文相同
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptor_InvalidKeyLength(t *testing.T) {
	invalidKeys := [][]byte{
		{1, 2, 3},              // 3 字节
		make([]byte, 16),       // 16 字节 (AES-128)
		make([]byte, 64),       // 64 字节
	}

	for _, key := range invalidKeys {
		_, err := NewEncryptor(key)
		assert.Error(t, err)
	}
}

func TestEncryptor_InvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	invalidCiphertexts := []string{
		"not base64!",
		"dGVzdA==", // 有效的 Base64 但太短
		"invalid encrypted data",
	}

	for _, ciphertext := range invalidCiphertexts {
		_, err := encryptor.Decrypt(ciphertext)
		assert.Error(t, err)
	}
}

func TestEncryptor_NewEncryptorFromBase64Key(t *testing.T) {
	// 生成密钥
	keyBase64, err := GenerateKeyBase64()
	require.NoError(t, err)

	// 从 Base64 创建加密器
	encryptor, err := NewEncryptorFromBase64Key(keyBase64)
	require.NoError(t, err)

	// 测试加密解密
	plaintext := "test password"
	ciphertext, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestIsEncrypted(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "空字符串",
			input:    "",
			expected: false,
		},
		{
			name:     "普通字符串",
			input:    "hello world",
			expected: false,
		},
		{
			name:     "无效的 Base64",
			input:    "not base64!",
			expected: false,
		},
		{
			name:     "有效的 Base64 但太短",
			input:    "dGVzdA==",
			expected: false,
		},
		{
			name:     "加密后的数据",
			input:    func() string { s, _ := encryptor.Encrypt("test"); return s }(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEncrypted(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	require.NoError(t, err)
	assert.Equal(t, 32, len(key1))

	key2, err := GenerateKey()
	require.NoError(t, err)
	assert.Equal(t, 32, len(key2))

	// 两次生成的密钥应该不同
	assert.NotEqual(t, key1, key2)
}

func TestGenerateKeyBase64(t *testing.T) {
	keyBase64, err := GenerateKeyBase64()
	require.NoError(t, err)

	// 应该是有效的 Base64
	key, err := NewEncryptorFromBase64Key(keyBase64)
	require.NoError(t, err)
	assert.NotNil(t, key)
}
