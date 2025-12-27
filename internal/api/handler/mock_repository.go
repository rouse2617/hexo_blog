package handler

import (
	"ai-ops/internal/model"
	"ai-ops/internal/repository"
	"strings"
	"time"

	"gorm.io/gorm"
)

// mockSessionRepository mock会话仓库
type mockSessionRepository struct {
	sessions map[string]*model.Session
	messages map[string][]*model.Message
}

func newMockSessionRepository() repository.SessionRepository {
	return &mockSessionRepository{
		sessions: make(map[string]*model.Session),
		messages: make(map[string][]*model.Message),
	}
}

func (m *mockSessionRepository) Create(session *model.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionRepository) GetByID(id string) (*model.Session, error) {
	session, ok := m.sessions[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return session, nil
}

func (m *mockSessionRepository) List(limit int) ([]*model.Session, error) {
	sessions := make([]*model.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}
	return sessions, nil
}

func (m *mockSessionRepository) Delete(id string) error {
	delete(m.sessions, id)
	delete(m.messages, id)
	return nil
}

func (m *mockSessionRepository) AddMessage(sessionID string, msg *model.Message) error {
	if m.messages[sessionID] == nil {
		m.messages[sessionID] = make([]*model.Message, 0)
	}
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return nil
}

func (m *mockSessionRepository) GetMessages(sessionID string) ([]*model.Message, error) {
	messages, ok := m.messages[sessionID]
	if !ok {
		return []*model.Message{}, nil
	}
	return messages, nil
}

func (m *mockSessionRepository) UpdateTitle(sessionID string, title string) error {
	session, ok := m.sessions[sessionID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	session.Title = title
	session.UpdatedAt = time.Now()
	return nil
}

// mockHostRepository mock主机仓库
type mockHostRepository struct {
	hosts map[string]*model.Host
}

func newMockHostRepository() repository.HostRepository {
	return &mockHostRepository{
		hosts: make(map[string]*model.Host),
	}
}

func (m *mockHostRepository) Create(host *model.Host) error {
	m.hosts[host.ID] = host
	return nil
}

func (m *mockHostRepository) Update(host *model.Host) error {
	m.hosts[host.ID] = host
	return nil
}

func (m *mockHostRepository) Delete(id string) error {
	delete(m.hosts, id)
	return nil
}

func (m *mockHostRepository) GetByID(id string) (*model.Host, error) {
	host, ok := m.hosts[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return host, nil
}

func (m *mockHostRepository) GetByName(name string) (*model.Host, error) {
	for _, host := range m.hosts {
		if host.Name == name {
			return host, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockHostRepository) List(filter repository.HostFilter) ([]*model.Host, error) {
	hosts := make([]*model.Host, 0, len(m.hosts))
	for _, host := range m.hosts {
		// 应用过滤条件
		if filter.Group != "" && host.Group != filter.Group {
			continue
		}
		if filter.Status != "" && host.Status != filter.Status {
			continue
		}
		if filter.Keyword != "" {
			keyword := strings.ToLower(filter.Keyword)
			if !strings.Contains(strings.ToLower(host.Name), keyword) &&
				!strings.Contains(strings.ToLower(host.IP), keyword) &&
				!strings.Contains(strings.ToLower(host.User), keyword) {
				continue
			}
		}
		hosts = append(hosts, host)
	}
	return hosts, nil
}

func (m *mockHostRepository) UpdateStatus(id string, status string) error {
	host, ok := m.hosts[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	host.Status = status
	return nil
}

// mockGroupRepository mock分组仓库
type mockGroupRepository struct {
	groups map[string]*model.Group
}

func newMockGroupRepository() repository.GroupRepository {
	return &mockGroupRepository{
		groups: make(map[string]*model.Group),
	}
}

func (m *mockGroupRepository) Create(group *model.Group) error {
	m.groups[group.ID] = group
	return nil
}

func (m *mockGroupRepository) GetByID(id string) (*model.Group, error) {
	group, ok := m.groups[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return group, nil
}

func (m *mockGroupRepository) GetByName(name string) (*model.Group, error) {
	for _, group := range m.groups {
		if group.Name == name {
			return group, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockGroupRepository) List() ([]*model.Group, error) {
	groups := make([]*model.Group, 0, len(m.groups))
	for _, group := range m.groups {
		groups = append(groups, group)
	}
	return groups, nil
}

func (m *mockGroupRepository) Delete(id string) error {
	delete(m.groups, id)
	return nil
}

func (m *mockGroupRepository) DeleteByName(name string) error {
	for id, group := range m.groups {
		if group.Name == name {
			delete(m.groups, id)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

// mockConfigRepository mock配置仓库
type mockConfigRepository struct {
	configs map[string]*model.Config
}

func newMockConfigRepository() repository.ConfigRepository {
	return &mockConfigRepository{
		configs: make(map[string]*model.Config),
	}
}

func (m *mockConfigRepository) Get(key string) (*model.Config, error) {
	config, ok := m.configs[key]
	if !ok {
		return nil, nil
	}
	return config, nil
}

func (m *mockConfigRepository) Set(key string, value string) error {
	m.configs[key] = &model.Config{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	return nil
}

func (m *mockConfigRepository) GetAll() (map[string]string, error) {
	result := make(map[string]string)
	for key, config := range m.configs {
		result[key] = config.Value
	}
	return result, nil
}

