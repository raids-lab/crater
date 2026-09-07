package session

import (
	"errors"
	"fmt"

	"github.com/raids-lab/crater/cli/internal/state"
)

// ErrNoToken indicates the matching auth info exists without a persisted token,
// or no matching auth info is saved.
var ErrNoToken = errors.New("no token saved for these credentials")

func accountKey(ac state.ActiveContext) string {
	return fmt.Sprintf("%s|%s|%s", ac.PlatformURL, ac.Username, ac.Method)
}

// LoadToken reads the access token from state.json for the given active context.
func LoadToken(ac state.ActiveContext) (string, error) {
	if ac.PlatformURL == "" || ac.Username == "" || ac.Method == "" {
		return "", fmt.Errorf("active context is empty")
	}
	if testSessionEnabled() {
		return fakeTokenFor(ac), nil
	}
	st, err := LoadState()
	if err != nil {
		return "", err
	}
	info, ok := AuthInfoFor(st, ac)
	if !ok || info.Token == "" {
		return "", ErrNoToken
	}
	return info.Token, nil
}
