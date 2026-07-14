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

import "testing"

func TestMetadataLogoAllowedHosts(t *testing.T) {
	var defaultConfig Config
	defaults := defaultConfig.MetadataLogoAllowedHosts()
	if len(defaults) != 2 || defaults[1] != "cdn-avatars.huggingface.co" {
		t.Fatalf("unexpected default logo hosts: %v", defaults)
	}

	var customConfig Config
	customConfig.ModelMetadata.LogoAllowedHosts = []string{
		" CDN.EXAMPLE. ", "cdn.example", "mirror.example",
	}
	custom := customConfig.MetadataLogoAllowedHosts()
	if len(custom) != 2 || custom[0] != "cdn.example" || custom[1] != "mirror.example" {
		t.Fatalf("unexpected normalized logo hosts: %v", custom)
	}
}
