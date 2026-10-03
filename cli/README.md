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
