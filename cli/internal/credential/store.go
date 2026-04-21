package credential

import (
	"fmt"

	"github.com/99designs/keyring"
)

const serviceName = "crater-cli"

// Store 在系统 Keyring 中存取与账号关联的 token（与 HTTP 登录、Cobra 子命令 auth 解耦）。
type Store struct {
	ring keyring.Keyring
}

// NewStore 打开本 CLI 使用的 Keyring 后端。
func NewStore() (*Store, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}
	return &Store{ring: ring}, nil
}

// StoreToken 将 token 写入 Keyring（platformID、accountID 由调用方约定，如 crater + platform|user|mode）。
func (s *Store) StoreToken(platformID, accountID, token string) error {
	key := fmt.Sprintf("%s:%s", platformID, accountID)
	return s.ring.Set(keyring.Item{
		Key:  key,
		Data: []byte(token),
	})
}

// GetToken 从 Keyring 读取 token。
func (s *Store) GetToken(platformID, accountID string) (string, error) {
	key := fmt.Sprintf("%s:%s", platformID, accountID)
	item, err := s.ring.Get(key)
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

// RemoveToken 从 Keyring 删除 token。
func (s *Store) RemoveToken(platformID, accountID string) error {
	key := fmt.Sprintf("%s:%s", platformID, accountID)
	return s.ring.Remove(key)
}
