[English](README.md) | [简体中文](README.zh-CN.md)

# Crater

这是一个使用 [Create Fumadocs](https://github.com/fuma-nama/fumadocs) 生成的 Next.js 应用程序。

## 快速开始

运行开发服务器：

```bash
pnpm config set registry https://registry.npmmirror.com
pnpm install
pnpm dev
```

在浏览器中打开 http://localhost:3000/crater/zh 查看结果。

## 构建并在本地运行

```bash
pnpm build
pnpm dlx serve@latest out --serve-path /crater
```

在浏览器中打开 http://localhost:3000/crater/zh 查看结果。

## 提交前准备

```bash
./hack/squoosh_images.py
```

## 了解更多

要了解更多关于 Next.js 和 Fumadocs 的信息，请查看以下资源：

- [Next.js 文档](https://nextjs.org/docs) - 了解 Next.js 的功能和 API
- [学习 Next.js](https://nextjs.org/learn) - 交互式 Next.js 教程
- [Fumadocs](https://fumadocs.vercel.app) - 了解 Fumadocs

