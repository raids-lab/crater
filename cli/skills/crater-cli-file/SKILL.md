---
name: crater-cli-file
version: 0.4.0
description: "Use Crater CLI to list files and upload one regular file in user, public, and account storage spaces."
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater file --help"
---

# Crater CLI File

**CRITICAL — Before doing anything else, MUST read `crater-cli-shared` (possible path: [`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)) for global options, non-interactive use, errors, and sensitive information handling.**

Use `crater file` when a user needs to inspect or upload files visible through their ordinary Crater identity.

## Supported workflow

- List visible storage roots: `crater file ls`
- List a nested directory: `crater file ls <remote-path>`
- Return structured data for a script or agent: add `--json --no-interactive`

Remote paths are logical Crater paths. They must start with `user`, `public`, or `account`; do not pass local filesystem paths or construct paths containing `.` or `..`.

## Safety

- `file ls` is read-only.
- Do not ask the user to provide a token or Keyring content.
- Do not substitute `crater admin ...` endpoints for an ordinary-user request.
- Prefer exact paths shown by a previous `file ls` result.

## Examples

```bash
crater file ls --json --no-interactive
crater file ls user/projects --json --no-interactive
crater file ls "account/共享数据" --json --no-interactive
```

## Troubleshooting

1. Run `crater auth ls --json` and confirm an active context exists.
2. Use `crater file ls --help` to verify the local binary supports the command.
3. A path validation error means the path is outside the ordinary-user logical roots or contains an unsafe segment.
4. For API errors, inspect `category`, `code`, and `context.http_status` from JSON stderr without exposing credentials.

## Upload a single file

Use `crater file upload` when a user wants to copy one local regular file into Crater storage.

### Upload workflow

- Create a new remote file:

  ```bash
  crater file upload ./train.py user/jobs/train.py
  ```

- Upload a binary file to current-account storage:

  ```bash
  crater file upload ./weights.bin "account/模型/weights.bin"
  ```

- Replace an existing regular remote file only after the user explicitly asks for it:

  ```bash
  crater file upload ./train.py user/jobs/train.py --overwrite
  ```

- Return structured metadata:

  ```bash
  crater file upload ./train.py user/jobs/train.py --json --no-interactive
  ```

### Upload safety

- The local path must resolve to one open regular file. Directories, devices, sockets, and pipes are rejected before any API request.
- Remote paths must start with `user`, `public`, or `account` and must name an entry below that root.
- Never add `--overwrite` unless replacing that exact remote target is part of the user's request.
- The server stages the complete stream in the target directory and atomically publishes it. A failed transfer never exposes a partial new file or truncates the previous file.
- Parent directories are never created automatically.
- This command uploads one file only. Do not pass a directory or shell glob.
- JSON stdout contains metadata only; it never includes file bytes.
- Do not ask the user to provide a token or Keyring content.

### Upload troubleshooting

1. Run `crater auth ls --json` and confirm an active context exists.
2. Use `crater file upload --help` to verify the local binary supports the command.
3. If the target exists, choose a new path or obtain explicit permission to add `--overwrite`.
4. A `404` from `/api/ss/upload` can indicate an older storage service or incorrect routing. Check the deployed service and route; this command requires API contract 2 and never falls back to WebDAV PUT.
5. For API errors, inspect `category`, `code`, and `context.http_status` from JSON stderr without exposing credentials.

## Create and move entries

- Create exactly one directory: `crater file mkdir user/jobs/new-run`. Its parent must exist.
- Move one entry to an exact destination: `crater file mv user/jobs/train.py user/archive/train.py`.
- The destination must not exist. There is no overwrite mode for `mv`.
- Do not move an entry to itself or below itself. Unsupported atomic no-clobber rename fails safely.
- These commands require backend API contract 3. Inspect JSON error metadata for permission, missing-parent, or destination-conflict errors.

## Remove an entry

- Remove one file: `crater file rm user/results/old.bin`.
- Removing any directory, including an empty one, requires `--recursive`. Recursive removal requires explicit user authorization and `--recursive`; it can partially complete before an error.
- JSON and non-interactive removal require `--yes`. Never add it without user authorization for the exact target.
- Logical storage roots cannot be removed. Symlinks are removed as entries; their targets are not followed.
- The safe remove endpoint requires backend API contract 4. Never fall back to the legacy `/delete` endpoint.
