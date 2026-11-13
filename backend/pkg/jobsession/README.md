# Job user sessions

`jobsession` prepares the task account and shell environment for Kubernetes exec. Its POSIX scripts run inside Linux job containers.

- The container’s `NB_USER` selects the task account, with the current container account as the default. The account database supplies HOME and the preferred shell. A missing account or failed user switch stops execution.
- `bootstrap.sh` captures container exports before switching accounts with runuser, sudo or su. The unprivileged stage restores those values so PATH and loader variables survive the switch. The capture and shell probe skip Bash’s read-only `SHELLOPTS` and `BASHOPTS`.
- An interactive session starts that account's shell with `-il` in HOME. Each shell controls its startup files; Bash login profiles decide whether to source `.bashrc`.
- Explicit TTY commands run through the account login/interactive shell using safely quoted argv. They send no private environment prefix through the PTY, whose echo/canonical settings could otherwise disclose or alter environment data. Shell startup output remains part of that interactive terminal.
- A non-TTY explicit command defaults to probing the login/interactive shell's exported environment, with a 10-second deadline and a 1 MiB output limit. Markers separate startup output from NUL-delimited environment values. Export names and identity variables are filtered; values are quoted and argument boundaries are preserved. Startup output is discarded. Prepared environment values travel as a byte-counted private stdin prefix, never as Kubernetes URL parameters. The wrapper consumes exactly that prefix, verifies its length, and leaves command stdin untouched. `env_mode=container` skips this probe, retaining account identity and HOME. Neither mode recreates aliases, functions, or another terminal's runtime state.
- Commands keep the container working directory if accessible, otherwise HOME. User and task environment preparation happens before the requested argv runs; no user-provided argument is interpolated as shell syntax.

Custom tasks which bypass the platform startup scripts must prepare NB_USER in the image or during job startup; connection never creates an absent account.


The image needs `/bin/sh`, `env -0`, POSIX utilities including dd/wc, and getent or `/etc/passwd` plus awk. Cross-user execution additionally needs runuser, sudo, or su. No GPU tooling or particular accelerator environment variable is required.

Tests exercise the actual scripts with isolated account, HOME and shell fixtures, argv boundaries, multiline/quoted environment values, Ascend/loader variables, configuration output filtering, probe failure/cancellation, account failures, exit status, and Bash option filtering. They do not create or change host users.
