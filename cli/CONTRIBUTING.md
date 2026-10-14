[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

# Contributing to Crater CLI

Crater CLI is developed in a documentation-driven way. Before changing code, first identify which document defines the behavior or rule you are touching. This keeps the implementation, tests, Agent Skills, and user-visible command contract aligned.

## 1. Understand The Contract

Start from the document that owns the change:

- Use [docs/COMMANDS.md](docs/COMMANDS.md) when adding or changing commands, flags, arguments, stdout/stderr behavior, JSON fields, errors, or exit-code expectations.
- Use [docs/SPEC.md](docs/SPEC.md) when changing cross-command rules, shared output contracts, error categories, snapshot requirements, i18n rules, completion rules, sandbox behavior, or development workflow.
- Use [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) when changing package responsibilities, module boundaries, request flow, state/credential access, or test infrastructure.
- Use [docs/REVIEW.md](docs/REVIEW.md) before finishing a development stage. It explains what reviewers and AI reviewers should check.

If a planned code change has no matching contract, update the appropriate document first. If the document is correct and the code differs, fix the code and tests to match the document.

You can run `make help` in `cli/` at any time to see the available local workflow commands.

Before running CLI build or test targets, run `go version` and make sure it matches `cli/go.mod` (currently `1.25.4`).

## 2. Implement The Change

Keep changes scoped to the command domain or shared module you are touching.

Use [docs/SPEC.md](docs/SPEC.md) for cross-command rules and shared implementation constraints, [docs/COMMANDS.md](docs/COMMANDS.md) for user-visible command contracts, and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for package boundaries and request flow. If Agent Skills change, follow the Skills rules in `docs/SPEC.md`.

When an API used by the CLI changes, apply the root [CLI / Backend API Compatibility Versions](../CONTRIBUTING.md#cli--backend-api-compatibility-versions) decision table. Update the backend and CLI `APIVersion` together only when the API contract itself changes. If the CLI starts requiring a capability already present in that contract, keep `APIVersion` unchanged and raise `MinSupportedBackendAPIVersion` to the version that first provided the capability only when no fallback remains. Record both decisions, including an unchanged decision and its reason, in the PR description or verification notes.

When you need to manually try the CLI, build the local binary first:

```bash
make build
```

This runs `go mod tidy` and builds the local `./crater` binary.

Release or packaged builds should pass the CLI product version, for example `make build APP_VERSION=0.4.0`; it is embedded in the standard `User-Agent` header. Local builds default to `dev`.

For user-visible CLI behavior, ask the developer to manually run the key command paths described by `docs/COMMANDS.md` and `docs/SPEC.md`. Agent-run tests can narrow risk, but they do not replace developer verification of the platform contract.

## 3. Test The Change

Choose tests based on what changed.

If you changed pure logic such as parsing, mapping, completion, output helpers, state helpers, or test utilities, run the unit-test target. It runs package-level unit tests and excludes snapshot tests:

```bash
make unit-test
```

If user-visible CLI output should stay unchanged, run:

```bash
make snapshot-check
```

If the command contract intentionally changes, regenerate snapshots:

```bash
make snapshot-update
```

Then review the `cli/testdata/snapshots/` diff manually according to `docs/SPEC.md` and `docs/REVIEW.md`. Golden files must be generated this way, not edited by hand.

The npm packaging helpers have their own unit tests:

```bash
make npm-test
```

Before opening or updating a pull request, run the shared local and CI check unless the change is documentation-only and does not affect generated files or code. It runs unit tests, snapshot checks, and npm packaging-script tests before any cross-build:

```bash
make pre-commit-check
```

## 4. Release Maintenance

Repository-wide publish triggers are defined in the root [Publish Workflows](../CONTRIBUTING.md#publish-workflows) section. CLI follows that split: `main` updates do not publish CLI artifacts; only an exact `vX.Y.Z` tag publishes npm packages. Do not create GitHub Release assets or use a GitHub Release to start other workflows.

CLI release automation has two entry points:

- `cli-pr.yml` runs `make pre-commit-check`, cross-builds six targets in separate jobs, packs all seven npm packages, and smoke-tests a Linux installation of the entry package. It does not stage or publish packages.
- `cli-release.yml` accepts only exact `vX.Y.Z` tags, runs the same pre-commit and packaging checks, then stages the npm packages through Trusted Publishing. It does not create or update a GitHub Release.

An exact release tag is the single formal-release trigger: it also starts the existing frontend, backend, storage, and Helm workflows, which publish immediately. Push the tag once and wait for npm staging and maintainer approval before pushing the next tag. Do not move a published release tag. A GitHub Release, if created, is only for human-written notes; it does not trigger workflows or carry CLI binaries.

The npm distribution consists of the entry package `@raids-lab/crater-cli` and these optional native packages:

- `@raids-lab/crater-cli-darwin-arm64`
- `@raids-lab/crater-cli-darwin-x64`
- `@raids-lab/crater-cli-linux-arm64`
- `@raids-lab/crater-cli-linux-x64`
- `@raids-lab/crater-cli-win32-arm64`
- `@raids-lab/crater-cli-win32-x64`

### Staged npm publication

All seven packages already exist on npm. Before the next release tag, configure a GitHub Actions Trusted Publisher for each package: organization `raids-lab`, repository `crater`, workflow filename `cli-release.yml`, no environment, and permission for `npm stage publish` only. The maintainer's npm account must have publish access and 2FA enabled. The workflow uses Node 24, npm 11.19.1, and GitHub OIDC; it does not use the bootstrap `NPM_TOKEN`.

The workflow packs all seven tarballs and smoke-tests an entry-package installation with the Linux x64 package before staging. Six independent platform jobs stage their packages first; the entry-package job runs only after all six succeed. A successful workflow means all packages are **staged**, not publicly installable. On npm, inspect the staged packages and approve each platform package with 2FA before approving `@raids-lab/crater-cli`. npm approves packages individually, so this is not an atomic release.

Do not use `npm view` to decide whether a staged version exists: staged versions are not public. GitHub's OIDC credential cannot run `npm stage list`. If a staging job fails after npm may have accepted its package, check the Staged Packages page before rerunning it; a repeated submission of the same package/version will conflict. After one complete staged release has been approved and verified, delete the `NPM_TOKEN` GitHub secret, revoke the bootstrap token, and set each package to require 2FA and disallow traditional token publishing.

## 5. Before Submitting

Check that:

- The relevant docs in `cli/docs/` agree with the implementation.
- Tests cover the risk introduced by the change.
- Golden files, if updated, were generated by `make snapshot-update` and reviewed manually.
- README files remain user-facing and do not contain internal development guidance.
- Agent Skills, if changed, follow `docs/SPEC.md` and describe how to use existing CLI contracts rather than defining new command behavior.
- Changes to APIs used by the CLI include an explicit current-version and minimum-backend-version decision.
