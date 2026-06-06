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

## Usage

View available commands and options:

```bash
crater -h
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
