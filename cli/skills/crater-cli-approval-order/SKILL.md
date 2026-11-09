---
name: crater-cli-approval-order
description: "Crater CLI 用户审批工单域：提交、编辑、取消、查看当前用户的审批工单。管理员审核动作必须使用 crater-cli-admin-approval-order。"
version: 0.2.0
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater order --help"
---

# Crater CLI Approval Order

Use this skill when the user wants to manage their own approval orders through `crater order ...`.

**CRITICAL — Before doing anything else, MUST read `crater-cli-shared` (possible path: [`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)) for pagination, JSON, non-interactive use, errors, confirmation, and secret handling.**

## Commands

- `crater order ls [--status Pending|Approved|Rejected|Canceled] [--type job|dataset] [--search TEXT] [--page N] [--page-size N] [--all-pages] --json`
- `crater order get <id> --json`
- `crater order by-name <name> --json`
- `crater order submit --name <name> --type job --reason <reason> --hours <n> --json`
- `crater order edit <id> [--name <name>] [--type job|dataset] [--reason <reason>] [--hours <n>] --json`
- `crater order cancel <id> --yes --json`

## Rules

- Do not use `--admin` with user approval commands.
- `order ls` and `order get` only read orders created by the active user, including when that user is a platform administrator. For cross-user reads as an administrator, use `crater admin order ls` or `crater admin order get <id>`. If `order get` returns `ERR_FORBIDDEN_403`, check `context.suggested_command` for an administrator command and use it only if the task calls for administrator access.
- `order by-name` returns orders only for jobs and datasets the active user can access. An empty list may mean there are no visible orders for that name.
- Do not approve, reject, or check platform-wide orders with this skill.
- `order edit` preserves fields that are not explicitly supplied by reading the current order first.
- Use `--yes` for `cancel` in non-interactive or JSON workflows.
- Prefer `--json` for automation and agent workflows.
- List output is filtered before local pagination and defaults to 15 records (maximum 200). Use `data.pagination` to request the next page; use `--all-pages` only when the full list is required.
- `--creator` is intentionally not available on the ordinary user list. It belongs only to `crater admin order ls`; use `--search` to match an order name or creator information within the current user's visible orders.
- Pagination, status, and type validation are aggregated. If stderr JSON contains `context.issues`, fix all fields before retrying.

## Escalation

If the task requires approving, rejecting, or checking invalid pending job orders, switch to `crater-cli-admin-approval-order` and use `crater admin order ...`.
