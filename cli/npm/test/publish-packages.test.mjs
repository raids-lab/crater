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
import test from "node:test";

import { publishPackages } from "../scripts/publish-packages.mjs";

function manifest() {
  return {
    version: "1.2.3",
    platformPackages: [
      "darwin-arm64",
      "darwin-x64",
      "linux-arm64",
      "linux-x64",
      "win32-arm64",
      "win32-x64",
    ].map((target) => ({
      name: `@raids-lab/crater-cli-${target}`,
      directory: `/packages/${target}`,
    })),
    rootPackage: {
      name: "@raids-lab/crater-cli",
      directory: "/packages/root",
    },
  };
}

test("publishes platform packages before the entry package", async () => {
  const published = [];
  const visible = new Set();
  const input = manifest();

  await publishPackages(input, {
    lookupVersion: (name, version) => (visible.has(name) ? version : null),
    publish: async (entry) => {
      published.push(entry.name);
      visible.add(entry.name);
    },
    waitUntilVisible: async (name, version, lookupVersion) => {
      assert.equal(lookupVersion(name, version), version);
    },
  });

  assert.deepEqual(published, [
    ...input.platformPackages.map((entry) => entry.name),
    input.rootPackage.name,
  ]);
});

test("skips package versions that are already public", async () => {
  const input = manifest();
  const alreadyPublished = new Set([
    input.platformPackages[0].name,
    input.platformPackages[3].name,
  ]);
  const published = [];

  await publishPackages(input, {
    lookupVersion: (name, version) => (alreadyPublished.has(name) ? version : null),
    publish: async (entry) => {
      published.push(entry.name);
      alreadyPublished.add(entry.name);
    },
    waitUntilVisible: async () => {},
  });

  assert.deepEqual(published, [
    input.platformPackages[1].name,
    input.platformPackages[2].name,
    input.platformPackages[4].name,
    input.platformPackages[5].name,
    input.rootPackage.name,
  ]);
});

test("rejects an incomplete publish manifest", async () => {
  const input = manifest();
  input.platformPackages.pop();
  await assert.rejects(publishPackages(input), /exactly six platform packages/);
});

test("rejects unexpected or duplicate platform packages", async () => {
  const input = manifest();
  input.platformPackages[5] = input.platformPackages[0];
  await assert.rejects(publishPackages(input), /unexpected platform package names/);
});

test("rejects a non-release manifest version", async () => {
  const input = manifest();
  input.version = "01.2.3";
  await assert.rejects(publishPackages(input), /without leading zeroes/);
});
