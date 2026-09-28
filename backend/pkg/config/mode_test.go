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

package config

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsDebugModeIncludesGinTestMode(t *testing.T) {
	originalMode := gin.Mode()
	t.Cleanup(func() { gin.SetMode(originalMode) })

	gin.SetMode(gin.TestMode)
	if !IsDebugMode() {
		t.Fatal("gin test mode must use the debug configuration path")
	}

	gin.SetMode(gin.ReleaseMode)
	if IsDebugMode() {
		t.Fatal("gin release mode must use the production configuration path")
	}
}

func TestConfigPathForTestModeUsesBundledExample(t *testing.T) {
	path := configPathForMode(gin.TestMode, "", true)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("test config path %q is not available: %v", path, err)
	}
}

func TestConfigPathForDebugTestProcessUsesBundledExample(t *testing.T) {
	path := configPathForMode(gin.DebugMode, "", true)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("test config path %q is not available: %v", path, err)
	}
}

func TestConfigPathForDebugModeUsesOverride(t *testing.T) {
	const want = "/tmp/crater-test-debug-config.yaml"
	if got := configPathForMode(gin.DebugMode, want, false); got != want {
		t.Fatalf("configPathForMode() = %q, want %q", got, want)
	}
}

func TestConfigPathForDebugModeRequiresLocalConfig(t *testing.T) {
	const want = "./etc/debug-config.yaml"
	if got := configPathForMode(gin.DebugMode, "", false); got != want {
		t.Fatalf("configPathForMode() = %q, want %q", got, want)
	}
}
