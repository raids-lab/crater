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

import "strings"

const (
	DefaultHuggingFaceEndpoint       = "https://huggingface.co"
	DefaultModelScopeEndpoint        = "https://modelscope.cn"
	DefaultMetadataTimeout           = 20
	DefaultMaxLogoBytes        int64 = 512 * 1024
	DefaultLogicalPublicPrefix       = "public"
)

func (c *Config) HuggingFaceDownloadEndpoint() string {
	return endpointOrDefault(c.ModelDownload.HuggingFaceEndpoint, DefaultHuggingFaceEndpoint)
}

func (c *Config) ModelScopeDownloadEndpoint() string {
	return endpointOrDefault(c.ModelDownload.ModelScopeEndpoint, DefaultModelScopeEndpoint)
}

func (c *Config) HuggingFaceMetadataEndpoints() []string {
	return endpointsOrDefault(c.ModelMetadata.HuggingFaceEndpoints, c.HuggingFaceDownloadEndpoint())
}

func (c *Config) ModelScopeMetadataEndpoints() []string {
	return endpointsOrDefault(c.ModelMetadata.ModelScopeEndpoints, c.ModelScopeDownloadEndpoint())
}

func (c *Config) MetadataTimeoutSeconds() int {
	if c.ModelMetadata.TimeoutSeconds > 0 {
		return c.ModelMetadata.TimeoutSeconds
	}
	return DefaultMetadataTimeout
}

func (c *Config) MetadataMaxLogoBytes() int64 {
	if c.ModelMetadata.MaxLogoBytes > 0 {
		return c.ModelMetadata.MaxLogoBytes
	}
	return DefaultMaxLogoBytes
}

func (c *Config) MetadataLogicalPublicPrefix() string {
	prefix := strings.Trim(strings.TrimSpace(c.ModelMetadata.LogicalPublicPrefix), "/")
	if prefix == "" {
		return DefaultLogicalPublicPrefix
	}
	return prefix
}

func endpointOrDefault(endpoint, fallback string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return fallback
	}
	return endpoint
}

func endpointsOrDefault(endpoints []string, fallback string) []string {
	result := make([]string, 0, len(endpoints))
	seen := make(map[string]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
		if endpoint == "" {
			continue
		}
		if _, exists := seen[endpoint]; exists {
			continue
		}
		seen[endpoint] = struct{}{}
		result = append(result, endpoint)
	}
	if len(result) == 0 {
		return []string{fallback}
	}
	return result
}
