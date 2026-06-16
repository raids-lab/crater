package session

import (
	"fmt"
	"os"

	"github.com/raids-lab/crater/cli/internal/state"
	"github.com/raids-lab/crater/cli/internal/testenv"
)

// testSessionEnabled controls session-level fakes for snapshot/testing.
// When enabled, session must not touch disk state or OS keyring.
func testSessionEnabled() bool {
	return testenv.SandboxSessionEnabled()
}

func fakeAuthState() state.State {
	if testenv.SandboxSessionEmpty() {
		return state.State{Language: fakeLanguage()}
	}

	// Deterministic fake data for snapshot/testing.
	infos := []state.AuthInfo{
		{
			PlatformURL: "https://example.invalid",
			Username:    "alice",
			Method:      "normal",
			UserID:      1001,
			Nickname:    "Alice",
			Role:        "admin",
		},
		{
			PlatformURL: "https://example.invalid",
			Username:    "bob",
			Method:      "ldap",
			UserID:      1002,
			Nickname:    "Bob",
			Role:        "user",
		},
		{
			PlatformURL: "https://staging.invalid",
			Username:    "carol",
			Method:      "normal",
			UserID:      1003,
			Nickname:    "Carol",
			Role:        "user",
		},
	}

	ac := state.ActiveContext{
		PlatformURL: infos[0].PlatformURL,
		Username:    infos[0].Username,
		Method:      infos[0].Method,
	}

	return state.State{
		AuthInfos:     infos,
		ActiveContext: ac,
		Language:      fakeLanguage(),
	}
}

func fakeLanguage() string {
	// Keep Language stable but allow snapshots to control it.
	lang := os.Getenv("CRATER_LANG")
	if lang == "" {
		lang = "en"
	}
	return lang
}

func fakeTokenFor(ac state.ActiveContext) string {
	return fmt.Sprintf("fake-token:%s", KeyringAccountKey(ac))
}
