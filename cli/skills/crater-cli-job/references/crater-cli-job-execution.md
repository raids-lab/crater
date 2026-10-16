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

`-t` requires `-i`, local terminal input/output, and interactive mode. Interactive exec and SSH connection cannot use `--json` or `--no-interactive`. TTY output combines stderr and stdout. `job ssh` without `--connect` continues to open SSH access and print the address; `--json` returns `data.ssh` instead of starting a session.

Exec requires backend API v4. It uses the platform's Kubernetes connection and does not require local kubectl, a kubeconfig, sshd, bash, GPUs, or NVIDIA tooling. It can target CPU, ARM, and domestic accelerator nodes. The requested executable must exist in the image. Direct SSH uses the local OpenSSH client and the image's SSH service; retain normal host-key checks and SSH configuration.

SSH and exec share the task account's HOME and preferred shell; the user does not need to invoke `/crater-start.sh`. `job exec JOB` automatically enables stdin/TTY and opens an interactive login shell. Explicit TTY commands load configuration through the account login/interactive shell with safely quoted argv and do not send environment through terminal stdin. Non-TTY commands use `--env-mode user` by default: the backend probes exported login/interactive environment (10 seconds, 1 MiB maximum), discards startup output, and then executes argv. `--env-mode container` skips probing while retaining the task user and HOME. Commands keep the container working directory when accessible; shell sessions start in HOME. Shell aliases and functions require explicit shell execution.

Platform startup scripts save exported initialization state. Custom tasks bypassing those scripts must initialize the task account in the image or during startup. Existing jobs without that snapshot still load container variables and user configuration. Recreate a job when startup-only generated settings are required. A new session does not inherit temporary exports or Conda activation from another terminal. Do not edit rc files or rerun account provisioning during connection. Cross-user execution needs an image-provided runuser, sudo, or su; a missing task account must not fall back to root.

WebIDE and Jupyter jobs use the same task user environment. Exec starts a new process; it does not attach to an existing IDE terminal or Notebook kernel, or infer the Python interpreter selected in the IDE. When the requested operation needs a specific environment, use its interpreter path or `conda run -n ENV COMMAND`, after confirming that environment with the user or job configuration.

The account's shell controls startup files. Bash login startup does not itself guarantee `.bashrc` is sourced; the user's login profile determines that relationship. Zsh, Bash and POSIX sh retain their own initialization behavior. SSH defaults to the backend-confirmed task user, with the active login username as a fallback for older backends. The shared SSH launcher contains private task environment and is readable by the task account and root; a different `--user` must have access to it.
