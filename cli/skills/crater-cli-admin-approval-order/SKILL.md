---
name: crater-cli-admin-approval-order
description: "Crater CLI 管理员审批工单域：通过 crater admin order ... 查看、批准、拒绝、清理审批工单。仅在用户明确要求管理员审核时使用。"
version: 0.3.0
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater admin order --help"
---

# Crater CLI Admin Approval Order

Use this skill only for administrator approval-order workflows.

**CRITICAL — Before doing anything else, MUST read `crater-cli-shared` (possible path: [`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)) for pagination, JSON, non-interactive use, errors, confirmation, and secret handling.**

## Commands

- `crater admin order ls [--status Pending|Approved|Rejected|Canceled] [--type job|dataset] [--creator TEXT] [--search TEXT] [--page N] [--page-size N] [--all-pages] --json`
- `crater admin order get <id> --json`
- `crater admin order approve <id> --json`
- `crater admin order approve <id> --lock --days <n> --hours <n> --minutes <n> --json`
- `crater admin order approve <id> --lock --permanent --json`
- `crater admin order reject <id> --review-notes <reason> --json`
- `crater admin order check --yes --json`

## Rules

- Admin commands always use the `crater admin order ...` prefix. Do not use `--admin`.
- Use `crater admin order get <id>` to inspect the job's resource requests and order history before reviewing it. History shows the current request number, each request's status, requested extension, applied lock when recorded, and time to review. With `--json`, read `data.order`, `data.job`, and `data.history`. When an enrichment is unavailable, `data.job_error` or `data.history_error` explains why.
- `reject` requires `--review-notes`.
- `approve` approves and locks a running job in one operation. It uses the requested hours by default. To override, pass `--lock` with a positive duration or `--permanent`. Read the applied lock from `data.approved_lock`.
- Lock duration values must be non-negative. Permanent and finite durations are mutually exclusive.
- The CLI does not send `reviewerID`; the backend derives the reviewer from the active token.
- The server preserves the original request and records the applied lock separately. Job approval requires backend API 7.
- Lists are filtered and then locally paginated. They are sorted with pending orders first and newest orders first within each status, default to 15 records (maximum 200), and expose `data.pagination`.
- `--creator` is admin-only and matches creator username or display name. `--search` also matches the order name and creator information.
- Pagination, status, and type validation are aggregated. If stderr JSON contains `context.issues`, fix all fields before retrying.

## Safety

Do not use this skill for ordinary user order submission or cancellation. Use `crater-cli-approval-order` for user-owned approval-order workflows.
