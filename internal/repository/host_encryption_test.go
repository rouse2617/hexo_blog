package repository

import (
	"ai-ops/internal/crypto"
	"ai-ops/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostRepository_Encryption(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建加密器
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	encryptor, err := crypto.NewEncryptor(key)
	require.NoError(t, err)

	// 创建带加密的 Repository
	repo := NewHostRepositoryWithEncryption(db, encryptor)

	// 测试数据
	testPassword := "MySecretPassword123!"
	testKeyContent := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyKf7KmFm1Cy2FvJ3O2jVSzYlQYzQJ4Q8zH9K0L1vN2mP3qR5
-----END RSA PRIVATE KEY-----`

	host := &model.Host{
		Name:       "test-host",
		IP:         "192.168.1.100",
		Port:       22,
		User:       "root",
		AuthType:   model.AuthTypePassword,
		Password:   testPassword,
		KeyContent: testKeyContent,
		Status:     model.HostStatusUnknown,
	}

	// 1. 测试创建（应该加密）
	t.Run("Create - 应该加密敏感数据", func(t *testing.T) {
		err := repo.Create(host)
		require.NoError(t, err)

		// 直接查询数据库，验证数据已加密
		var dbHost model.Host
		err = db.Where("name = ?", host.Name).First(&dbHost).Error
		require.NoError(t, err)

		// 密码应该被加密
		assert.NotEqual(t, testPassword, dbHost.Password)
		assert.True(t, crypto.IsEncrypted(dbHost.Password),
			"密码应该被加密")

		// 私钥应该被加密
		assert.NotEqual(t, testKeyContent, dbHost.KeyContent)
		assert.True(t, crypto.IsEncrypted(dbHost.KeyContent),
			"私钥应该被加密")
	})

	// 2. 测试读取（应该解密）
	t.Run("GetByID - 应该解密敏感数据", func(t *testing.T) {
		fetchedHost, err := repo.GetByID(host.ID)
		require.NoError(t, err)

		// 密码应该被解密
		assert.Equal(t, testPassword, fetchedHost.Password,
			"密码应该被正确解密")

		// 私钥应该被解密
		assert.Equal(t, testKeyContent, fetchedHost.KeyContent,
			"私钥应该被正确解密")
	})

	// 3. 测试更新
	t.Run("Update - 应该保持加密", func(t *testing.T) {
		// 先获取主机，避免重复插入
		fetchedHost, err := repo.GetByID(host.ID)
		require.NoError(t, err)

		newPassword := "NewPassword456!"
		fetchedHost.Password = newPassword

		err = repo.Update(fetchedHost)
		require.NoError(t, err)

		// 重新获取
		updatedHost, err := repo.GetByID(host.ID)
		require.NoError(t, err)

		// 新密码应该被正确加密和解密
		assert.Equal(t, newPassword, updatedHost.Password)

		// 私钥应该保持不变
		assert.Equal(t, testKeyContent, updatedHost.KeyContent)
	})

	// 4. 测试列表查询
	t.Run("List - 应该解密所有主机的敏感数据", func(t *testing.T) {
		hosts, err := repo.List(HostFilter{})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(hosts), 1)

		// 找到测试主机（注意：密码可能已在 Update 测试中被修改）
		var found *model.Host
		for _, h := range hosts {
			if h.ID == host.ID {
				found = h
				break
			}
		}

		require.NotNil(t, found)
		// 密钥内容应该保持不变
		assert.Equal(t, testKeyContent, found.KeyContent)
		// 密码应该能正确解密（不管是原值还是更新后的值）
		assert.NotEmpty(t, found.Password)
		assert.False(t, crypto.IsEncrypted(found.Password), "返回的密码应该是解密后的")
	})
}

func TestHostRepository_NoEncryption(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建不带加密的 Repository
	repo := NewHostRepository(db)

	testPassword := "PlainPassword123!"

	host := &model.Host{
		Name:     "plain-host",
		IP:       "192.168.1.101",
		Password: testPassword,
	}

	// 创建主机
	err := repo.Create(host)
	require.NoError(t, err)

	// 读取主机
	fetchedHost, err := repo.GetByID(host.ID)
	require.NoError(t, err)

	// 密码应该保持明文
	assert.Equal(t, testPassword, fetchedHost.Password)

	// 直接查询数据库验证
	var dbHost model.Host
	err = db.Where("id = ?", host.ID).First(&dbHost).Error
	require.NoError(t, err)

	// 应该是明文（不是加密格式）
	assert.Equal(t, testPassword, dbHost.Password)
	assert.False(t, crypto.IsEncrypted(dbHost.Password))
}

func TestHostRepository_MixedEncryption(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	encryptor, err := crypto.NewEncryptor(key)
	require.NoError(t, err)

	repo := NewHostRepositoryWithEncryption(db, encryptor)

	// 场景: 数据库中已有明文数据
	plainHost := &model.Host{
		Name:     "existing-plain-host",
		IP:       "192.168.1.200",
		Password: "ExistingPlainPassword",
	}

	// 直接插入数据库（绕过 Repository，模拟已有数据）
	err = db.Create(plainHost).Error
	require.NoError(t, err)

	// 通过 Repository 读取时，应该保持明文（因为无法识别是否已加密）
	fetched, err := repo.GetByID(plainHost.ID)
	require.NoError(t, err)
	assert.Equal(t, "ExistingPlainPassword", fetched.Password)

	// 更新该主机，应该触发加密
	fetched.Password = "UpdatedPassword"
	err = repo.Update(fetched)
	require.NoError(t, err)

	// 验证数据库中已加密
	var dbHost model.Host
	err = db.Where("id = ?", plainHost.ID).First(&dbHost).Error
	require.NoError(t, err)
	assert.True(t, crypto.IsEncrypted(dbHost.Password))

	// 再次读取应该正确解密
	reloaded, err := repo.GetByID(plainHost.ID)
	require.NoError(t, err)
	assert.Equal(t, "UpdatedPassword", reloaded.Password)
}
