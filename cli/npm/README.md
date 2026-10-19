# Crater CLI

Crater CLI is the command-line client for the [Crater platform](https://github.com/raids-lab/crater). The npm package installs the native binary for the current operating system and CPU architecture, while keeping the command name `crater`.

## Install

To get started, share [this npm package page](https://www.npmjs.com/package/@raids-lab/crater-cli) with your AI Agent and ask it to install Crater CLI and the accompanying CLI Skills for you. You can also follow the steps below to install them yourself.

With npm:

```bash
npm install --global @raids-lab/crater-cli
```

With pnpm:

```bash
pnpm add --global @raids-lab/crater-cli
```

Run a single command without a global installation:

```bash
npx @raids-lab/crater-cli --help
pnpm dlx @raids-lab/crater-cli --help
```

The package provides native binaries for Linux, macOS, and Windows on x64 and ARM64.

**Windows support is currently experimental and may be more limited than support for Linux and macOS.** It has not received the same level of validation. If you encounter any problems, please contact us by opening an issue on [Crater GitHub Issues](https://github.com/raids-lab/crater/issues).

## Agent Skills

**We strongly recommend installing the [Crater CLI Skills](https://github.com/raids-lab/crater/tree/main/cli/skills) alongside Crater CLI.** These Skills guide your AI Agent through CLI operations so it can help you use Crater. They are installed separately from the npm package.

List the available Crater CLI Skills:

```bash
npx skills add https://github.com/raids-lab/crater/tree/main/cli -l
```

Install all Crater CLI Skills globally for all supported agents without confirmation prompts:

```bash
npx skills add https://github.com/raids-lab/crater/tree/main/cli -g --all
```

## License

Crater CLI is licensed under the Apache License 2.0.
