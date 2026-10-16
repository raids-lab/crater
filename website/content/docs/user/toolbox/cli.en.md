---
title: Connect and Execute Commands with the CLI
description: Open a job shell, connect over SSH, or run automated commands using Crater CLI.
---

After signing in to Crater CLI, use `crater job ls` to find the platform job name, then connect to your running job. Replace `JOB` below with that name. These commands support CPU-only nodes, domestic accelerators and other supported nodes. `job exec` requires a backend providing the feature (API v4), with no local kubectl, kubeconfig or container SSH service needed.

## Open Your Shell

Run in a local terminal:

```bash
crater job exec JOB
```

The CLI forwards input, allocates a terminal, tracks its size, and opens the task user's interactive login shell in their HOME. It restores the local terminal on exit. For multiple Pods or containers, select `--pod POD` and `--container CONTAINER`; use `crater job pods JOB` to inspect candidates.

## Run Commands and Scripts

```bash
crater job exec JOB -- python -V
crater job exec JOB --pod POD -c CONTAINER -- uname -m
cat script.sh | crater job exec JOB -i -- sh
crater job exec JOB -it -- python
crater job exec JOB --json --no-interactive -- python -V
```

Place the command and arguments after `--`; argument boundaries are preserved. For pipelines, variable expansion or redirection, explicitly use `-- sh -c '...'`. `-i` forwards stdin. `-t` allocates a terminal and requires `-i` and local terminal input/output; it cannot be combined with `--json` or `--no-interactive`.

Ordinary mode streams stdout and stderr separately; TTY mode merges them. Successful JSON results appear in `data.exec`, containing the target, stdout, stderr and `exit_code`, with a combined output limit of 16 MiB. Redirect ordinary output for large or binary results. A nonzero remote exit makes the CLI return status 4; the remote status is available in the error's `context.exit_code`.

## User Environment

SSH connections and exec share user environment handling, so you do not need to invoke `/crater-start.sh`. The container's `NB_USER` selects the task user; otherwise the current container user is used. HOME and shell come from the container's user database. Each shell follows its own startup rules; a Bash login profile determines whether `.bashrc` is sourced.

Platform startup scripts save exported initialization settings, and new sessions load user configuration. If saving fails, startup logs report `could not save startup environment` and the job continues starting. New sessions then use container settings and user configuration; fix snapshot saving and recreate the job when startup-only generated values are needed. Temporary exports or Conda activation in another terminal are not copied. Commands retain the container working directory when accessible, otherwise HOME; interactive shells start in HOME.

Non-TTY commands first probe the user's exported shell environment (up to 10 seconds and 1 MiB), discard startup output, then execute the original arguments. If configuration blocks or you only need container settings, use:

```bash
crater job exec JOB --env-mode container -- env
```

This mode retains the task user and HOME. Aliases and shell functions require an explicit shell command. Custom images bypassing platform startup must prepare the task user in the image or at job startup. Images also need `/bin/sh`, `env -0`, basic utilities such as dd/wc, and runuser, sudo or su for user switching. Missing users or failed identity switches stop execution. Older jobs can use container settings and user configuration; recreate a job with the updated deployment when newly saved startup settings are needed.

## WebIDE and Jupyter Jobs

`job exec` starts a new process in the same job container. WebIDE, Jupyter and existing Notebooks continue running. It uses the task user's environment without reading WebIDE's selected Python interpreter or attaching to a Notebook kernel. Select a specific Conda environment or interpreter explicitly:

```bash
crater job exec JOB -- conda run -n myenv python script.py
crater job exec JOB -- /opt/conda/envs/myenv/bin/python script.py
```

Jupyter's notebook directory and WebIDE's default working directory remain `/home/USERNAME`. If an existing image user has a different HOME, startup prepares both directories: applications retain their working directory and shell configuration stays in the user's own HOME. The platform web terminal retains its existing endpoint and terminal behavior.

## Use Local SSH

Configure the container SSH service and public key using the [SSH guide](./ssh/ssh-func.md), then run:

```bash
crater job ssh JOB
crater job ssh JOB --connect
crater job ssh JOB --connect --identity-file ~/.ssh/id_ed25519
crater job ssh JOB --connect --user USER
```

Without `--connect`, the command only requests the address and does not install a shared session launcher; the web SSH UI and older clients retain this behavior. With it, the CLI explicitly requests the user environment launcher, then local OpenSSH connects using your SSH configuration and normal host-key checks. The backend-confirmed task user is the default. `--user` and `--identity-file` require `--connect`; an alternative user must have access to the shared session launcher. Interactive connections require local terminal input/output and cannot use `--json` or `--no-interactive`. Use `job exec` for images without sshd.

SSH retains platform administrator access to jobs, while ordinary users require ownership in the current account. `job exec` always requires ownership in the current account. SSH automatically selects the job's main container even when sidecars are injected into the Pod.
