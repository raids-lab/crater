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

Crater CLI 是 Crater 的命令行客户端，通过 HTTP API 与 Crater 平台通信，面向终端用户和 AI Agent。

## 功能特性

- 面向 Agent 的 `--json` 与 `--no-interactive` 模式。
- 本地认证上下文管理。
- Bash 与 zsh 补全。
- 中文与英文显示语言支持。
- 面向 Linux、macOS 和 Windows 的 x64 与 ARM64 原生二进制。

## 安装

使用 npm 安装最新正式版本：

```bash
npm install --global @raids-lab/crater-cli
```

也可以使用 pnpm 安装，或通过 npx 临时运行：

```bash
pnpm add --global @raids-lab/crater-cli
npx @raids-lab/crater-cli --help
```

npm 入口包会根据当前操作系统和 CPU 架构选择对应的原生二进制，对外命令仍为 `crater`。

Windows 二进制目前属于实验性支持，其验证程度尚未达到 Linux 和 macOS 的水平。如遇问题，请通过 [GitHub Issues](https://github.com/raids-lab/crater/issues) 反馈。

## 使用

查看可用命令和选项：

```bash
crater -h
```

查看已安装二进制及其构建信息：

```bash
crater --version
crater version
```

日常使用时，安装随 CLI 提供的 Skills 后，可以直接让 AI Agent 操作 Crater。

## Agent Skills

列出本目录下可安装的 Skills：

```bash
npx skills add https://github.com/raids-lab/crater/tree/main/cli -l
```

为支持的 Agent 全局安装全部 Crater CLI Skills：

```bash
npx skills add https://github.com/raids-lab/crater/tree/main/cli -g --all
```

## 许可证

Crater CLI 使用 Apache License 2.0 许可证，详见 [LICENSE](LICENSE)。
