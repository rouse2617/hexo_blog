package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	// ErrInvalidCiphertext 密文格式错误
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")
	// ErrInvalidBlockSize 块大小错误
	ErrInvalidBlockSize = errors.New("invalid block size")
)

// Encryptor AES 加密器
type Encryptor struct {
	key []byte
}

// NewEncryptor 创建加密器
// key: 32字节的加密密钥
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("加密密钥必须是 32 字节，当前长度: %d", len(key))
	}
	return &Encryptor{key: key}, nil
}

// NewEncryptorFromBase64Key 从 Base64 编码的密钥创建加密器
func NewEncryptorFromBase64Key(base64Key string) (*Encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("解码 Base64 密钥失败: %w", err)
	}
	return NewEncryptor(key)
}

// NewEncryptorFromEnv 从环境变量创建加密器
// 优先使用 ENCRYPTION_KEY，其次使用 ENCRYPTION_BASE64_KEY
func NewEncryptorFromEnv() (*Encryptor, error) {
	// 优先尝试 Base64 编码的密钥
	if base64Key := os.Getenv("ENCRYPTION_BASE64_KEY"); base64Key != "" {
		return NewEncryptorFromBase64Key(base64Key)
	}

	// 尝试原始密钥
	if key := os.Getenv("ENCRYPTION_KEY"); key != "" {
		// 如果密钥长度不是 32 字节，进行填充或截断
		keyBytes := []byte(key)
		if len(keyBytes) < 32 {
			// 填充到 32 字节
			padded := make([]byte, 32)
			copy(padded, keyBytes)
			keyBytes = padded
		} else if len(keyBytes) > 32 {
			// 截断到 32 字节
			keyBytes = keyBytes[:32]
		}
		return NewEncryptor(keyBytes)
	}

	return nil, errors.New("未设置环境变量 ENCRYPTION_KEY 或 ENCRYPTION_BASE64_KEY")
}

// Encrypt 加密明文
// 使用 AES-256-GCM 算法
// 返回 Base64 编码的密文
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("创建 cipher 失败: %w", err)
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 模式失败: %w", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 返回 Base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密密文
// 密文应为 Base64 编码的字符串
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("Base64 解码失败: %w", err)
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("创建 cipher 失败: %w", err)
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 模式失败: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	// 分离 nonce 和密文
	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// GenerateKey 生成随机加密密钥
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256 需要 32 字节
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("生成密钥失败: %w", err)
	}
	return key, nil
}

// GenerateKeyBase64 生成 Base64 编码的随机加密密钥
func GenerateKeyBase64() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// IsEncrypted 判断字符串是否为加密数据
// 通过检查是否为有效的 Base64 编码且长度合理来判断
func IsEncrypted(s string) bool {
	if s == "" {
		return false
	}

	// 尝试 Base64 解码
	_, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return false
	}

	// 加密后的数据应该至少包含 nonce (12 bytes) + tag (16 bytes) = 28 bytes
	// Base64 编码后至少 38 个字符
	return len(s) >= 38
}
