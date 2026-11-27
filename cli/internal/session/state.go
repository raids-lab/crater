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
	return UpdateState(func(current *State) error {
		*current = st
		return nil
	})
}

// UpdateState changes only the specified fields of the latest saved state.
func UpdateState(change func(*State) error) error {
	if testSessionEnabled() {
		st := fakeAuthState()
		return change(&st)
	}
	return state.Update(change)
}

// AuthInfoFor returns the AuthInfo that matches ac.
// If no matching AuthInfo exists, ok is false.
func AuthInfoFor(st state.State, ac state.ActiveContext) (info state.AuthInfo, ok bool) {
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

// ActiveAuthInfo returns the AuthInfo that matches the current ActiveContext in state.
// If no active context is set or no matching AuthInfo exists, ok is false.
func ActiveAuthInfo(st state.State) (info state.AuthInfo, ok bool) {
	return AuthInfoFor(st, st.ActiveContext)
}
