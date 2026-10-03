#!/usr/bin/env bash

# Copyright 2026 The Crater Project Team, RAIDS-Lab
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

commit_sha="${BUILD_COMMIT_SHA:-${GITHUB_SHA:-}}"
ref_type="${BUILD_REF_TYPE:-${GITHUB_REF_TYPE:-branch}}"
ref_name="${BUILD_REF_NAME:-${GITHUB_REF_NAME:-}}"
build_time="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
release_tag_pattern='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'

if [[ -z "$commit_sha" ]]; then
	echo "BUILD_COMMIT_SHA or GITHUB_SHA is required" >&2
	exit 1
fi

commit_sha_input="$commit_sha"
if ! commit_sha="$(git rev-parse --verify "${commit_sha_input}^{commit}" 2>/dev/null)"; then
	echo "Commit does not exist in the checkout: $commit_sha_input" >&2
	exit 1
fi

if [[ "$ref_type" == "tag" ]]; then
	if [[ -z "$ref_name" ]]; then
		echo "BUILD_REF_NAME or GITHUB_REF_NAME is required for tag builds" >&2
		exit 1
	fi
	if [[ ! "$ref_name" =~ $release_tag_pattern ]]; then
		echo "Release tag must use the vX.Y.Z format: $ref_name" >&2
		exit 1
	fi

	app_version="${ref_name#v}"
	build_type="release"
else
	if ! base_tag="$(
		git describe \
			--tags \
			--match 'v[0-9]*.[0-9]*.[0-9]*' \
			--exclude '*-*' \
			--exclude '*+*' \
			--first-parent \
			--abbrev=0 \
			"$commit_sha"
	)"; then
		echo "No reachable release tag matching v<major>.<minor>.<patch>" >&2
		exit 1
	fi
	if [[ ! "$base_tag" =~ $release_tag_pattern ]]; then
		echo "Release tag must use the vX.Y.Z format: $base_tag" >&2
		exit 1
	fi

	base_version="${base_tag#v}"
	commit_distance="$(git rev-list --first-parent --count "${base_tag}..${commit_sha}")"
	short_sha="$(git rev-parse --short "$commit_sha")"
	app_version="${base_version}+dev.${commit_distance}.g${short_sha}"
	build_type="development"
fi

version_info="$(
	printf 'app_version=%s\n' "$app_version"
	printf 'commit_sha=%s\n' "$commit_sha"
	printf 'build_type=%s\n' "$build_type"
	printf 'build_time=%s\n' "$build_time"
)"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
	printf '%s\n' "$version_info" >>"$GITHUB_OUTPUT"
else
	printf '%s\n' "$version_info"
fi
