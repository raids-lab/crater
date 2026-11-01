package session

import (
	"errors"
	"fmt"

	"github.com/raids-lab/crater/cli/internal/state"
)

var (
	// ErrAuthInfoNotFound indicates that no saved auth info matches the active context.
	ErrAuthInfoNotFound = errors.New("no saved auth info matches the active context")
	// ErrNoToken indicates that the matching auth info exists without a persisted token.
	ErrNoToken = errors.New("no token saved for these credentials")
)

func accountKey(ac state.ActiveContext) string {
	return fmt.Sprintf("%s|%s|%s", ac.PlatformURL, ac.Username, ac.Method)
}

// LoadToken returns the access token for the given active context from st.
func LoadToken(st state.State, ac state.ActiveContext) (string, error) {
	if ac.PlatformURL == "" || ac.Username == "" || ac.Method == "" {
		return "", ErrAuthInfoNotFound
	}
	if testSessionEnabled() {
		return fakeTokenFor(ac), nil
	}
	info, ok := AuthInfoFor(st, ac)
	if !ok {
		return "", ErrAuthInfoNotFound
	}
	if info.Token == "" {
		return "", ErrNoToken
	}
	return info.Token, nil
}

// ReplaceToken updates one saved identity without changing the active identity.
func ReplaceToken(ac ActiveContext, token string) error {
	if token == "" {
		return ErrNoToken
	}
	if testSessionEnabled() {
		return nil
	}
	st, err := LoadState()
	if err != nil {
		return err
	}
	if _, ok := AuthInfoFor(st, ac); !ok {
		return ErrAuthInfoNotFound
	}
	for i := range st.AuthInfos {
		info := &st.AuthInfos[i]
		if info.PlatformURL == ac.PlatformURL && info.Username == ac.Username && info.Method == ac.Method {
			info.Token = token
			return SaveState(st)
		}
	}
	return ErrAuthInfoNotFound
}
