package session

import (
	"github.com/raids-lab/crater/cli/internal/state"
)

// SaveLogin persists a successful login:
// - store token in secure storage (keyring)
// - upsert AuthInfo into state.json
// - set ActiveContext to this account
func SaveLogin(info state.AuthInfo, accessToken string) error {
	ac := state.ActiveContext{
		PlatformURL: info.PlatformURL,
		Username:    info.Username,
		Method:      info.Method,
	}
	if err := SaveToken(ac, accessToken); err != nil {
		return err
	}

	st, err := LoadState()
	if err != nil {
		return err
	}

	found := false
	for i := range st.AuthInfos {
		it := st.AuthInfos[i]
		if it.PlatformURL == info.PlatformURL && it.Username == info.Username && it.Method == info.Method {
			st.AuthInfos[i] = info
			found = true
			break
		}
	}
	if !found {
		st.AuthInfos = append(st.AuthInfos, info)
	}
	st.ActiveContext = ac
	return SaveState(st)
}

