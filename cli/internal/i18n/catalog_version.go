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

package i18n

// Version command domain: local CLI build and compatibility metadata.
var catalogVersion = map[Language]map[string]string{
	En: {
		"version_short":                 "Show CLI version and build information",
		"version_long":                  "Show the local Crater CLI product version, source revision, build metadata, runtime, and API compatibility versions.",
		"version_heading":               "Crater CLI",
		"version_label_product_version": "Version",
		"version_label_commit_sha":      "Git commit",
		"version_label_build_type":      "Build type",
		"version_label_build_time":      "Built",
		"version_label_go_version":      "Go version",
		"version_label_platform":        "OS/Arch",
		"version_label_api_version":     "API version",
		"version_label_min_backend":     "Minimum backend API version",
		"root_version_output":           "Crater CLI version %s, build %s",
		"err_version_json_conflict":     "--version cannot be combined with --json; use \"crater version --json\" for structured build information",
	},
	ZhCN: {
		"version_short":                 "显示 CLI 版本和构建信息",
		"version_long":                  "显示本地 Crater CLI 的产品版本、源码提交、构建信息、运行时以及 API 兼容版本。",
		"version_heading":               "Crater CLI",
		"version_label_product_version": "版本",
		"version_label_commit_sha":      "Git 提交",
		"version_label_build_type":      "构建类型",
		"version_label_build_time":      "构建于",
		"version_label_go_version":      "Go 版本",
		"version_label_platform":        "操作系统/架构",
		"version_label_api_version":     "API 版本",
		"version_label_min_backend":     "最低后端 API 版本",
		"root_version_output":           "Crater CLI 版本 %s，构建 %s",
		"err_version_json_conflict":     "--version 不能与 --json 同时使用；如需结构化构建信息，请使用 \"crater version --json\"",
	},
}
