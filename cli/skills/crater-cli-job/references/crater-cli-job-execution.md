# Connect to a job and execute commands

Apply `crater-cli-shared` first. Confirm the platform `jobName` using `job ls` or `job get`; do not substitute a display name. Container commands may change data or processes, so stay within the user's requested operation.

For automation, run an explicit command with structured output:

```bash
crater job exec cpu-job --json --no-interactive -- uname -m
crater job exec training-job --pod training-job-worker-0 -c worker --json -- python -V
```

The command and its arguments follow `--`. Preserve their boundaries; shell operators are not interpreted. The backend prepares the task user and its exported shell environment before executing argv. Use `-- sh -c '...'` only when shell evaluation is intended. `--stdin` / `-i` forwards input, including piped data. EOF closes only the input stream; the command can still produce output.

`data.exec` contains the server-confirmed namespace, Pod, container, stdout, stderr, and exit code. JSON capture is limited to 16 MiB; use ordinary output with redirection for large or binary results. Use `job pods` to identify Pod names; select `--pod` and `--container` explicitly when there are multiple candidates. These flags select existing containers and do not alter scheduling or resource allocation.

For a human terminal session:

```bash
crater job exec cpu-job
crater job exec cpu-job --env-mode container -- env
crater job ssh training-job --connect
crater job ssh training-job --connect --user alice --identity-file ~/.ssh/id_ed25519
```

`-t` requires `-i` and a local terminal. Interactive exec and SSH connection cannot use `--json` or `--no-interactive`. TTY output combines stderr and stdout. `job ssh` prints the address or returns it in `data.ssh` for JSON output. `--connect` invokes local OpenSSH for a native sshd login. SSH preserves platform administrator access and selects the main container despite injected sidecars. Exec requires ownership in the current account and explicit selection for multiple containers.

Exec requires backend API v7. It uses the platform's Kubernetes connection and does not require local kubectl, a kubeconfig, sshd, bash, GPUs, or NVIDIA tooling. It can target CPU, ARM, and domestic accelerator nodes. The requested executable must exist in the image. Direct SSH uses the local OpenSSH client and the image's SSH service; retain normal host-key checks and SSH configuration.

`job exec JOB` automatically enables stdin/TTY and opens an interactive login shell. Explicit TTY commands load configuration through the account login/interactive shell with safely quoted argv and do not send environment through terminal stdin. Non-TTY commands use `--env-mode user` by default: the backend probes exported login/interactive environment (10 seconds, 1 MiB maximum), discards startup output, and then executes argv. `--env-mode container` skips probing while retaining the task user and HOME. Commands keep the container working directory when accessible; shell sessions start in HOME. Shell aliases and functions require explicit shell execution.

Exec starts with container environment variables. Place variables that commands need in the container configuration or the task user’s shell startup files. Custom tasks bypassing platform initialization must prepare the task account in the image or during startup. Cross-user execution requires runuser, sudo or su.

WebIDE and Jupyter jobs use the same task user environment. Exec starts a new process; it does not attach to an existing IDE terminal or Notebook kernel, or infer the Python interpreter selected in the IDE. When the requested operation needs a specific environment, use its interpreter path or `conda run -n ENV COMMAND`, after confirming that environment with the user or job configuration.

SSH uses the platform login username by default. Use `--user` to select an existing container user. The container’s sshd and account shell control the login environment.

If the image permits SSH password authentication and you have neither a usable key nor a container password, enter the job with `crater job exec JOB`. Run `sudo passwd "$(id -un)"` in the container and enter the password at the terminal prompt. Root can run `passwd` directly. Exec works without a container password, while setting one requires root or usable sudo. This password belongs to the container user and is separate from the platform LDAP password. Connect with `crater job ssh JOB --connect` and specify `--user` if needed. Ask the user to enter the password in their terminal.
