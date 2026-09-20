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

if [[ $# -ne 7 ]]; then
	echo "usage: $0 <binary> <expected-os> <expected-arch> <expected-build-type> <expected-commit> <expected-version> <expected-build-time>" >&2
	exit 2
fi

binary="$1"
expected_os="$2"
expected_arch="$3"
expected_build_type="$4"
expected_commit="$5"
expected_version="$6"
expected_build_time="$7"

if [[ ! -f "$binary" ]]; then
	echo "release binary does not exist: $binary" >&2
	exit 1
fi

chmod 0755 "$binary"
expected_short_version="Crater CLI version ${expected_version}, build ${expected_commit:0:7}"
short_version="$(CRATER_LANG=en "$binary" --version)"
if [[ "$short_version" != "$expected_short_version" ]]; then
	printf '%s\n' "--version output is '$short_version', expected '$expected_short_version'" >&2
	exit 1
fi
if [[ "$(CRATER_LANG=en "$binary" -v)" != "$expected_short_version" ]]; then
	echo "-v output does not match --version" >&2
	exit 1
fi

version_json="$("$binary" version --json)"
printf '%s' "$version_json" | node -e '
  const fs = require("node:fs");
  const [expectedOS, expectedArch, expectedBuildType, expectedCommit, expectedVersion, expectedBuildTime] = process.argv.slice(1);
  const payload = JSON.parse(fs.readFileSync(0, "utf8"));
  const info = payload?.data?.version;
  if (payload?.status !== "OK" || !info) {
    throw new Error("version output does not contain the success envelope");
  }
  const expected = {
    os: expectedOS,
    arch: expectedArch,
    build_type: expectedBuildType,
    commit_sha: expectedCommit,
    product_version: expectedVersion,
    build_time: expectedBuildTime,
  };
  for (const [field, value] of Object.entries(expected)) {
    if (info[field] !== value) {
      throw new Error(`${field} is ${JSON.stringify(info[field])}, expected ${JSON.stringify(value)}`);
    }
  }
  if (typeof info.product_version !== "string" || info.product_version.length === 0) {
    throw new Error("product_version is empty");
  }
  if (typeof info.go_version !== "string" || !info.go_version.startsWith("go")) {
    throw new Error(`invalid go_version: ${JSON.stringify(info.go_version)}`);
  }
  if (!Number.isInteger(info.api_version) || info.api_version <= 0) {
    throw new Error(`invalid api_version: ${JSON.stringify(info.api_version)}`);
  }
  if (Number.isNaN(Date.parse(info.build_time))) {
    throw new Error(`invalid build_time: ${JSON.stringify(info.build_time)}`);
  }
' "$expected_os" "$expected_arch" "$expected_build_type" "$expected_commit" "$expected_version" "$expected_build_time"

"$binary" --help >/dev/null
