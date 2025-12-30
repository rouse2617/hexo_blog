package repository

import (
	"ai-ops/internal/crypto"
	"ai-ops/internal/model"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// HostRepository 主机仓库接口
type HostRepository interface {
	Create(host *model.Host) error
	Update(host *model.Host) error
	Delete(id string) error
	GetByID(id string) (*model.Host, error)
	GetByName(name string) (*model.Host, error)
	List(filter HostFilter) ([]*model.Host, error)
	UpdateStatus(id string, status string) error
}

// HostFilter 主机过滤条件
type HostFilter struct {
	Group   string
	Keyword string
	Status  string
	Tags    []string
}

// hostRepository 主机仓库实现
type hostRepository struct {
	db        *gorm.DB
	encryptor *crypto.Encryptor
}

// NewHostRepository 创建主机仓库
func NewHostRepository(db *gorm.DB) HostRepository {
	return &hostRepository{db: db, encryptor: nil}
}

// NewHostRepositoryWithEncryption 创建带加密的主机仓库
func NewHostRepositoryWithEncryption(db *gorm.DB, encryptor *crypto.Encryptor) HostRepository {
	return &hostRepository{db: db, encryptor: encryptor}
}

// setEncryptor 设置加密器（可选）
func (r *hostRepository) setEncryptor(encryptor *crypto.Encryptor) {
	r.encryptor = encryptor
}

// Create 创建主机
func (r *hostRepository) Create(host *model.Host) error {
	// 保存前加密敏感数据
	if err := r.encryptBeforeSave(host); err != nil {
		return err
	}
	return r.db.Create(host).Error
}

// Update 更新主机
func (r *hostRepository) Update(host *model.Host) error {
	// 保存前加密敏感数据
	if err := r.encryptBeforeSave(host); err != nil {
		return err
	}
	return r.db.Model(&model.Host{}).Where("id = ?", host.ID).Updates(host).Error
}

// Delete 删除主机
func (r *hostRepository) Delete(id string) error {
	return r.db.Delete(&model.Host{}, "id = ?", id).Error
}

// GetByID 根据ID获取主机
func (r *hostRepository) GetByID(id string) (*model.Host, error) {
	var host model.Host
	err := r.db.Where("id = ?", id).First(&host).Error
	if err != nil {
		return nil, err
	}
	// 读取后解密敏感数据
	if err := r.decryptAfterLoad(&host); err != nil {
		return nil, err
	}
	return &host, nil
}

// GetByName 根据名称获取主机
func (r *hostRepository) GetByName(name string) (*model.Host, error) {
	var host model.Host
	err := r.db.Where("name = ?", name).First(&host).Error
	if err != nil {
		return nil, err
	}
	// 读取后解密敏感数据
	if err := r.decryptAfterLoad(&host); err != nil {
		return nil, err
	}
	return &host, nil
}

// List 列表查询
func (r *hostRepository) List(filter HostFilter) ([]*model.Host, error) {
	var hosts []*model.Host
	query := r.db.Model(&model.Host{})

	if filter.Group != "" {
		query = query.Where("`group` = ?", filter.Group)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR ip LIKE ? OR user LIKE ?", keyword, keyword, keyword)
	}
	if len(filter.Tags) > 0 {
		// SQLite JSON 数组过滤：检查 tags JSON 数组中是否包含任一指定标签
		// 使用 JSON_EACH 函数展开 JSON 数组并检查值（OR 逻辑：匹配任一标签）
		tagPlaceholders := make([]string, len(filter.Tags))
		tagArgs := make([]interface{}, len(filter.Tags))
		for i, tag := range filter.Tags {
			tagPlaceholders[i] = "?"
			tagArgs[i] = tag
		}
		// 构建查询：检查 tags JSON 数组中是否包含任一指定标签
		query = query.Where(`EXISTS (
			SELECT 1 FROM json_each(tags) WHERE value IN (`+strings.Join(tagPlaceholders, ",")+`)
		)`, tagArgs...)
	}

	err := query.Order("created_at DESC").Find(&hosts).Error
	if err != nil {
		return nil, fmt.Errorf("查询主机列表失败: %w", err)
	}

	// 读取后批量解密敏感数据（优化：减少解密调用的开销）
	if r.encryptor != nil {
		for _, host := range hosts {
			if err := r.decryptAfterLoad(host); err != nil {
				return nil, fmt.Errorf("解密主机数据失败: %w", err)
			}
		}
	}

	return hosts, nil
}

// UpdateStatus 更新主机状态
func (r *hostRepository) UpdateStatus(id string, status string) error {
	return r.db.Model(&model.Host{}).Where("id = ?", id).Update("status", status).Error
}

// ContainsKeyword 检查主机是否包含关键字（用于内存过滤）
func ContainsKeyword(host *model.Host, keyword string) bool {
	if keyword == "" {
		return true
	}
	keyword = strings.ToLower(keyword)
	return strings.Contains(strings.ToLower(host.Name), keyword) ||
		strings.Contains(strings.ToLower(host.IP), keyword) ||
		strings.Contains(strings.ToLower(host.User), keyword)
}

// encryptBeforeSave 保存前加密敏感数据
func (r *hostRepository) encryptBeforeSave(host *model.Host) error {
	if r.encryptor == nil {
		return nil
	}

	// 加密密码
	if host.Password != "" {
		// 检查是否已经加密
		if !crypto.IsEncrypted(host.Password) {
			encrypted, err := r.encryptor.Encrypt(host.Password)
			if err != nil {
				return err
			}
			host.Password = encrypted
		}
	}

	// 加密私钥内容
	if host.KeyContent != "" {
		// 检查是否已经加密
		if !crypto.IsEncrypted(host.KeyContent) {
			encrypted, err := r.encryptor.Encrypt(host.KeyContent)
			if err != nil {
				return err
			}
			host.KeyContent = encrypted
		}
	}

	return nil
}

// decryptAfterLoad 读取后解密敏感数据
func (r *hostRepository) decryptAfterLoad(host *model.Host) error {
	if r.encryptor == nil {
		return nil
	}

	// 解密密码
	if host.Password != "" {
		// 检查是否为加密数据
		if crypto.IsEncrypted(host.Password) {
			decrypted, err := r.encryptor.Decrypt(host.Password)
			if err != nil {
				return err
			}
			host.Password = decrypted
		}
	}

	// 解密私钥内容
	if host.KeyContent != "" {
		// 检查是否为加密数据
		if crypto.IsEncrypted(host.KeyContent) {
			decrypted, err := r.encryptor.Decrypt(host.KeyContent)
			if err != nil {
				return err
			}
			host.KeyContent = decrypted
		}
	}

	return nil
}

