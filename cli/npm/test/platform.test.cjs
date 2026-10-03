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

"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");

const { resolveBinaryPath, targetFor } = require("../lib/platform.cjs");

test("maps supported Node targets to platform packages", () => {
  assert.equal(
    targetFor("darwin", "arm64").packageName,
    "@raids-lab/crater-cli-darwin-arm64",
  );
  assert.equal(
    targetFor("linux", "x64").packageName,
    "@raids-lab/crater-cli-linux-x64",
  );
  assert.equal(targetFor("win32", "x64").binaryName, "crater.exe");
});

test("rejects unsupported targets with an actionable message", () => {
  assert.throws(
    () => targetFor("freebsd", "x64"),
    /does not provide a binary for freebsd\/x64/,
  );
});

test("resolves the executable relative to the platform package", () => {
  const resolved = resolveBinaryPath(
    "linux",
    "x64",
    (request) => `/packages/${request}`,
  );
  assert.equal(
    resolved,
    "/packages/@raids-lab/crater-cli-linux-x64/bin/crater",
  );
});

test("explains when optional dependencies were disabled", () => {
  assert.throws(
    () =>
      resolveBinaryPath("linux", "arm64", () => {
        throw new Error("module not found");
      }),
    /Reinstall @raids-lab\/crater-cli without disabling optional dependencies/,
  );
});
