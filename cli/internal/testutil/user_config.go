package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// IsolateUserConfigDir points platform-specific user configuration paths at a
// fresh temporary directory and returns the path selected by os.UserConfigDir.
func IsolateUserConfigDir(t testing.TB) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData", "Roaming"))

	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("get isolated user config dir: %v", err)
	}
	return configDir
}

// UserConfigEnv returns the platform-specific user configuration environment
// for subprocess tests rooted at home.
func UserConfigEnv(home string) []string {
	return []string{
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"AppData=" + filepath.Join(home, "AppData", "Roaming"),
	}
}
