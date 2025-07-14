---
title: 使用 SSH 功能快速连接

description: 为了帮助用户更便捷的连接到容器，本平台提供了 SSH 功能 。在配置 SSH 免密登陆后，用户可以一键复制连接命令，通过 Terminal 或 VSCode 连接至容器。
---

## 配置 SSH 免密登录

可使用 `authorized_keys` 配置免密登录，上传公钥（通常是 `id_rsa.pub`）到服务器（生成过程可参考 [VSCode 连接到 Jupyter 容器内](./vscode-ssh.md) 的 “确保本机已生成公私钥” 章节）。

- 若 `.ssh` 文件夹不存在，可执行以下命令创建 `.ssh` 文件夹，并设置适当的权限：

```bash
mkdir ~/.ssh
chmod 700 ~/.ssh
```

- 将本机公钥添加到 `~/.ssh/authorized_keys` 文件

```bash
# 将本机 id_rsa.pub 文件内容复制到 ~/.ssh/authorized_keys
vim ~/.ssh/authorized_keys
# 为 authorized_keys 设置适当权限
chmod 600 ~/.ssh/authorized_keys
```

该过程 **只需首次连接时进行** ，配置完成后再次连接无需额外操作。

## 使用包含 SSHD 的镜像创建任务

目前平台提供的官方镜像内**已包含 SSHD**，无需额外安装 🚀。

如您需要使用自定义镜像，则需保证您构建的镜像内已包含 SSHD 。

## 一键复制连接命令

进入**作业详情页**，可以在页面右上角点击 **“SSH 连接”** 按钮。

![](./img/ssh-func/ssh-detail.webp)

点击后会弹出如下对话框:

![](./img/ssh-func/ssh-func.webp)

可以根据需要复制 Terminal 和 VSCode 的连接命令。

## Terminal 连接

在 Terminal 中输入复制的命令，即可连接到容器内。

![](./img/ssh-func/terminal.webp)

## VSCode 连接

（1）VSCode 中要安装 Remote-SSH 扩展，见下：

![](./img/ssh-func/remote-ssh.webp)

（2）点击 VSCode 左下角的远程连接图标，在弹出的菜单中选择 **“Remote - SSH: Connect to Host”** 。

（3）如果是第一次连接，VSCode 会提示您选择操作系统类型，选择对应的操作系统（如 Linux）。

（4）等待 VSCode 在远程服务器上安装必要的组件（这一步可能需要等待较长的时间）。

![](./img/ssh-func/download-server.webp)

（5）安装完成后，VSCode 即可连接到容器内。

![](./img/ssh-func/connect.webp)
