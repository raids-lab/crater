---
name: crater-cli-job
version: 0.1.1
description: "Use Crater CLI job commands to list, inspect, create, stop, and snapshot jobs."
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater job --help"
---

# Crater CLI Job

Use this skill when the user asks to operate Crater jobs from the CLI: list jobs, inspect details, view pods/events/YAML/templates, get Jupyter/WebIDE access, open SSH, create jobs, stop/delete jobs, or snapshot jobs.

**CRITICAL — Before doing anything else, MUST read `crater-cli-shared` (possible path: [`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)) for global options, non-interactive use, errors, confirmation, and secret handling.**

## Command Map

- List jobs: `crater job ls`
- Detail surfaces: `crater job get|pods|events|yaml|template <jobName>`
- Access helpers: `crater job token <jobName>`, `crater job secret <jobName>`, `crater job ssh <jobName>`
- Lifecycle helpers: `crater job snapshot <jobName>`, `crater job alert <jobName>`, `crater job delete <jobName>`
- Create interactive jobs: `crater job create jupyter|webide ...`
- Create custom jobs: `crater job create custom ...`
- Create distributed jobs: `crater job create tensorflow|pytorch --file request.json`

## Safe Defaults

Use `crater job ls --json --no-interactive` before destructive actions to confirm the exact `jobName`. User-facing display names are not always accepted by job APIs.

For create commands, validate resource values before calling the platform. CPU, memory, and GPU counts must not be negative; task replicas must be positive. Workspace mounts use `subPath:mountPath`; dataset mounts use `datasetID:mountPath`; forwards use `name:port`.

For Jupyter/WebIDE access commands, the returned token or password is sensitive. Prefer JSON only when the next tool needs structured fields, and avoid echoing secrets into logs or issue bodies.

## Common Workflows

List running GPU jobs for a user:

```bash
crater job ls --user alice --status Running --json --no-interactive
```

Inspect a job:

```bash
crater job get jpt-alice-abcde --json --no-interactive
crater job pods jpt-alice-abcde --json --no-interactive
crater job events jpt-alice-abcde --json --no-interactive
```

Create a Jupyter job:

```bash
crater job create jupyter \
  --name experiment-notebook \
  --image harbor.example/project/jupyter:latest \
  --cpu 4 \
  --memory 16Gi \
  --gpu 1 \
  --gpu-resource nvidia.com/gpu \
  --json --no-interactive
```

Create a distributed PyTorch job from an exact backend-compatible request:

```bash
crater job create pytorch --file pytorch-job.json --json --no-interactive
```

Stop or delete a job:

```bash
crater job delete jpt-alice-abcde --yes --json --no-interactive
```

## Notes

`crater job create tensorflow|pytorch` intentionally uses `--file` because the backend accepts a nested `tasks[]` request. The CLI rejects unknown JSON fields. Keep the JSON aligned with the backend DTO fields: `name`, `tasks`, `resource`, `image.imageLink`, `volumeMounts`, `envs`, `selectors`, `alertEnabled`, `template`, and optional scheduling fields. Distributed TensorFlow and PyTorch jobs do not support backfill scheduling.
