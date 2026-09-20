# Crater CLI

Crater CLI is the command-line client for the Crater platform. The npm package installs the native binary for the current operating system and CPU architecture, while keeping the command name `crater`.

## Install

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

The package provides native binaries for Linux, macOS, and Windows on x64 and ARM64. Windows support is currently experimental and has not received the same level of validation as Linux and macOS. Please report problems through the [Crater issue tracker](https://github.com/raids-lab/crater/issues).

## License

Crater CLI is licensed under the Apache License 2.0.
