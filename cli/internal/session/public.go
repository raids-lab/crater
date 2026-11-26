package session

// PublicAuthInfo returns a copy of info that is safe for command output.
// Persisted tokens are omitted from JSON via omitempty.
func PublicAuthInfo(info AuthInfo) AuthInfo {
	info.Token = ""
	info.RefreshToken = ""
	return info
}

// PublicAuthInfos maps PublicAuthInfo over a slice.
func PublicAuthInfos(infos []AuthInfo) []AuthInfo {
	if infos == nil {
		return nil
	}
	out := make([]AuthInfo, len(infos))
	for i, info := range infos {
		out[i] = PublicAuthInfo(info)
	}
	return out
}
