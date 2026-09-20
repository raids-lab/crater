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

if [[ $# -ne 3 ]]; then
	echo "usage: $0 <goos> <goarch> <output-directory>" >&2
	exit 2
fi

goos="$1"
goarch="$2"
output_arg="$3"

case "${goos}/${goarch}" in
	linux/amd64 | linux/arm64 | darwin/amd64 | darwin/arm64 | windows/amd64 | windows/arm64) ;;
	*)
		echo "unsupported release target: ${goos}/${goarch}" >&2
		exit 2
		;;
esac

: "${APP_VERSION:?APP_VERSION is required}"
: "${COMMIT_SHA:?COMMIT_SHA is required}"
: "${BUILD_TYPE:?BUILD_TYPE is required}"
: "${BUILD_TIME:?BUILD_TIME is required}"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
repo_root="$(cd "${script_dir}/../.." && pwd -P)"
mkdir -p "$output_arg"
output_dir="$(cd "$output_arg" && pwd -P)"

target_name="crater-${goos}-${goarch}"
package_dir="${output_dir}/${target_name}"
binary_name="crater"
archive_name="${target_name}.tar.gz"
if [[ "$goos" == "windows" ]]; then
	binary_name="crater.exe"
	archive_name="${target_name}.zip"
fi

if [[ -e "$package_dir" || -e "${output_dir}/${archive_name}" ]]; then
	echo "release output already exists for ${goos}/${goarch}: $output_dir" >&2
	exit 1
fi

mkdir -p "$package_dir"
make -C "${repo_root}/cli" release-build \
	GOOS="$goos" \
	GOARCH="$goarch" \
	OUTPUT="${package_dir}/${binary_name}" \
	APP_VERSION="$APP_VERSION" \
	COMMIT_SHA="$COMMIT_SHA" \
	BUILD_TYPE="$BUILD_TYPE" \
	BUILD_TIME="$BUILD_TIME"

chmod 0755 "${package_dir}/${binary_name}"
cp "${repo_root}/cli/LICENSE" "${package_dir}/LICENSE"
cp "${repo_root}/NOTICE" "${package_dir}/NOTICE"
cp "${repo_root}/cli/README.md" "${package_dir}/README.md"

if [[ "$goos" == "windows" ]]; then
	(
		cd "$output_dir"
		zip -q -r "$archive_name" "$target_name"
	)
else
	tar -C "$output_dir" -czf "${output_dir}/${archive_name}" "$target_name"
fi

printf '%s\n' "${output_dir}/${archive_name}"
