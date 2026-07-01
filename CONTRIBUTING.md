[English](CONTRIBUTING.md) | [简体中文](docs/zh-CN/CONTRIBUTING.md)

# Contributing to Crater

This document is the starting point for contributing to the Crater monorepo. It explains the repository-wide workflow first, then points you to the module-specific rules you need for the files you are changing.

If you are new to Crater, read this file in order once. During day-to-day work, use it as a checklist and jump into the relevant module `CONTRIBUTING` before editing code.

## How The Docs Fit Together

- **README** files are for users. They explain what Crater is, how to use it, and how to deploy it.
- **CONTRIBUTING** files are for contributors. This root file defines the repository-wide workflow; module files define module-specific engineering rules.
- **`.github/instructions/*`** files are Copilot review checklists. They reference `CONTRIBUTING` and may repeat important MUST/SHOULD checks, but they are not independent rule sources.
- **`skills/`** contains Agent workflows. Skills route AI agents through tasks and may repeat important rules, but development constraints that affect code, docs, APIs, UI, charts, or tests must live in `CONTRIBUTING` first.
- **`cli/docs/*`** contains the CLI command contract and implementation narrative. Start from `cli/CONTRIBUTING.md`.

## Choose The Right Module Guide

Before changing files, open the guide for the area you are touching:

| Change area | Required guide |
|-------------|----------------|
| `backend/` including `internal/storage/` | [backend/CONTRIBUTING.md](backend/CONTRIBUTING.md) |
| `frontend/` | [frontend/CONTRIBUTING.md](frontend/CONTRIBUTING.md) |
| `website/`, `docs/` | [website/CONTRIBUTING.md](website/CONTRIBUTING.md) |
| `cli/` | [cli/CONTRIBUTING.md](cli/CONTRIBUTING.md), then the relevant `cli/docs/*` file |
| `charts/` | [charts/CONTRIBUTING.md](charts/CONTRIBUTING.md) |

If a change crosses modules, follow every relevant module guide and the cross-module rules near the end of this file.

## First-Time Setup

Command blocks below fall into two categories:

- **Run in order**: execute sequentially during first-time setup, usually once (e.g. clone, configure remotes, install hooks).
- **Example commands**: illustrate everyday Git operations; replace placeholders (such as `YOUR_USERNAME`, `feature/your-feature-name`) and adapt to your context. **Do not copy and run an example block as-is.**

### Fork

Creating new development branches on the main repository (`raids-lab/crater`) and pushing directly to `main` are not allowed. You must fork the repository, develop on your fork, and open a Pull Request to merge into the main repository.

