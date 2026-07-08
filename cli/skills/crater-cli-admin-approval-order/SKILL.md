---
name: crater-cli-admin-approval-order
description: "Crater CLI 管理员审批工单域：通过 crater admin order ... 查看、批准、拒绝、清理审批工单。仅在用户明确要求管理员审核时使用。"
version: 0.1.0
metadata:
  cliHelp: "crater admin order --help"
---

# Crater CLI Admin Approval Order

Use this skill only for administrator approval-order workflows.

## Commands

- `crater admin order ls --json`
- `crater admin order get <id> --json`
- `crater admin order approve <id> --json`
- `crater admin order approve <id> --lock --days <n> --hours <n> --minutes <n> --json`
- `crater admin order approve <id> --lock --permanent --json`
- `crater admin order reject <id> --review-notes <reason> --json`
- `crater admin order check --yes --json`

## Rules

- Admin commands always use the `crater admin order ...` prefix. Do not use `--admin`.
- `reject` requires `--review-notes`.
- `approve --lock` first reads the order detail, locks the target job, and only then reviews the order.
- Lock duration values must be non-negative. Unless `--permanent` is set, `--lock` requires a positive duration.
- The CLI does not send `reviewerID`; the backend derives the reviewer from the active token.
- The review API only updates status and review notes. It does not overwrite the original order content.

## Safety

Do not use this skill for ordinary user order submission or cancellation. Use `crater-cli-approval-order` for user-owned approval-order workflows.
