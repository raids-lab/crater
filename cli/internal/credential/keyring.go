package credential

import (
	"fmt"

	"github.com/99designs/keyring"
)

const keyringServiceName = "crater-cli"

// Keyring 在系统 Keyring 中存取与账号关联的 token（与 HTTP 登录、Cobra 子命令 auth 解耦）。
type Keyring struct {
	ring keyring.Keyring
}

// NewKeyring 打开本 CLI 使用的 Keyring 后端。
func NewKeyring() (*Keyring, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: keyringServiceName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}
	return &Keyring{ring: ring}, nil
}

// StoreToken 将 token 写入 Keyring（platformID、accountID 由调用方约定，如 crater + platform|user|mode）。
func (k *Keyring) StoreToken(platformID, accountID, token string) error {
	key := fmt.Sprintf("%s:%s", platformID, accountID)
	return k.ring.Set(keyring.Item{
		Key:  key,
		Data: []byte(token),
	})
}

// GetToken 从 Keyring 读取 token。
func (k *Keyring) GetToken(platformID, accountID string) (string, error) {
	key := fmt.Sprintf("%s:%s", platformID, accountID)
	item, err := k.ring.Get(key)
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

// RemoveToken 从 Keyring 删除 token。
func (k *Keyring) RemoveToken(platformID, accountID string) error {
	key := fmt.Sprintf("%s:%s", platformID, accountID)
	return k.ring.Remove(key)
}
