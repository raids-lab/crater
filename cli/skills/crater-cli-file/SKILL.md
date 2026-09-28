---
name: crater-cli-file
version: 0.2.0
description: "Use Crater CLI to list and download ordinary-user remote files in user, public, and account storage spaces."
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater file --help"
---

# Crater CLI File

**CRITICAL — Before doing anything else, MUST read `crater-cli-shared` (possible path: [`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)) for global options, non-interactive use, errors, and sensitive information handling.**

Use `crater file` when a user needs to inspect or download files visible through their ordinary Crater identity.

## Supported workflow

- List visible storage roots: `crater file ls`
- List a nested directory: `crater file ls <remote-path>`
- Return structured data for a script or agent: add `--json --no-interactive`

Remote paths are logical Crater paths. They must start with `user`, `public`, or `account`; do not pass local filesystem paths or construct paths containing `.` or `..`.

- Download to the current directory using the remote basename:

  ```bash
  crater file download user/results/model.bin
  ```

- Choose an exact local file path:

  ```bash
  crater file download "account/共享数据/result.bin" ./downloads/result.bin
  ```

- Replace an existing local file only after the user explicitly asks for it:

  ```bash
  crater file download user/results/model.bin ./model.bin --overwrite
  ```

- Return structured metadata:

  ```bash
  crater file download user/results/model.bin ./model.bin --json --no-interactive
  ```

## Safety

- `file ls` is read-only.
- `file download` downloads one file; the optional local path is a file path, not a directory.
- Never add `--overwrite` unless replacing the exact local target is part of the user's request.
- Binary content is written only to the target file; JSON stdout contains metadata.
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