In your browser, open the main repository at [https://github.com/raids-lab/crater](https://github.com/raids-lab/crater), sign in to GitHub, and click **Fork** in the top-right corner to create a fork under your account.

### Clone and configure remotes

The main repository is `https://github.com/raids-lab/crater`. Task branches and PR branches must be pushed to your fork, not to the main repository. Do not push directly to `main` on the main repository, and do not create branches there.

**Recommended: clone the main repository and add `myfork` for your fork**

**Run in order** (first-time remote setup):

```bash
git clone https://github.com/raids-lab/crater.git
cd crater
git remote add myfork https://github.com/YOUR_USERNAME/crater.git
git remote -v
```

Convention: `origin` = main repository (`raids-lab/crater`), `myfork` = your fork (replace `YOUR_USERNAME` with your GitHub username).

**Example commands** (how to sync `main` and push task branches; adapt branch and remote names):

```bash
git checkout main
git fetch origin
git rebase origin/main
git push myfork main
```

```bash
git push myfork feature/your-feature-name
```

**Alternative: clone your fork, then add the main repository as `origin`**

If you cloned your fork first (so the default `origin` points to your fork), rename it and add the main repository.

**Run in order** (first-time remote setup):

```bash
git clone https://github.com/YOUR_USERNAME/crater.git
cd crater
git remote rename origin myfork
git remote add origin https://github.com/raids-lab/crater.git
git remote -v
```

Then pull from `origin` and push to `myfork` using the **example commands** under the recommended setup.

**Alternative: use only your fork as the remote**

If you prefer a single remote, clone your fork and keep the default `origin` pointing to it.

**Run in order** (clone the repository, usually once):

```bash
git clone https://github.com/YOUR_USERNAME/crater.git
cd crater
```

After **Sync fork** on GitHub, **example commands** (update local `main`):

```bash
git checkout main
git pull origin main
```

**Example commands** (push a task branch):

```bash
git push origin feature/your-feature-name
```

### Install Git hooks

Installs the repository pre-commit hook so `git commit` automatically delegates to `make pre-commit-check` in the affected sub-projects before the commit is created. The hook is primarily maintained for macOS and Linux POSIX shell environments; it may fail on Windows, so Windows contributors should run the relevant module checks manually or use WSL.

**Run in order** (once after cloning):

```bash
make install-hooks
```

## Local Debugging

### Topology

Crater is deployed as multiple Kubernetes-backed components, but local development usually does **not** mean recreating the whole cluster or starting every dependency locally. Frontend and backend are rarely useful in isolation; for local debugging, start at least both of them together. Start `backend`'s storage server only when the task touches storage-server behavior or storage paths.

Runtime configuration should point local services at the existing test-cluster services that the project already uses, such as Kubernetes, PostgreSQL, storage, registry, networking, or integrations. Do not start a separate local database or mock a full cluster unless the task explicitly requires isolated infrastructure work. Environment-specific values must come from a developer or administrator and stay out of Git.

### Configuration

Some backend/frontend runtime config is environment-specific and may require administrator-provided Kubernetes config, database, network, or integration credentials. Do not invent private config and never commit it.

You can keep local config in one directory and link it into the modules:

```text
config/
├── backend/
│   ├── .debug.env
│   ├── kubeconfig
│   └── debug-config.yaml
└── frontend/
    └── .env.development
```

Useful targets:

- `make config-link CONFIG_DIR=~/develop/crater/config`: create symlinks, backing up existing files as `.bak`
- `make config-status`: show config file status
- `make config-unlink`: remove symlinks only
- `make config-restore`: restore files from `.bak`

Module-specific setup details live in each module guide.

## Start A Change

Everyday Git operations use the same two categories: **run in order** and **example commands** (do not copy example blocks as-is).

### Understand the requirement before editing

For complex or cross-module work, outline the plan in an issue, design note, or PR discussion before implementation.

### Sync local `main` from the correct source

Do this before creating a task branch. Repeat the same freshness check before the final commit and before any push: if local `main` has fallen behind the main repository, update `main` first, then rebase the task branch onto the latest local `main`. Keep task branches linear; do not merge `main` into a task branch unless a maintainer explicitly asks for it.

With the dual-remote setup (`origin` + `myfork`), **example commands** (update local `main` from the main repository):

```bash
git checkout main
git fetch origin
git rebase origin/main
```

If you use only your fork as the remote, run **Sync fork** on GitHub first, then **example commands**:

```bash
git checkout main
git pull origin main
```

### Branch prefixes and commit types

Before creating a task branch, choose the **type** for your change from the table below and use the matching prefix in the branch name (such as `feature/` or `fix/`). Commit subjects use the same types.

| type | Branch prefix | Meaning |
|------|---------------|---------|
| `feat` | `feature/` | New feature |
| `fix` | `fix/` | Bug fix |
| `docs` | `docs/` | Documentation change |
| `style` | `style/` | Code style / formatting |
| `refactor` | `refactor/` | Refactoring |
| `test` | `test/` | Tests |
| `chore` | `chore/` | Build / tooling |

Commit subjects use the format `type: subject` or `type(scope): subject`, where `scope` is optional and can name the changed area, such as `docs(cli): add command examples`. This format applies only to the first line of the commit message; the full commit message may include a body and trailers such as `Signed-off-by`.

### Create a task branch locally

If you are on `main` or on a branch that does not match the task, pick a prefix from the table above and create or switch to a clear task branch. Unless a maintainer requests otherwise, use the form `<prefix><short-description>` with lowercase English and hyphens in the description (for example, `feature/add-job-submission-form`).

**Example commands**:

```bash
git checkout -b feature/your-feature-name
```

For an existing branch, rebase it onto the latest local `main` to keep history linear. Prefer rebase over merging `main` into a feature branch.

### Implement according to the module guide

Use the existing architecture, helper APIs, and style. Keep changes focused on the task.

## Core Engineering Rules

These rules apply in every module:

- **Build for real user needs**. Features should solve concrete user problems, not just add implementation surface. Keep user experience in mind when designing behavior, error messages, workflows, defaults, and documentation.
- **Use `make` for build, lint, and tests** when a Makefile target exists. Prefer module targets over direct `go`, `pnpm`, or `helm` commands.
- **Check Go before Go builds/tests**. For Go sub-projects such as `backend/` and `cli/`, run `go version` and make sure it matches the sub-project `go.mod` / module guide before running build, test, or local-run targets.
- **Write code comments in English**, consistent with existing naming style and architecture.
- **Use the project copyright owner in new license headers**. Starting from June 2026, new or updated file-level copyright headers and NOTICE files should use `The Crater Project Team, RAIDS-Lab` regardless of where the contributor comes from. Use the correct year for the file header: for example, a file first published in 2026 should use `Copyright 2026 The Crater Project Team, RAIDS-Lab`, while project-level notices may use a project year range such as `2023-2026`.
- **Never commit secrets**: no keys, tokens, passwords, internal IPs, kubeconfigs, certificates, or production credentials.
- **Ask when unsure**. If a rule or context is missing, clarify instead of guessing.
- **Suggest rule changes openly**. If an existing rule would hurt quality, architecture, or security in a specific case, point it out and propose updating the relevant document instead of silently violating it.

## Verify Before Commit Or Push

### Run the relevant automated checks

The root pre-commit check works on staged files. **Example commands** (replace `<your-files>` with actual paths):

```bash
git add <your-files>
make pre-commit-check
```

You can also run full checks inside a sub-project (enter the relevant directory as needed; **not** a single block to run end-to-end):

```bash
cd frontend && make pre-commit-check
cd backend && make pre-commit-check
cd website && make pre-commit-check
cd cli && make pre-commit-check
```

### Require developer manual verification

Automated or AI-run checks do not replace developer judgment. Before the final commit or any push, the developer must personally inspect the affected behavior, pages, commands, generated artifacts, or docs and decide whether the change is acceptable.

Agents may help by listing pages to open, roles to use, operations to perform, commands to try, or documents to read. Agents may keep a temporary task note with the checks they ran and the manual checks requested, then reuse those notes when preparing the PR description.

Documentation changes require manual reading by the developer before commit or push. Do not submit AI-generated docs that have not been read and judged by a developer; generated docs often miss important context, steps, or operational details. The developer should revise the docs directly or ask the Agent to adjust them before submission.

Before the final commit and before any push, confirm again that the task branch is based on the latest intended base. If `main` changed while the task was in progress, update local `main`, rebase the task branch, rerun the relevant checks, and re-review any conflict resolution.

## Commit And Push

If the PR has only one commit, we recommend making the commit message consistent with the branch name: replace `/` with `:` and `-` with spaces. For example, branch `docs/contributing-command-hints` becomes `docs:contributing command hints`; branch `feature/portal-add-job-form` becomes `feature:portal add job form`.

If the PR has multiple commits, write each commit subject in `type(scope): subject` format according to the table above.

New commit subjects must use one of these forms:

```text
type: subject
type(scope): subject
```

Allowed types are `feat`, `fix`, `docs`, `style`, `refactor`, `test`, and `chore`. Scope is optional, but useful for naming the changed area, such as `docs(cli): add command examples`.

The subject format applies only to the first line. If the commit needs more context, add a blank line after the subject and write a body. DCO sign-offs are commit trailers and should appear after the body, for example:

```text
docs: add project notice

Explain how Crater publishes license and notice files.

Signed-off-by: Your Name <you@example.com>
```

### Developer Certificate of Origin

By submitting a contribution to Crater, you agree that your contribution is licensed under the Apache License, Version 2.0, unless you explicitly state otherwise. You also represent that you have the right to submit the contribution.

DCO sign-off is recommended for all contributors and required for contributors who are not members of The Crater Project Team. By signing off a commit, you certify that you have the right to submit the contribution and agree that it is licensed under the Apache License, Version 2.0.

Use `git commit -s` to add a sign-off when committing:

```bash
git commit -s -m "feat(portal): add job submission form"
```

If you forgot the sign-off on your latest commit, add it with:

```bash
git commit --amend -s
```

Crater does not currently reject pull requests with an automated DCO workflow. Maintainers may still ask external contributors to add or confirm their sign-off before merging.

When squash-merging a PR, maintainers must preserve existing `Signed-off-by` lines in the final squash commit message, without duplicating them. Maintainers must not add a contributor's `Signed-off-by` line unless the same sign-off already appears in the contributor's commits or the contributor has explicitly confirmed it in the pull request. If a maintainer makes substantive changes while merging, the maintainer should add their own sign-off.

Automation and Agent workflows that create commits on behalf of a developer should use `git commit -s` by default, but must not add a sign-off silently. Before creating the commit, the Agent must explain that `Signed-off-by` is a DCO statement that the developer has the right to submit the contribution under the project license, then show the complete commit message, including every `Signed-off-by` line, for developer review. The proposed subject must also follow the commit subject format above.

After `make install-hooks`, `git commit` triggers the installed pre-commit hook. The hook checks staged files, enters the affected sub-projects, and runs their `make pre-commit-check` targets before allowing the commit. This automatic commit-time check is mainly designed for macOS and Linux; it may not work correctly on Windows. On Windows, run the affected modules' `make pre-commit-check` targets manually, or use WSL.

Specify files or directories explicitly and avoid `git add .`. **Example commands**:

```bash
git add backend/internal/handler/user.go
git commit -s -m "feat(portal): add job submission form"
git push myfork feature/your-feature-name
```

Push task branches only to your fork remote (`myfork` with the dual-remote setup; `origin` if your fork is the only remote).

## Open And Iterate On A Pull Request

AI assistance is welcome, but the developer remains responsible for understanding and controlling the change. Before a PR is created or updated, the developer must review the branch name, changed-file list, final diff, verification results, and PR description. If an Agent created the commit, it should show the actual branch name and committed file list before preparing or creating the PR.

The PR description must be **bilingual Markdown** and cover:

- **Intent**: one-sentence summary and motivation if any.
- **Core changes**: grouped by what changed, not merely by file.
- **Test verification**: only checks actually performed, clearly separating automation/AI checks from developer manual checks.
- **Screenshots**: required for frontend / UI changes, showing the affected interface state(s).
- **Other notes**: optional special risks, migration notes, rollout notes, or compatibility notes.
- **Related issues**: GitHub-recognizable references such as `Resolve #208`, one per line when applicable.

Keep the description short when the change is simple. Agent-specific PR description workflow lives in `skills/crater-devel-shared/SKILL.md`; the template and examples live in `skills/crater-devel-shared/references/pr-description-template.md`.

After pushing a branch, an Agent may create the PR directly when tooling such as `gh` is available, or output the PR description in a code block for the developer to use. In either case, the Agent must show the PR description to the developer and get confirmation before using it to create or update a PR.

Before opening or updating a PR, run the relevant local review path for the changed area when one exists, such as `skills/crater-devel-review/SKILL.md` for repository-wide Agent review or `cli/docs/REVIEW.md` for CLI changes. Treat review as an iteration step: answer valid findings with fixes and document any accepted residual risk instead of treating the first diff as final.

After creating the PR, check workflow status. The PR may need multiple rounds of iteration with Copilot review or human review. An Agent may fetch the PR link itself or ask the developer for it, inspect review comments, judge whether each suggestion is correct and worth changing, then propose a modification plan. Do not apply review-driven code changes until the developer has discussed and approved the plan.

## Cross-Module Rules

- **Front/back identity consistency**: Admin views call admin APIs only (URL / function name carries the `admin` prefix); regular users call user APIs. Both sides must correspond.
- **Sync docs on feature launch**: New features or significant changes must update `website/` docs accordingly.
- **Sync chart on config-structure change**: When a change touches configuration structure, update `charts/`, bump both `version` and `appVersion` in `charts/crater/Chart.yaml` to the exact same value, and synchronize `charts/crater/README.md`. Version bump level and GitHub tag reminders follow `charts/CONTRIBUTING.md`.

## Maintaining Development Conventions

When adding or changing development conventions, keep the source of truth clear:

- Put development constraints and expected engineering outcomes in the appropriate `CONTRIBUTING` document: root for repository-wide rules, module-level files for module-specific rules, and `cli/docs/*` for CLI behavior contracts.
- Use `.github/instructions/*` as review checklists that reference `CONTRIBUTING`; they may repeat important MUST/SHOULD checks, but must not become independent rule sources.
- Use `skills/` for Agent task routing, AI development workflow, and operational guidance. Skills should reference the relevant `CONTRIBUTING` documents, and may repeat important conventions as high-priority reminders.
- If a rule primarily controls the AI development process, it may live in the relevant Skill; if it affects resulting code, docs, APIs, UI, charts, or tests, record it in `CONTRIBUTING` first.

## Agent Skills

The repository ships developer Agent Skills under `skills/` that guide AI agents through Crater development tasks. See [skills/README.md](skills/README.md) for their scope, boundaries, and per-skill responsibilities.

List available skills from GitHub:

```bash
npx skills add https://github.com/raids-lab/crater/tree/main/skills -l
```

Install all Crater developer skills globally for supported agents:

```bash
npx skills add https://github.com/raids-lab/crater/tree/main/skills -g --all
```
