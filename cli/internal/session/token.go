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
	// ErrSessionChanged indicates that another command updated the active session.
	ErrSessionChanged = errors.New("the active session changed; run the command again")
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

// CheckTokens verifies that the request still belongs to the active session.
func CheckTokens(ac ActiveContext, token, refreshToken string) error {
	if testSessionEnabled() {
		return nil
	}
	st, err := LoadState()
	if err != nil {
		return err
	}
	if st.ActiveContext != ac {
		return ErrSessionChanged
	}
	return matchTokens(st, ac, token, refreshToken)
}

func matchTokens(st State, ac ActiveContext, token, refreshToken string) error {
	info, ok := AuthInfoFor(st, ac)
	if !ok || info.Token != token || info.RefreshToken != refreshToken {
		return ErrSessionChanged
	}
	return nil
}

// ReplaceTokens updates the pair only if the saved credentials still match.
func ReplaceTokens(ac ActiveContext, previousAccess, previousRefresh, access, refresh string, requireActive bool) error {
	if access == "" {
		return ErrNoToken
	}
	if testSessionEnabled() {
		return nil
	}
	return UpdateState(func(st *State) error {
		if requireActive && st.ActiveContext != ac {
			return ErrSessionChanged
		}
		if err := matchTokens(*st, ac, previousAccess, previousRefresh); err != nil {
			return err
		}
		for i := range st.AuthInfos {
			info := &st.AuthInfos[i]
			if info.PlatformURL == ac.PlatformURL && info.Username == ac.Username && info.Method == ac.Method {
				info.Token, info.RefreshToken = access, refresh
				return nil
			}
		}
		return ErrSessionChanged
	})
}
