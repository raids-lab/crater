---
name: crater-cli-job
version: 0.4.0
description: "Use Crater CLI job commands to list, inspect, view logs, create, stop, and snapshot jobs."
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater job --help"
---

# Crater CLI Job

Use this skill when the user asks to operate Crater jobs from the CLI: list jobs, inspect details, view pods/logs/events/YAML/templates, get Jupyter/WebIDE access, open SSH, create jobs, stop/delete jobs, or snapshot jobs.

**CRITICAL — Before doing anything else, MUST read `crater-cli-shared` (possible path: [`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)) for global options, non-interactive use, errors, confirmation, and secret handling.**

## Command Map

- List jobs: `crater job ls [--search TEXT] [--page N] [--page-size N] [--all-pages]`
- Detail surfaces: `crater job get|events|yaml|template <jobName>`
- Pod list: `crater job pods <jobName> [--status STATUS] [--search TEXT] [--page N] [--page-size N] [--all-pages]`
- Logs: `crater job logs <jobName> [--pod POD | --all-pods] [-c CONTAINER | --all-containers]`
- Access helpers: `crater job token <jobName>`, `crater job secret <jobName>`, `crater job ssh <jobName>`
- Lifecycle helpers: `crater job snapshot <jobName>`, `crater job alert <jobName>`, `crater job delete <jobName>`
- Create interactive jobs: `crater job create jupyter|webide ...`
- Create custom jobs: `crater job create custom ...`
- Create distributed jobs: `crater job create tensorflow|pytorch --file request.json`

## Safe Defaults

Use `crater job ls --search <text> --json --no-interactive` before destructive actions to confirm the exact `jobName`. The list defaults to page 1 with 15 records (maximum page size 200); follow `data.pagination` rather than assuming the first page is complete. User-facing display names are not always accepted by job APIs.

Job list pagination is server-side. When `--owner`, `--from`, or `--to` is used, the CLI fetches all candidate server pages, applies those local filters, and re-paginates the filtered result. `--all-pages` starts at the first server page and omits `data.pagination`.

`job get|pods|logs|events|yaml` locate resources through the job API and preserve the backend's real namespace. Direct `crater pod ...` diagnostics require an explicit namespace; `crater node pods` requires either `--namespace` or `--all-namespaces`. Do not invent a fixed namespace for either path.

For create commands, validate resource values before calling the platform. CPU, memory, and GPU counts must not be negative; task replicas must be positive. Workspace mounts use `subPath:mountPath`; dataset mounts use `datasetID:mountPath`; forwards use `name:port`.

For Jupyter/WebIDE access commands, the returned token or password is sensitive. Prefer JSON only when the next tool needs structured fields, and avoid echoing secrets into logs or issue bodies.

## Job Status

`crater job ls --status` accepts `Prequeue`, `Pending`, `Running`, `Restarting`, `Completing`, `Completed`, `Aborting`, `Aborted`, `Terminating`, `Terminated`, `Failed`, `Deleted`, `Freed` and `Cancelled`. `crater job pods --status` filters pods instead and takes `Pending | Running | Succeeded | Failed | Unknown`.

- `Pending`: the job exists in the cluster but is not running. `status` alone does not say why; read `podGroupPhase` in the `--json` record: `Pending` means not admitted yet (queue quota or capacity), `Inqueue` means admitted and waiting for nodes, `Running` or `Completed` means nodes are assigned and containers are starting. These are PodGroup phases, not job statuses; never pass them to `--status`.
- `Running`: the job's pods are running. `Completed` and `Failed` are the final results; `Aborted` and `Terminated` mean volcano stopped the job by policy or on request. `Restarting`, `Completing`, `Aborting` and `Terminating` are the transitions into those states.
- `Deleted`: stopped by the user; resources are released and the record is kept. `Freed`: released automatically after long idle resource usage.
- `Prequeue`: only jobs submitted before the scheduler extender upgrade that are still being handed to the cluster; new jobs never enter it. `Cancelled`: accepted by the filter for historical records; the current backend does not produce it.

## Common Workflows

List running or pending PyTorch/TensorFlow jobs for a user:

```bash
crater job ls \
  --user alice \
  --search experiment \
  --status Running,Pending \
  --type pytorch,tensorflow \
  --all-pages \
  --json --no-interactive
```

Inspect a job:

```bash
crater job get jpt-alice-abcde --json --no-interactive
crater job pods jpt-alice-abcde --status Running --page-size 15 --json --no-interactive
crater job events jpt-alice-abcde --json --no-interactive
```

View recent logs for a single-pod job:

```bash
crater job logs sg-alice-abcde --tail 200 --timestamps --no-interactive
```

View every Pod in a distributed job:

```bash
crater job logs pyt-alice-abcde --all-pods --tail 100 --no-interactive
```

Follow one selected worker:

```bash
crater job logs pyt-alice-abcde \
  --pod pyt-alice-abcde-worker-0 \
  --container worker \
  --follow
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

`crater job create tensorflow|pytorch` intentionally uses `--file` because the backend accepts a nested `tasks[]` request. The CLI rejects unknown JSON fields. Keep the JSON aligned with the backend DTO fields: `name`, `tasks`, `resource`, `image.imageLink`, `volumeMounts`, `envs`, `selectors`, `alertEnabled`, and `template`.

Pagination and job filter validation are aggregated. If a JSON `usage_error` contains `context.issues`, fix all listed fields before retrying.

`crater job logs` automatically selects a single Pod and regular container. It never falls back to an init container: use `--container` or `--all-containers` to select init containers explicitly. For distributed jobs, explicitly choose `--pod` or `--all-pods`; for sidecar Pods, choose `--container` or `--all-containers`. `--follow` requires exactly one Pod and container and cannot be combined with `--json` or `--previous`. `--prefix` affects text output only; JSON already identifies each source with `pod` and `container` fields.
