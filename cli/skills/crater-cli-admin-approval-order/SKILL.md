---
name: crater-cli-admin-approval-order
description: "Crater CLI 管理员审批工单域：通过 crater admin order ... 查看、批准、拒绝、清理审批工单。仅在用户明确要求管理员审核时使用。"
version: 0.2.0
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
- Use `crater admin order get <id>` to inspect the job's resource requests and order history before reviewing it. History shows the current request number, each request's status and requested extension, and time to review when known. Requested hours are the submitter's request. With `--json`, read `data.order`, `data.job`, and `data.history`. When an enrichment is unavailable, `data.job_error` or `data.history_error` explains why.
- `reject` requires `--review-notes`.
- `approve --lock` first reads the order detail, locks the target job, and only then reviews the order.
- Lock duration values must be non-negative. Unless `--permanent` is set, `--lock` requires a positive duration.
- The CLI does not send `reviewerID`; the backend derives the reviewer from the active token.
- The review API only updates status and review notes. It does not overwrite the original order content.
- Lists are filtered and then locally paginated. They are sorted with pending orders first and newest orders first within each status, default to 15 records (maximum 200), and expose `data.pagination`.
- `--creator` is admin-only and matches creator username or display name. `--search` also matches the order name and creator information.
- Pagination, status, and type validation are aggregated. If stderr JSON contains `context.issues`, fix all fields before retrying.

## Safety

Do not use this skill for ordinary user order submission or cancellation. Use `crater-cli-approval-order` for user-owned approval-order workflows.
