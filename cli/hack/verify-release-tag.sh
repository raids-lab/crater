#!/usr/bin/env bash

# Copyright 2026 The Crater Project Team, RAIDS-Lab
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

if [[ $# -ne 2 ]]; then
	echo "usage: $0 <vX.Y.Z-tag> <expected-commit>" >&2
	exit 2
fi

tag_name="$1"
expected_commit_input="$2"
release_tag_pattern='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'

if [[ ! "$tag_name" =~ $release_tag_pattern ]]; then
	echo "release tag must use the vX.Y.Z format: $tag_name" >&2
	exit 2
fi
if ! expected_commit="$(git rev-parse --verify "${expected_commit_input}^{commit}" 2>/dev/null)"; then
	echo "expected release commit does not exist: $expected_commit_input" >&2
	exit 2
fi

if ! remote_refs="$(git ls-remote --exit-code origin "refs/tags/${tag_name}" "refs/tags/${tag_name}^{}")"; then
	echo "release tag does not exist on origin: $tag_name" >&2
	exit 1
fi

remote_tag_object=""
remote_commit=""
while read -r object_id ref_name; do
	case "$ref_name" in
	"refs/tags/${tag_name}")
		remote_tag_object="$object_id"
		;;
	"refs/tags/${tag_name}^{}")
		remote_commit="$object_id"
		;;
	esac
done <<<"$remote_refs"

# Lightweight tags point at the commit directly; annotated tags expose the
# peeled commit through the ^{} ref returned by git ls-remote.
if [[ -z "$remote_commit" ]]; then
	remote_commit="$remote_tag_object"
fi
if [[ "$remote_commit" != "$expected_commit" ]]; then
	echo "release tag ${tag_name} moved: expected ${expected_commit}, found ${remote_commit}" >&2
	exit 1
fi

printf '%s points to %s\n' "$tag_name" "$expected_commit"
