package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AuthInfo 代表一组保存的认证凭据
type AuthInfo struct {
	PlatformURL string `json:"platform_url"`
	Username    string `json:"username"`
	Method      string `json:"method"` // ldap | normal
	UserID      int    `json:"user_id"`
	Nickname    string `json:"nickname"`
	Role        string `json:"role"` // 默认角色
}

// ActiveContext 当前激活的认证环境
type ActiveContext struct {
	PlatformURL string `json:"platform_url"`
	Username    string `json:"username"`
	Method      string `json:"method"`
}

// State 存储在 state.json 中的完整结构
type State struct {
	AuthInfos     []AuthInfo    `json:"auth_infos"`
	ActiveContext ActiveContext `json:"active_context"`
	Language      string        `json:"language"` // en | zh-CN
}

// Manager manages reading/writing state.json.
type Manager struct {
	Path  string
	State State
}

func NewManager() (*Manager, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}

	path := filepath.Join(configDir, "crater", "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	m := &Manager{Path: path}
	if err := m.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return m, nil
}

func (m *Manager) Load() error {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.State)
}

func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.State, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.Path, data, 0600)
}
