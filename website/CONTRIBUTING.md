[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

# Contributing to Crater Documentation

This guide covers product documentation in `website/` and developer documentation in repository Markdown files such as `docs/` and module-level `.md` files. Read the root [CONTRIBUTING.md](../CONTRIBUTING.md) first for repository workflow, PR requirements, and manual documentation review rules.

Use this file when you add, move, or edit user-facing docs, deployment docs, troubleshooting docs, contributor docs, or generated documentation references.

## Choose The Right Documentation Home

Start every documentation change by deciding who the reader is.

- **`website/`** is for platform users: cluster users and cluster administrators. Administrators are still platform users here. Put deployment, usage, administration, troubleshooting, and operational guidance here. Avoid code-level implementation details unless they are needed to explain observable behavior.
- **`docs/` and module-level Markdown** are for developers and contributors who work with the code. Put architecture notes, development workflows, implementation reasoning, maintenance instructions, and contributor-facing design notes here.
- **README files** stay user-facing and concise. Do not turn README files into internal development manuals.

If a document is in the wrong place, move it rather than duplicating competing explanations.

## Running The Website Locally

For `website/` changes:

- Node.js v22+ and pnpm v10+ are expected.

```bash
cd website
pnpm install
make run
```

Common targets:

| Command | Purpose |
|---------|---------|
| `make run` | Start the local docs site |
| `make build` | Build the docs site |
| `make pre-commit-check` | Run docs checks before submitting |

## Writing Rules

- Keep docs accurate to current code and chart behavior.
- Write for the target reader. Platform users need operational steps and observable behavior; contributors need source paths, architecture, and maintenance context.
- Prefer clear procedures over broad descriptions. Include prerequisites, expected results, and rollback/troubleshooting notes when useful.
- Avoid leaking secrets, internal-only endpoints, private cluster names, or credentials.
- Documentation changes must be read and checked manually by the developer before commit or push, as required by the root CONTRIBUTING.

## Terminology

Use Crater terms consistently:

- **Account** / **账户** refers to the scheduling queue/accounting concept in Crater. Do not use it as a generic login account term.
- When a term has established product meaning, keep it consistent across website docs, UI text, and contributor docs.

## Chart Version Placeholders

For `website/` Helm deployment/install/upgrade commands involving `oci://ghcr.io/raids-lab/crater`, never hardcode chart versions.

- Put `<CraterChartVersionNotice />` near deployment commands that require a chart version.
- Use `<chart-version>` inside bash/yaml examples.
- Do not write literal examples such as `--version 1.2.3`.
- Chart configuration detail pages should use `<ChartBadge />`.

`docs/` and contributor docs do not have the same website component injection. Prefer placeholders such as `<chart-version>` when mentioning deployment commands there.

## Before Submitting Documentation Changes

- Confirm the document is in the right place for its reader.
- Run `make pre-commit-check` in `website/` when touching the docs site.
- Check links, examples, terminology, and chart version placeholders.
- Ask the developer to manually read the changed docs and judge whether key context, steps, and operational details are complete.
- If frontend/UI docs changed, include screenshots in the PR when relevant to the user-visible behavior.
