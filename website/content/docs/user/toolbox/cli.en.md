---
title: Connect and Execute Commands with the CLI
description: Open a job shell, connect over SSH, or run automated commands using Crater CLI.
---

After signing in to Crater CLI, use `crater job ls` to find the platform job name, then connect to your running job. Replace `JOB` below with that name. These commands support CPU-only nodes, domestic accelerators and other supported nodes. `job exec` requires a backend providing the feature (API v7), with no local kubectl, kubeconfig or container SSH service needed.

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

Exec selects the task user from `NB_USER` or uses the current container user. HOME and shell come from the container’s user database. Each shell controls its startup files.

Exec starts with the container environment and loads the task user’s shell configuration. Configure variables needed by the command in the container environment or shell startup files. Commands retain the container working directory when accessible, otherwise HOME. Interactive shells start in HOME.

Non-TTY commands first probe the user's exported shell environment (up to 10 seconds and 1 MiB), discard startup output, then execute the original arguments. If configuration blocks or you only need container settings, use:

```bash
crater job exec JOB --env-mode container -- env
```

This mode retains the task user and HOME. Aliases and shell functions require an explicit shell command. Custom images bypassing platform startup must prepare the task user in the image or at job startup. Images also need `/bin/sh`, `env -0`, basic utilities such as dd/wc, and runuser, sudo or su for user switching. Missing users or failed identity switches stop execution.

## WebIDE and Jupyter Jobs

`job exec` starts a new process in the same job container. WebIDE, Jupyter and existing Notebooks continue running. It uses the task user's environment without reading WebIDE's selected Python interpreter or attaching to a Notebook kernel. Select a specific Conda environment or interpreter explicitly:

```bash
crater job exec JOB -- conda run -n myenv python script.py
crater job exec JOB -- /opt/conda/envs/myenv/bin/python script.py
```


## Use Local SSH

Configure the container SSH service and public key using the [SSH guide](./ssh/ssh-func.md), then run:

```bash
crater job ssh JOB
crater job ssh JOB --connect
crater job ssh JOB --connect --identity-file ~/.ssh/id_ed25519
crater job ssh JOB --connect --user USER
```

`job ssh` prints the address. With `--connect`, local OpenSSH connects using your SSH configuration and host-key checks. The platform login username is the default. Set `--user` for a different container account and `--identity-file` to select a key. Interactive connections require a local terminal.

If the image permits SSH password authentication and you have neither a usable key nor a container password, enter the job with `crater job exec JOB`. Run `sudo passwd "$(id -un)"` in the container and enter the password at the terminal prompt. Root can run `passwd` directly. Exec works without a container password, while setting one requires root or usable sudo. This password belongs to the container user and is separate from the platform LDAP password. Connect with `crater job ssh JOB --connect` and specify `--user` if needed.

SSH retains platform administrator access to jobs, while ordinary users require ownership in the current account. `job exec` always requires ownership in the current account. SSH automatically selects the job's main container even when sidecars are injected into the Pod.
