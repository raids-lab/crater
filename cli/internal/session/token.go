package session

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/credential"
	"github.com/raids-lab/crater/cli/internal/state"
)

// KeyringAccountKey is the stable accountID used for keyring storage.
// It must stay consistent with the auth login command's write/delete behavior.
func KeyringAccountKey(ac state.ActiveContext) string {
	return fmt.Sprintf("%s|%s|%s", ac.PlatformURL, ac.Username, ac.Method)
}

// LoadToken reads the access token from OS keyring for the given active context.
func LoadToken(ac state.ActiveContext) (string, error) {
	if ac.PlatformURL == "" || ac.Username == "" || ac.Method == "" {
		return "", fmt.Errorf("active context is empty")
	}
	if testSessionEnabled() {
		return fakeTokenFor(ac), nil
	}
	k, err := credential.NewKeyring()
	if err != nil {
		return "", err
	}
	return k.GetToken("crater", KeyringAccountKey(ac))
}

// SaveToken writes the access token into OS keyring for the given active context.
func SaveToken(ac state.ActiveContext, token string) error {
	if ac.PlatformURL == "" || ac.Username == "" || ac.Method == "" {
		return fmt.Errorf("active context is empty")
	}
	if testSessionEnabled() {
		return nil
	}
	k, err := credential.NewKeyring()
	if err != nil {
		return err
	}
	return k.StoreToken("crater", KeyringAccountKey(ac), token)
}

// DeleteToken removes the access token from OS keyring for the given active context.
func DeleteToken(ac state.ActiveContext) error {
	if ac.PlatformURL == "" || ac.Username == "" || ac.Method == "" {
		return fmt.Errorf("active context is empty")
	}
	if testSessionEnabled() {
		return nil
	}
	k, err := credential.NewKeyring()
	if err != nil {
		return err
	}
	return k.RemoveToken("crater", KeyringAccountKey(ac))
}
