// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
