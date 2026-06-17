---
name: crater-cli-admin-job
version: 0.1.0
description: Use Crater CLI admin job commands to list, delete, lock, keep, and clean jobs.
---

# Crater CLI Admin Job

Use this skill when the user asks for administrator job operations from the CLI. Administrator commands use the `crater admin job ...` prefix. Do not use `--admin` on ordinary `crater job ...` commands.

First apply the shared Crater CLI rules from `crater-cli-shared`: prefer `--json --no-interactive` for automation, treat stderr JSON as the error contract, and confirm exact `jobName` values before destructive operations.

## Command Map

- List admin-visible jobs: `crater admin job ls`
- Delete a job as admin: `crater admin job delete <jobName>`
- Lock cleanup: `crater admin job lock <jobName> [--permanent | --days N | --hours N | --minutes N]`
- Unlock cleanup: `crater admin job unlock <jobName>`
- Toggle low-usage keep state: `crater admin job keep <jobName>`
- Cleanup jobs: `crater admin job clean waiting-jupyter|waiting-custom|long-running|low-gpu ...`

## Safe Defaults

Use `crater admin job ls --json --no-interactive` before destructive actions to confirm the exact backend `jobName`. User-facing display names are not always accepted by job APIs.

Do not pass negative durations or cleanup thresholds. `lock` requires `--permanent` or at least one positive duration field.

## Common Workflows

List a user's jobs as admin:

```bash
crater admin job ls --user alice --json --no-interactive
```

Delete a job as admin:

```bash
crater admin job delete jpt-alice-abcde --json --no-interactive
```

Lock a job for cleanup protection:

```bash
crater admin job lock jpt-alice-abcde --days 1 --json --no-interactive
```

Clean low GPU usage jobs:

```bash
crater admin job clean low-gpu --time-range 3600 --wait-time 600 --util 10 --json --no-interactive
```
