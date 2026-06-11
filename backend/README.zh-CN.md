[English](README.md) | [简体中文](README.zh-CN.md)

# Crater Backend

Crater 是一个基于 Kubernetes 的异构集群管理系统，支持英伟达 GPU 等多种异构硬件。

Crater Backend 是 Crater 的子系统，包含作业提交、作业生命周期管理、深度学习环境管理等功能。

<table>
  <tr>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/jupyter.gif"><br>
      <em>Jupyter Lab</em>
    </td>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/ray.gif"><br>
      <em>Ray 任务</em>
    </td>
  </tr>
  <tr>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/monitor.gif"><br>
      <em>监控</em>
    </td>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/datasets.gif"><br>
      <em>模型</em>
    </td>
  </tr>
</table>

如果您希望安装或使用完整的 Crater 项目，可访问 [Crater 官方文档](https://raids-lab.github.io/crater/en/docs/admin/) 了解更多。

## 开发

开发规范——本地运行、`make` target、API / 路由 / 错误约定、GORM 数据库工作流、Storage Server 与 VSCode 调试——见 [CONTRIBUTING.zh-CN.md](./CONTRIBUTING.zh-CN.md)。

仓库级通用规范（全局开发守则、hooks、统一配置、Commit / PR 约定）见仓库根 [CONTRIBUTING](../docs/zh-CN/CONTRIBUTING.md)。
