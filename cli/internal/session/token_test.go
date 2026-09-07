package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/state"
)

func TestSaveLoginPersistsTokenInState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CRATER_TEST_SANDBOX", "")
	t.Setenv("CRATER_TEST_SANDBOX_SESSION", "")

	info := state.AuthInfo{
		PlatformURL: "https://example.invalid",
		Username:    "alice",
		Method:      "ldap",
		UserID:      7,
		Nickname:    "Alice",
		Role:        "user",
	}
	if err := SaveLogin(info, "secret-token"); err != nil {
		t.Fatal(err)
	}

	st, err := LoadState()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := ActiveAuthInfo(st)
	if !ok {
		t.Fatal("expected saved auth info")
	}
	if got.Token != "secret-token" {
		t.Fatalf("token = %q, want secret-token", got.Token)
	}

	raw, err := os.ReadFile(mustStatePath(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"token": "secret-token"`) {
		t.Fatalf("state.json missing persisted token: %s", raw)
	}

	token, err := LoadToken(st.ActiveContext)
	if err != nil {
		t.Fatal(err)
	}
	if token != "secret-token" {
		t.Fatalf("LoadToken = %q, want secret-token", token)
	}
}

func TestLoadTokenMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CRATER_TEST_SANDBOX", "")
	t.Setenv("CRATER_TEST_SANDBOX_SESSION", "")

	m, err := state.NewManager()
	if err != nil {
		t.Fatal(err)
	}
	m.State = state.State{
		AuthInfos: []state.AuthInfo{{
			PlatformURL: "https://example.invalid",
			Username:    "alice",
			Method:      "ldap",
		}},
		ActiveContext: state.ActiveContext{
			PlatformURL: "https://example.invalid",
			Username:    "alice",
			Method:      "ldap",
		},
	}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}

	_, err = LoadToken(m.State.ActiveContext)
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("LoadToken error = %v, want ErrNoToken", err)
	}
}

func TestPublicAuthInfoOmitsTokenFromJSON(t *testing.T) {
	info := state.AuthInfo{
		PlatformURL: "https://example.invalid",
		Username:    "alice",
		Method:      "ldap",
		UserID:      1,
		Nickname:    "Alice",
		Role:        "user",
		Token:       "secret-token",
	}

	stored, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stored), `"token":"secret-token"`) {
		t.Fatalf("persisted JSON should keep token: %s", stored)
	}

	exported, err := json.Marshal(PublicAuthInfo(info))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(exported), "secret-token") || strings.Contains(string(exported), `"token"`) {
		t.Fatalf("public JSON leaked token: %s", exported)
	}
}

func mustStatePath(t *testing.T) string {
	t.Helper()
	configHome, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(configHome, "crater", "state.json")
}
