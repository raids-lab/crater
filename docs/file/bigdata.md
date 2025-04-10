# 如何将大文件上传到 Crater 平台

::: tip
感谢 changc 的贡献！
:::

## `rsync` 安装

- linux、mac 直接装就行
- windows 不能直接用，可以在 cygwin、WSL 里使用

## 例子——windows 下 cygwin 下使用 rsync 向 crater 传输大文件

1. 开一个 jupyter lab，在详情页找到 ssh 连接，点击进入，了解 ssh 的 ip 及端口![image-20250410171329052](img/show-ssh.png)

2. 在本地 windows 下安装 cygwin，其中有个安装软件包的环节，记得安装 rsync 和 openssh

3. 在 cygwin 中，使用命令

   ```bash
   rsync -avzP -e "ssh -p 端口"  本地文件路径  用户名@ip:远程存放路径
   ```

   rsync 用法可以参考[rsync 用法详解：最全面的 rsync 使用指南 - 阿小信的博客](https://blog.axiaoxin.com/post/rsync-guide/)
