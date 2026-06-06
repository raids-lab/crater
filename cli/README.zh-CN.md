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

## 使用

查看可用命令和选项：

```bash
crater -h
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
