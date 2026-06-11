[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

# Contributing to Crater Helm Charts

This guide covers `charts/`, especially the `charts/crater` Helm Chart. Read the root [CONTRIBUTING.md](../CONTRIBUTING.md) first for repository workflow, branch rules, PR requirements, and cross-module constraints.

Use this file when you change chart metadata, `values.yaml`, Helm templates, dependencies, generated chart README content, or release versioning.

## Chart Files

- `charts/crater/Chart.yaml`: chart metadata and release version fields.
- `charts/crater/values.yaml`: user-facing chart configuration.
- `charts/crater/templates/`: Kubernetes manifests rendered by Helm.
- `charts/crater/README.md`: chart parameter documentation generated from metadata and values.

## Change Workflow

1. **Identify the user-facing configuration impact**. If you add, remove, rename, or change behavior/defaults for a value, treat it as a configuration change.
2. **Update the chart and docs together**. Changes to `values.yaml` parameters must be reflected in `charts/crater/README.md`.
3. **Bump the chart release fields when required**. Template, dependency, configuration-logic, or configuration-behavior changes require a version bump.
4. **Validate rendering and docs**. PRs trigger `.github/workflows/helm-chart-validate.yml` (`helm lint`, `helm template`, version bump check for release-impacting chart changes, and a packaging smoke test). After merging to `main`, `.github/workflows/helm-chart-publish.yml` publishes the chart to GHCR OCI. Run `helm lint` / `helm template` locally when useful.

## Versioning

`charts/crater/Chart.yaml` uses one shared release version. `version` and `appVersion` must always be exactly the same value for Crater releases; do not treat them as separate chart and application version numbers.

- Changes to chart templates, dependencies, or configuration logic must update both fields to the same new value.
- Changes that add, remove, rename, or change behavior/defaults of configuration items must update both fields to the same new value and update `charts/crater/README.md`.
- Use semantic versioning for the shared `version` / `appVersion` value.
- `values.yaml`-only configuration changes may use a patch bump unless the change also alters frontend/backend API compatibility.
- Frontend/backend API changes that affect the charted application contract require a minor bump.
- Do not proactively bump the major version unless the maintainer explicitly decides the release is a major breaking release.
- When the release version changes, remind the developer to create the corresponding GitHub tag, usually `v<version>`.

## Values And Documentation

- New configuration parameters need a clear description, default value, and usage guidance/example where useful.
- New `values.yaml` entries must include accurate, clear English comments that help users understand purpose and expected value.
- Prefer maintaining `charts/crater/README.md` with `helm-docs`.
- Do not let generated chart documentation drift from `Chart.yaml` or `values.yaml`.

## Quality And Security

- Keep value names semantic and consistent with existing naming.
- Keep templates simple, robust, and readable; use Helm helpers/pipelines when they make rendered manifests safer or clearer.
- Pay attention to resource requests/limits, security-sensitive defaults, and compatibility with Kubernetes objects already managed by the chart.
- Never commit real secrets, tokens, passwords, certificates, kubeconfigs, internal-only endpoints, or production credentials.
- Placeholder secrets in examples must be visibly fake, such as `<MUSTEDIT>` or `<MASKED>`.

## Before Submitting Chart Changes

- PRs that change `charts/**` automatically run the **Validate Helm Chart** workflow. After merging to `main`, the **Publish Helm Chart** workflow packages and pushes the chart to `oci://ghcr.io/raids-lab/crater`.
- The root pre-commit hook delegates staged `charts/**` changes to `cd charts && make pre-commit-check`. It requires a higher shared `version` / `appVersion` only when release-impacting chart files change (`Chart.yaml`, `values.yaml`, templates, dependencies, or `Chart.lock`).
- Run `helm lint crater/` and `helm template crater crater/ --dry-run` locally when useful; `charts/` does not currently have a dedicated `make` target.
- Include exact commands and results in the PR description when you ran local checks.
- Confirm `version` and `appVersion` are identical when either field changes.
- Confirm `charts/crater/README.md` reflects `values.yaml` changes.
