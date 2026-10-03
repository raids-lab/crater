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

import assert from "node:assert/strict";
import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import { buildPackages, targets } from "../scripts/build-packages.mjs";

test("builds platform packages before the root package", async () => {
  const temporaryRoot = await mkdtemp(path.join(os.tmpdir(), "crater-npm-packages-"));
  const artifactsDir = path.join(temporaryRoot, "artifacts");
  const outputDir = path.join(temporaryRoot, "output");

  for (const target of targets) {
    const targetName = `crater-${target.goos}-${target.goarch}`;
    const binaryName = target.goos === "windows" ? "crater.exe" : "crater";
    const targetDirectory = path.join(artifactsDir, targetName);
    await mkdir(targetDirectory, { recursive: true });
    await writeFile(path.join(targetDirectory, binaryName), `${targetName}\n`, "utf8");
  }

  const manifest = await buildPackages({
    version: "1.2.3",
    artifactsDir,
    outputDir,
  });

  assert.equal(manifest.platformPackages.length, 6);
  assert.equal(manifest.rootPackage.name, "@raids-lab/crater-cli");

  const rootPackage = JSON.parse(
    await readFile(path.join(manifest.rootPackage.directory, "package.json"), "utf8"),
  );
  assert.equal(rootPackage.version, "1.2.3");
  assert.equal(rootPackage.bin.crater, "bin/crater.js");
  assert.equal(
    rootPackage.optionalDependencies["@raids-lab/crater-cli-win32-x64"],
    "1.2.3",
  );

  const windowsPackage = manifest.platformPackages.find(
    (entry) => entry.name === "@raids-lab/crater-cli-win32-x64",
  );
  const windowsBinary = await readFile(
    path.join(windowsPackage.directory, "bin", "crater.exe"),
    "utf8",
  );
  assert.equal(windowsBinary, "crater-windows-amd64\n");
});

test("rejects non-release versions", async () => {
  const temporaryRoot = await mkdtemp(path.join(os.tmpdir(), "crater-npm-version-"));
  await assert.rejects(
    buildPackages({
      version: "1.2.3-dev.1",
      artifactsDir: path.join(temporaryRoot, "artifacts"),
      outputDir: path.join(temporaryRoot, "output"),
    }),
    /must use X.Y.Z/,
  );
});
