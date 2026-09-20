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

const path = require("node:path");

const TARGETS = Object.freeze({
  "darwin:arm64": Object.freeze({
    packageName: "@raids-lab/crater-cli-darwin-arm64",
    binaryName: "crater",
  }),
  "darwin:x64": Object.freeze({
    packageName: "@raids-lab/crater-cli-darwin-x64",
    binaryName: "crater",
  }),
  "linux:arm64": Object.freeze({
    packageName: "@raids-lab/crater-cli-linux-arm64",
    binaryName: "crater",
  }),
  "linux:x64": Object.freeze({
    packageName: "@raids-lab/crater-cli-linux-x64",
    binaryName: "crater",
  }),
  "win32:arm64": Object.freeze({
    packageName: "@raids-lab/crater-cli-win32-arm64",
    binaryName: "crater.exe",
  }),
  "win32:x64": Object.freeze({
    packageName: "@raids-lab/crater-cli-win32-x64",
    binaryName: "crater.exe",
  }),
});

function targetFor(platform = process.platform, arch = process.arch) {
  const target = TARGETS[`${platform}:${arch}`];
  if (target) {
    return target;
  }
  const supported = Object.keys(TARGETS).sort().join(", ");
  throw new Error(
    `Crater CLI does not provide a binary for ${platform}/${arch}. Supported targets: ${supported}.`,
  );
}

function resolveBinaryPath(
  platform = process.platform,
  arch = process.arch,
  resolvePackageJSON = require.resolve,
) {
  const target = targetFor(platform, arch);
  let packageJSONPath;
  try {
    packageJSONPath = resolvePackageJSON(`${target.packageName}/package.json`);
  } catch (error) {
    const detail = error instanceof Error ? ` ${error.message}` : "";
    throw new Error(
      `The optional package ${target.packageName} is missing. Reinstall @raids-lab/crater-cli without disabling optional dependencies.${detail}`,
    );
  }
  return path.join(path.dirname(packageJSONPath), "bin", target.binaryName);
}

module.exports = {
  TARGETS,
  resolveBinaryPath,
  targetFor,
};
