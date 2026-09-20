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

import { readFile } from "node:fs/promises";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const stableVersionPattern = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;
const expectedPlatformPackageNames = Object.freeze([
  "@raids-lab/crater-cli-darwin-arm64",
  "@raids-lab/crater-cli-darwin-x64",
  "@raids-lab/crater-cli-linux-arm64",
  "@raids-lab/crater-cli-linux-x64",
  "@raids-lab/crater-cli-win32-arm64",
  "@raids-lab/crater-cli-win32-x64",
]);
const registryURL = "https://registry.npmjs.org";

function runNpm(args, options = {}) {
  const result = spawnSync("npm", args, {
    encoding: "utf8",
    ...options,
  });
  if (result.error) {
    throw new Error(`unable to execute npm: ${result.error.message}`);
  }
  return result;
}

function registryVersion(packageName, version) {
  const result = runNpm([
    "view",
    `${packageName}@${version}`,
    "version",
    "--json",
    "--registry",
    registryURL,
  ]);
  if (result.status !== 0) {
    const detail = `${result.stderr ?? ""}\n${result.stdout ?? ""}`;
    if (/\bE404\b|is not in this registry|No match found for version/i.test(detail)) {
      return null;
    }
    throw new Error(
      `npm view failed for ${packageName}@${version} with exit code ${result.status}: ${detail.trim()}`,
    );
  }
  try {
    return JSON.parse(result.stdout);
  } catch (error) {
    throw new Error(`npm returned invalid metadata for ${packageName}@${version}: ${error.message}`);
  }
}

function delay(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

async function waitForRegistry(packageName, version, lookupVersion = registryVersion) {
  for (let attempt = 1; attempt <= 12; attempt += 1) {
    if (lookupVersion(packageName, version) === version) {
      return;
    }
    if (attempt < 12) {
      await delay(5000);
    }
  }
  throw new Error(`npm did not expose ${packageName}@${version} after publication`);
}

function publishDirectory(packageEntry) {
  const result = runNpm(
    [
      "publish",
      packageEntry.directory,
      "--access",
      "public",
      "--provenance",
      "--registry",
      registryURL,
    ],
    { stdio: "inherit" },
  );
  if (result.status !== 0) {
    throw new Error(`npm publish failed for ${packageEntry.name} with exit code ${result.status}`);
  }
}

async function publishPackage(
  packageEntry,
  version,
  {
    lookupVersion = registryVersion,
    publish = publishDirectory,
    waitUntilVisible = waitForRegistry,
  } = {},
) {
  const packageSpec = `${packageEntry.name}@${version}`;
  if (lookupVersion(packageEntry.name, version) === version) {
    process.stdout.write(`${packageSpec} is already published; skipping\n`);
    return;
  }

  process.stdout.write(`Publishing ${packageSpec}\n`);
  await publish(packageEntry, version);
  await waitUntilVisible(packageEntry.name, version, lookupVersion);
}

function validateManifest(manifest) {
  if (!manifest || typeof manifest.version !== "string") {
    throw new Error("publish manifest is missing a version");
  }
  if (!stableVersionPattern.test(manifest.version)) {
    throw new Error(`publish manifest version must use X.Y.Z without leading zeroes: ${manifest.version}`);
  }
  if (!Array.isArray(manifest.platformPackages) || manifest.platformPackages.length !== 6) {
    throw new Error("publish manifest must contain exactly six platform packages");
  }
  if (!manifest.rootPackage || manifest.rootPackage.name !== "@raids-lab/crater-cli") {
    throw new Error("publish manifest is missing the Crater CLI entry package");
  }
  for (const entry of [...manifest.platformPackages, manifest.rootPackage]) {
    if (typeof entry.name !== "string" || typeof entry.directory !== "string") {
      throw new Error("publish manifest contains an invalid package entry");
    }
  }
  const packageNames = manifest.platformPackages.map((entry) => entry.name).sort();
  if (JSON.stringify(packageNames) !== JSON.stringify([...expectedPlatformPackageNames].sort())) {
    throw new Error("publish manifest contains unexpected platform package names");
  }
}

export async function publishPackages(manifest, options = {}) {
  validateManifest(manifest);
  for (const packageEntry of manifest.platformPackages) {
    await publishPackage(packageEntry, manifest.version, options);
  }
  await publishPackage(manifest.rootPackage, manifest.version, options);
}

async function main() {
  const manifestArgument = process.argv[2];
  if (!manifestArgument || process.argv.length !== 3) {
    throw new Error("usage: publish-packages.mjs <publish-manifest.json>");
  }

  const manifestPath = path.resolve(manifestArgument);
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  await publishPackages(manifest);
}

const invokedPath = process.argv[1] ? path.resolve(process.argv[1]) : "";
if (invokedPath === fileURLToPath(import.meta.url)) {
  try {
    await main();
  } catch (error) {
    process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
    process.exitCode = 1;
  }
}
