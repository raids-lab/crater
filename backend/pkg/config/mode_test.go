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
