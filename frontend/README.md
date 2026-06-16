[English](README.md) | [简体中文](README.zh-CN.md)

# 🌋 Crater Frontend

Crater is a Kubernetes-based GPU cluster management system. This repository is its frontend: a unified web console for resource orchestration, job management, monitoring, and model/dataset management.

<table>
  <tr>
    <td align="center" width="45%">
      <img src="./docs/images/jupyter.gif"><br>
      <em>Jupyter Lab</em>
    </td>
    <td align="center" width="45%">
      <img src="./docs/images/ray.gif"><br>
      <em>Ray Job</em>
    </td>
  </tr>
  <tr>
    <td align="center" width="45%">
      <img src="./docs/images/monitor.gif"><br>
      <em>Monitor</em>
    </td>
    <td align="center" width="45%">
      <img src="./docs/images/datasets.gif"><br>
      <em>Models</em>
    </td>
  </tr>
</table>

Built on a modern React stack: TypeScript, React 19, TanStack Router, TanStack Query v5, Jotai, Tailwind CSS, and shadcn/ui.

## Development

The development specification — environment, `make run`, project structure, MSW mocking, local backend/storage proxy expectations, and the component / hooks / i18n / UX conventions — lives in [CONTRIBUTING.md](./CONTRIBUTING.md).

Repository-wide rules (global development rules, hooks, unified config, commit/PR conventions) live in the root [CONTRIBUTING.md](../CONTRIBUTING.md).

## Deployment

Deploy the full Crater system via Helm. See the [main documentation](https://raids-lab.github.io/crater/en/docs/admin/).
