[English](README.md) | [简体中文](README.zh-CN.md)

# Crater CLI

```
 ██████╗██████╗  █████╗ ████████╗███████╗██████╗      ██████╗██╗     ██╗
██╔════╝██╔══██╗██╔══██╗╚══██╔══╝██╔════╝██╔══██╗    ██╔════╝██║     ██║
██║     ██████╔╝███████║   ██║   █████╗  ██████╔╝    ██║     ██║     ██║
██║     ██╔══██╗██╔══██║   ██║   ██╔══╝  ██╔══██╗    ██║     ██║     ██║
╚██████╗██║  ██║██║  ██║   ██║   ███████╗██║  ██║    ╚██████╗███████╗██║
 ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝     ╚═════╝╚══════╝╚═╝
```

Crater CLI is the command-line client for Crater. It talks to the Crater platform through HTTP APIs and is designed for both human terminal users and AI Agents.

## Features

- Agent-friendly `--json` and `--no-interactive` modes.
- Local authentication context management.
- Direct SSH connections and container command execution on CPU and accelerator nodes.
- Bash and zsh completion.
- Chinese and English display language support.
- Native binaries for Linux, macOS, and Windows on x64 and ARM64.

## Installation

Install the latest stable release with npm:

```bash
npm install --global @raids-lab/crater-cli
```

The same package can be installed with pnpm or run once with npx:

```bash
pnpm add --global @raids-lab/crater-cli
npx @raids-lab/crater-cli --help
```

The npm entry package selects the native binary for the current operating system and CPU architecture while keeping the command name `crater`.

Windows binaries are currently experimental and have not received the same level of validation as Linux and macOS. Please report problems through [GitHub Issues](https://github.com/raids-lab/crater/issues).

## Usage

View available commands and options:

```bash
crater -h
```

Inspect the installed binary and its build metadata:

```bash
crater --version
crater version
```

For day-to-day use, you can ask an AI Agent to operate Crater after installing the bundled Skills.

Connect to a running job, execute a command, or open a shell:

```bash
crater job ssh my-job --connect
crater job exec my-job -- uname -m
crater job exec my-job
crater job exec my-job --env-mode container -- env
crater job exec my-job --pod worker-0 -c worker --json -- python -V
```

`job exec` requires backend API v4 and uses the platform connection, without local kubectl, sshd, or accelerator-specific tools. With no command it opens the task user's login shell. Commands after `--` retain their argument boundaries, use that user's HOME, and inherit exported shell configuration. Use `--env-mode container` to skip configuration probing. Select the Pod/container explicitly when there are multiple candidates. `--json` captures output; remote failures preserve `context.exit_code`.

SSH and exec share user environment handling; `/crater-start.sh` is not required in the command. SSH requires the image's SSH service and local OpenSSH. `job ssh` prints the address; `--connect` uses the confirmed container user and opens its shared session automatically. Jobs using the platform startup scripts save their startup environment for later connections. Custom images that bypass initialization must prepare the task account in the image or during job startup. Existing jobs without this snapshot still use container variables and user configuration; recreate the job if startup-only generated variables are needed. New sessions do not copy another terminal's temporary exports or Conda activation.

## Agent Skills

List the Skills available in this directory:

```bash
npx skills add https://github.com/raids-lab/crater/tree/main/cli -l
```

Install all Crater CLI Skills globally for supported agents:

```bash
npx skills add https://github.com/raids-lab/crater/tree/main/cli -g --all
```

## License

Crater CLI is licensed under the Apache License 2.0. See [LICENSE](LICENSE).
