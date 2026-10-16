# Job user sessions

`jobsession` owns user environment handling shared by Kubernetes exec and the CLI's SSH connection. Its embedded POSIX scripts run in Linux containers on CPU or accelerator nodes without downloading architecture-specific helpers.

- The container's `NB_USER` selects the task account; without it, the current container account is used. The account database supplies HOME and the preferred shell. An absent account or failed user switch stops execution.
- `bootstrap.sh` reads the root-managed startup snapshot, captures the container environment, and switches to the account using runuser, sudo, or su when necessary. The unprivileged stage restores exported variables after switching so PATH and loader/toolchain variables survive privilege helpers.
- An interactive session starts that account's shell with `-il` in HOME. Each shell controls its startup files; Bash login profiles decide whether to source `.bashrc`.
- Explicit TTY commands run through the account login/interactive shell using safely quoted argv. They send no private environment prefix through the PTY, whose echo/canonical settings could otherwise disclose or alter environment data. Shell startup output remains part of that interactive terminal.
- A non-TTY explicit command defaults to probing the login/interactive shell's exported environment, with a 10-second deadline and a 1 MiB output limit. Markers separate startup output from NUL-delimited environment values. Export names and identity variables are filtered; values are quoted and argument boundaries are preserved. Startup output is discarded. Prepared environment values travel as a byte-counted private stdin prefix, never as Kubernetes URL parameters. The wrapper consumes exactly that prefix, verifies its length, and leaves command stdin untouched. `env_mode=container` skips this probe, retaining account identity and HOME. Neither mode recreates aliases, functions, or another terminal's runtime state.
- Commands keep the container working directory if accessible, otherwise HOME. User and task environment preparation happens before the requested argv runs; no user-provided argument is interpolated as shell syntax.

The SSH API atomically installs a private `/run/crater/session.sh`, returns the resolved username and a fixed `session_command`, and the CLI invokes that command with OpenSSH and a TTY. This entry point uses the same `bootstrap.sh` and `user.sh`. The launcher contains task environment, so only the task account and root may read it. Its parent directory is root-owned. Normal direct SSH clients that do not request this entry point continue to use sshd's own session behavior.

Custom tasks which bypass the platform startup scripts must prepare NB_USER in the image or during job startup; connection never creates an absent account.

The Helm startup scripts save exported initialization state in root-only `/run/crater/environment.sh`. They preserve an existing account's HOME and shell. Connecting never reruns account provisioning, chmod/chown, or sudoers setup. Jobs created before this snapshot was introduced use their container environment and shell configuration; startup-only generated values require recreating the job with the updated chart.

The image needs `/bin/sh`, `env -0`, POSIX utilities including dd/wc, and getent or `/etc/passwd` plus awk. Cross-user execution additionally needs runuser, sudo, or su. SSH setup needs a writable `/run/crater` and the image's configured OpenSSH service. No GPU tooling or particular accelerator environment variable is required.

Tests exercise the actual scripts with isolated account, HOME and shell fixtures, argv boundaries, multiline/quoted environment values, Ascend/loader variables, configuration output filtering, probe failure/cancellation, account failures, exit status, and matching SSH/exec environments. They do not create or change host users.
