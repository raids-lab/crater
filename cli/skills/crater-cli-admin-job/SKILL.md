---
name: crater-cli-admin-job
version: 0.1.1
description: "Use Crater CLI admin job commands to list, delete, lock, keep, and clean jobs."
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater admin job --help"
---

# Crater CLI Admin Job

Use this skill when the user asks for administrator job operations from the CLI. Administrator commands use the `crater admin job ...` prefix. Do not use `--admin` on ordinary `crater job ...` commands.

**CRITICAL — Before doing anything else, MUST read `crater-cli-shared` (possible path: [`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)) for global options, non-interactive use, errors, confirmation, and secret handling.**

## Command Map

- List admin-visible jobs: `crater admin job ls`
- Delete a job as admin: `crater admin job delete <jobName>`
- Lock cleanup: `crater admin job lock <jobName> [--permanent | --days N | --hours N | --minutes N]`
- Unlock cleanup: `crater admin job unlock <jobName>`
- Toggle low-usage keep state: `crater admin job keep <jobName>`
- Cleanup jobs: `crater admin job clean waiting-jupyter|waiting-custom|long-running|low-gpu ...`

## Safe Defaults

Use `crater admin job ls --json --no-interactive` before destructive actions to confirm the exact backend `jobName`. User-facing display names are not always accepted by job APIs.

Do not pass negative durations or cleanup thresholds. `lock` requires `--permanent` or at least one positive duration field. Cleanup commands require `--yes` for non-interactive use. Long-running cleanup requires both positive day thresholds; low-GPU cleanup requires positive lookback and wait minutes, with utilization between 0 and 100.

## Common Workflows

List a user's jobs as admin:

```bash
crater admin job ls --user alice --json --no-interactive
```

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
