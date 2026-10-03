package session

import (
	"github.com/raids-lab/crater/cli/internal/state"
)

// SaveLogin persists a successful login:
// - store the access token on the AuthInfo
// - upsert AuthInfo into state.json
// - set ActiveContext to this account
func SaveLogin(info state.AuthInfo, accessToken string) error {
	if testSessionEnabled() {
		return nil
	}
	info.Token = accessToken
	ac := state.ActiveContext{
		PlatformURL: info.PlatformURL,
		Username:    info.Username,
		Method:      info.Method,
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
