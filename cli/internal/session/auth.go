package session

import (
	"github.com/raids-lab/crater/cli/internal/state"
)

// SaveLogin persists a successful login:
// - store the access token on the AuthInfo
// - upsert AuthInfo into state.json
// - set ActiveContext to this account
func SaveLogin(info state.AuthInfo, accessToken, refreshToken string) error {
	info.Token, info.RefreshToken = accessToken, refreshToken
	ac := state.ActiveContext{
		PlatformURL: info.PlatformURL,
		Username:    info.Username,
		Method:      info.Method,
	}

	return UpdateState(func(st *State) error {
		st.ActiveContext = ac
		for i, it := range st.AuthInfos {
			if it.PlatformURL == info.PlatformURL && it.Username == info.Username && it.Method == info.Method {
				st.AuthInfos[i] = info
				return nil
			}
		}
		st.AuthInfos = append(st.AuthInfos, info)
		return nil
	})
}
