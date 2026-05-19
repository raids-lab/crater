package session

import (
	"github.com/raids-lab/crater/cli/internal/state"
)

// LoadState loads state.json from disk.
func LoadState() (state.State, error) {
	if testSessionEnabled() {
		return fakeAuthState(), nil
	}
	m, err := state.NewManager()
	if err != nil {
		return state.State{}, err
	}
	return m.State, nil
}

// SaveState writes the given state.json to disk.
func SaveState(st state.State) error {
	if testSessionEnabled() {
		return nil
	}
	m, err := state.NewManager()
	if err != nil {
		return err
	}
	m.State = st
	if err := m.Save(); err != nil {
		return err
	}
	return nil
}

// ActiveAuthInfo returns the AuthInfo that matches the current ActiveContext in state.
// If no active context is set or no matching AuthInfo exists, ok is false.
func ActiveAuthInfo(st state.State) (info state.AuthInfo, ok bool) {
	ac := st.ActiveContext
	if ac.PlatformURL == "" || ac.Username == "" || ac.Method == "" {
		return state.AuthInfo{}, false
	}
	for _, it := range st.AuthInfos {
		if it.PlatformURL == ac.PlatformURL && it.Username == ac.Username && it.Method == ac.Method {
			return it, true
		}
	}
	return state.AuthInfo{}, false
}
