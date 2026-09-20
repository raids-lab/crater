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

Before opening or updating a pull request, run the full CLI test target unless the change is documentation-only and does not affect generated files or code. This target runs both unit tests and snapshot checks:

```bash
make test
```

The npm publishing helpers have their own unit tests:

```bash
make npm-test
```

`make pre-commit-check` is the local aggregate check. It runs both `make test` and `make npm-test`; CI keeps those responsibilities in separate jobs:

```bash
make pre-commit-check
```

## 4. Release Maintenance

Repository-wide publish triggers are defined in the root [Publish Workflows](../CONTRIBUTING.md#publish-workflows) section. CLI follows that split: `main` updates do not publish CLI artifacts; only an exact `vX.Y.Z` tag publishes npm packages. Do not create GitHub Release assets or use a GitHub Release to start other workflows.

CLI release automation has two entry points:

- `cli-pr.yml` runs `Check CLI` (`make test`) first, then `Check npm packaging` (npm packaging-script tests, a six-target cross-build, `npm pack`, and a Linux install of the entry package).
- `cli-release.yml` accepts only exact `vX.Y.Z` tags and publishes npm packages in platform-first order. It does not create or update a GitHub Release.

An exact release tag is the single formal-release trigger: it also starts the existing frontend, backend, storage, and Helm workflows. Push the tag once and wait for that formal release to finish before pushing the next tag. Do not move a published release tag. A GitHub Release, if created, is only for human-written notes; it does not trigger workflows or carry CLI binaries.

The npm distribution consists of the entry package `@raids-lab/crater-cli` and these optional native packages:

- `@raids-lab/crater-cli-darwin-arm64`
- `@raids-lab/crater-cli-darwin-x64`
- `@raids-lab/crater-cli-linux-arm64`
- `@raids-lab/crater-cli-linux-x64`
- `@raids-lab/crater-cli-win32-arm64`
- `@raids-lab/crater-cli-win32-x64`

### First npm publication

The packages must exist before npm trusted publishing or staged publishing can be configured. The bootstrap version of `cli-release.yml` therefore publishes directly and does not pause for approval. Before creating the first release tag:

1. Enable 2FA on the publishing npm account and confirm that it can publish public packages under `@raids-lab`.
2. Create a short-lived granular npm access token limited to the `@raids-lab` scope and the permissions required to publish these packages. Direct CI publication requires a token that can complete publication without an interactive OTP.
3. Store it as the repository Actions secret `NPM_TOKEN`. Never put the token in a file, command output, issue, PR, or workflow input.
4. Review the intended tag and the successful CLI PR checks. Pushing a matching tag starts the frontend, backend, storage, Helm, CLI, and irreversible npm publication workflows immediately.

The publisher is restart-safe for a partially completed bootstrap: it checks the registry and skips any package/version that is already public, then continues with the remaining platform packages and publishes the entry package last. A package/version that npm has accepted can never be reused.

### Migrate to staged trusted publishing

After all seven packages have completed their first publication, make a separate reviewed change before the next release:

1. With npm CLI 11.15 or newer and an npm session protected by 2FA, configure a GitHub trusted publisher for every package. Use repository `raids-lab/crater`, workflow file `cli-release.yml`, and grant `npm stage publish` only; do not grant direct `npm publish`.
2. Adapt the release publisher from direct `npm publish` to `npm stage publish`, remove the bootstrap-token requirement, and replace live-registry visibility waits with checks appropriate for staged submissions. Approve the six platform packages with 2FA before approving the entry package.
3. Verify one complete staged release, delete the `NPM_TOKEN` GitHub secret, revoke the temporary npm token, and configure each package to require 2FA and disallow traditional token publishing.

Do not make only the npm-side permission change while leaving the workflow on direct publication: the next release would fail after its build.

## 5. Before Submitting

Check that:

- The relevant docs in `cli/docs/` agree with the implementation.
- Tests cover the risk introduced by the change.
- Golden files, if updated, were generated by `make snapshot-update` and reviewed manually.
- README files remain user-facing and do not contain internal development guidance.
- Agent Skills, if changed, follow `docs/SPEC.md` and describe how to use existing CLI contracts rather than defining new command behavior.
- Changes to APIs used by the CLI include an explicit current-version and minimum-backend-version decision.
