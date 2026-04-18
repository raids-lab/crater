package auth

import (
	"fmt"

	"github.com/99designs/keyring"
)

const (
	serviceName = "crater-cli"
)

// AuthManager handles token storage using the system keyring
type AuthManager struct {
	ring keyring.Keyring
}

// NewAuthManager initializes the keyring manager
func NewAuthManager() (*AuthManager, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}
	return &AuthManager{ring: ring}, nil
}

// StoreToken saves the token for a specific platform and account
func (am *AuthManager) StoreToken(platformID, accountID, token string) error {
	key := fmt.Sprintf("%s:%s", platformID, accountID)
	return am.ring.Set(keyring.Item{
		Key:  key,
		Data: []byte(token),
	})
}

// GetToken retrieves the token for a specific platform and account
func (am *AuthManager) GetToken(platformID, accountID string) (string, error) {
	key := fmt.Sprintf("%s:%s", platformID, accountID)
	item, err := am.ring.Get(key)
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

// RemoveToken deletes the token for a specific platform and account
func (am *AuthManager) RemoveToken(platformID, accountID string) error {
	key := fmt.Sprintf("%s:%s", platformID, accountID)
	return am.ring.Remove(key)
}
