package config

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

type ConfigManager struct {
	Path  string
	State State
}

func NewConfigManager() (*ConfigManager, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}

	path := filepath.Join(configDir, "crater", "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	cm := &ConfigManager{Path: path}
	if err := cm.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return cm, nil
}

func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.Path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &cm.State)
}

func (cm *ConfigManager) Save() error {
	data, err := json.MarshalIndent(cm.State, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cm.Path, data, 0600)
}
