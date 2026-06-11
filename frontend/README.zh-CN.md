[English](README.md) | [简体中文](README.zh-CN.md)

# 🌋 Crater Frontend

Crater 是一个基于 Kubernetes 的 GPU 集群管理系统，本仓库为其前端：提供算力编排、作业管理、监控可视化、模型与数据集管理等一体化 Web 控制台。

<table>
  <tr>
    <td align="center" width="45%">
      <img src="./docs/images/jupyter.gif"><br>
      <em>Jupyter Lab</em>
    </td>
    <td align="center" width="45%">
      <img src="./docs/images/ray.gif"><br>
      <em>Ray 任务</em>
    </td>
  </tr>
  <tr>
    <td align="center" width="45%">
      <img src="./docs/images/monitor.gif"><br>
      <em>监控</em>
    </td>
    <td align="center" width="45%">
      <img src="./docs/images/datasets.gif"><br>
      <em>模型</em>
    </td>
  </tr>
</table>

基于现代 React 技术栈构建：TypeScript、React 19、TanStack Router、TanStack Query v5、Jotai、Tailwind CSS、shadcn/ui。

## 开发

开发规范——环境、`make run`、目录结构、MSW Mock，以及组件 / hooks / i18n / 体验约定——见 [CONTRIBUTING.zh-CN.md](./CONTRIBUTING.zh-CN.md)。

仓库级通用规范（全局开发守则、hooks、统一配置、Commit / PR 约定）见仓库根 [CONTRIBUTING](../docs/zh-CN/CONTRIBUTING.md)。

## 部署

通过 Helm 部署完整 Crater 系统，详见[主项目文档](https://raids-lab.github.io/crater/zh/docs/admin/)。
