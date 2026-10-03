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

import {
  chmod,
  copyFile,
  mkdir,
  readdir,
  writeFile,
} from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const npmRoot = fileURLToPath(new URL("../", import.meta.url));
const cliRoot = fileURLToPath(new URL("../../", import.meta.url));
const repoRoot = fileURLToPath(new URL("../../../", import.meta.url));

export const targets = Object.freeze([
  { goos: "darwin", goarch: "arm64", os: "darwin", cpu: "arm64", suffix: "darwin-arm64" },
  { goos: "darwin", goarch: "amd64", os: "darwin", cpu: "x64", suffix: "darwin-x64" },
  { goos: "linux", goarch: "arm64", os: "linux", cpu: "arm64", suffix: "linux-arm64" },
  { goos: "linux", goarch: "amd64", os: "linux", cpu: "x64", suffix: "linux-x64" },
  { goos: "windows", goarch: "arm64", os: "win32", cpu: "arm64", suffix: "win32-arm64" },
  { goos: "windows", goarch: "amd64", os: "win32", cpu: "x64", suffix: "win32-x64" },
]);

const stableVersionPattern = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;

function packageMetadata(name, version, description) {
  return {
    name,
    version,
    description,
    license: "Apache-2.0",
    author: "The Crater Project Team, RAIDS-Lab",
    repository: {
      type: "git",
      url: "https://github.com/raids-lab/crater.git",
      directory: "cli",
    },
    homepage: "https://github.com/raids-lab/crater/tree/main/cli",
    bugs: {
      url: "https://github.com/raids-lab/crater/issues",
    },
    publishConfig: {
      access: "public",
    },
  };
}

async function copyCommonFiles(destination) {
  await Promise.all([
    copyFile(path.join(npmRoot, "README.md"), path.join(destination, "README.md")),
    copyFile(path.join(cliRoot, "LICENSE"), path.join(destination, "LICENSE")),
    copyFile(path.join(repoRoot, "NOTICE"), path.join(destination, "NOTICE")),
  ]);
}

async function writePackageJSON(directory, value) {
  await writeFile(
    path.join(directory, "package.json"),
    `${JSON.stringify(value, null, 2)}\n`,
    "utf8",
  );
}

async function requireEmptyDirectory(directory) {
  await mkdir(directory, { recursive: true });
  const entries = await readdir(directory);
  if (entries.length !== 0) {
    throw new Error(`output directory must be empty: ${directory}`);
  }
}

export async function buildPackages({ version, artifactsDir, outputDir }) {
  if (!stableVersionPattern.test(version)) {
    throw new Error(`npm package version must use X.Y.Z without leading zeroes: ${version}`);
  }

  const artifacts = path.resolve(artifactsDir);
  const output = path.resolve(outputDir);
  await requireEmptyDirectory(output);

  const optionalDependencies = {};
  const platformPackages = [];

  for (const target of targets) {
    const targetName = `crater-${target.goos}-${target.goarch}`;
    const sourceBinaryName = target.goos === "windows" ? "crater.exe" : "crater";
    const sourceBinary = path.join(artifacts, targetName, sourceBinaryName);
    const packageName = `@raids-lab/crater-cli-${target.suffix}`;
    const directory = path.join(output, `crater-cli-${target.suffix}`);
    const binaryDirectory = path.join(directory, "bin");
    const destinationBinary = path.join(binaryDirectory, sourceBinaryName);

    await mkdir(binaryDirectory, { recursive: true });
    await copyFile(sourceBinary, destinationBinary);
    await chmod(destinationBinary, 0o755);
    await copyCommonFiles(directory);
    await writePackageJSON(directory, {
      ...packageMetadata(
        packageName,
        version,
        `Crater CLI native binary for ${target.os}/${target.cpu}`,
      ),
      os: [target.os],
      cpu: [target.cpu],
      files: ["bin", "README.md", "LICENSE", "NOTICE"],
    });

    optionalDependencies[packageName] = version;
    platformPackages.push({ name: packageName, directory });
  }

  const rootDirectory = path.join(output, "crater-cli");
  await mkdir(path.join(rootDirectory, "bin"), { recursive: true });
  await mkdir(path.join(rootDirectory, "lib"), { recursive: true });
  await Promise.all([
    copyFile(path.join(npmRoot, "bin", "crater.js"), path.join(rootDirectory, "bin", "crater.js")),
    copyFile(path.join(npmRoot, "lib", "platform.cjs"), path.join(rootDirectory, "lib", "platform.cjs")),
  ]);
  await chmod(path.join(rootDirectory, "bin", "crater.js"), 0o755);
  await copyCommonFiles(rootDirectory);
  await writePackageJSON(rootDirectory, {
    ...packageMetadata(
      "@raids-lab/crater-cli",
      version,
      "Official command-line client for the Crater platform",
    ),
    keywords: ["crater", "cli", "kubernetes", "gpu", "ai"],
    bin: {
      crater: "bin/crater.js",
    },
    files: ["bin", "lib", "README.md", "LICENSE", "NOTICE"],
    engines: {
      node: ">=18",
    },
    optionalDependencies,
  });

  const manifest = {
    version,
    platformPackages,
    rootPackage: {
      name: "@raids-lab/crater-cli",
      directory: rootDirectory,
    },
  };
  await writeFile(
    path.join(output, "publish-manifest.json"),
    `${JSON.stringify(manifest, null, 2)}\n`,
    "utf8",
  );
  return manifest;
}

function parseArguments(argv) {
  const values = {};
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index];
    const value = argv[index + 1];
    if (!key?.startsWith("--") || value === undefined) {
      throw new Error("usage: build-packages.mjs --version X.Y.Z --artifacts DIR --output DIR");
    }
    values[key.slice(2)] = value;
  }
  if (!values.version || !values.artifacts || !values.output) {
    throw new Error("usage: build-packages.mjs --version X.Y.Z --artifacts DIR --output DIR");
  }
  return {
    version: values.version,
    artifactsDir: values.artifacts,
    outputDir: values.output,
  };
}

const invokedPath = process.argv[1] ? path.resolve(process.argv[1]) : "";
if (invokedPath === fileURLToPath(import.meta.url)) {
  try {
    const manifest = await buildPackages(parseArguments(process.argv.slice(2)));
    process.stdout.write(`${JSON.stringify(manifest, null, 2)}\n`);
  } catch (error) {
    process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
    process.exitCode = 1;
  }
}
