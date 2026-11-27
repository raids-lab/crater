---
name: crater-cli-admin-job
version: 0.3.0
description: "Use Crater CLI as a platform administrator to list and manage jobs across users and projects and to execute commands in running jobs."
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater admin job --help"
---

# Crater CLI Admin Job

Use this skill when the user asks for administrator job operations from the CLI. Administrator commands use the `crater admin job ...` prefix. Do not use `--admin` on ordinary `crater job ...` commands.

**CRITICAL — Before doing anything else, MUST read `crater-cli-shared` (possible path: [`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)) for global options, non-interactive use, errors, confirmation, and secret handling.**

## Command Map

- List admin-visible jobs: `crater admin job ls [--search TEXT] [--page N] [--page-size N] [--all-pages]`
- Open a shell or execute a command in a running job: `crater admin job exec <jobName> [--pod POD] [-c CONTAINER] [-i] [-t] [--env-mode user|container] [-- COMMAND [ARG...]]`
- Delete a job as admin: `crater admin job delete <jobName>`
- Lock cleanup: `crater admin job lock <jobName> [--permanent | --days N | --hours N | --minutes N]`
- Unlock cleanup: `crater admin job unlock <jobName>`
- Toggle low-usage keep state: `crater admin job keep <jobName>`
- Cleanup jobs: `crater admin job clean waiting-jupyter|waiting-custom|long-running|low-gpu ...`

## Safe Defaults

Use `crater admin job ls --search <text> --json --no-interactive` before destructive actions to confirm the exact backend `jobName`. The list defaults to 15 records per page (maximum 200); follow `data.pagination`, or use `--all-pages` only when the complete admin-visible set is required. User-facing display names are not always accepted by job APIs.

The admin list uses server pagination and server-supported search/status/type/node/sort filters. When `--owner`, `--from`, or `--to` is present, the CLI fetches all candidate server pages, applies local filters, and re-paginates. Pagination and filter problems are aggregated in one `usage_error`; inspect `context.issues`.

Admin exec requires a platform administrator login and backend API v10 or later. Use `crater compatibility` to check version compatibility. Run only the commands the user authorizes in the requested job. Resolve the exact platform job name and owner with `crater admin job ls` when the target is uncertain.

Do not pass negative durations or cleanup thresholds. `lock` requires `--permanent` or at least one positive duration field. Cleanup commands require `--yes` for non-interactive use. Long-running cleanup requires both positive day thresholds; low-GPU cleanup requires positive lookback and wait minutes, with utilization between 0 and 100.

## Common Workflows

List a user's jobs as admin:

```bash
crater admin job ls --user alice --page-size 15 --json --no-interactive
```

Open the task user's login shell from a local terminal, or run an explicit command after `--` for scripted use.

```bash
crater admin job exec jpt-alice-abcde
crater admin job exec jpt-alice-abcde --json --no-interactive -- uname -m
```

Select `--pod` or `--container` explicitly when there are multiple candidates. For task user environment, stdin, TTY, and exit status behavior, see [Crater CLI job execution](../crater-cli-job/references/crater-cli-job-execution.md).

Use `crater admin operation-logs --operation-type AdminJobExec --json --no-interactive` to inspect accepted administrator exec requests and their targets.

Delete a job as admin:

```bash
crater admin job delete jpt-alice-abcde --yes --json --no-interactive
```

Lock a job for cleanup protection:

```bash
crater admin job lock jpt-alice-abcde --days 1 --json --no-interactive
```

Clean low GPU usage jobs:

```bash
crater admin job clean low-gpu --time-range 90 --wait-time 30 --util 10 --yes --json --no-interactive
```
