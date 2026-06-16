[English](README.md) | [简体中文](README.zh-CN.md)

# Crater Backend

Crater is a Kubernetes-based heterogeneous cluster management system that supports various heterogeneous hardware such as NVIDIA GPUs.

Crater Backend is a subsystem of Crater, including job submission, job lifecycle management, deep learning environment management, and other features.

<table>
  <tr>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/jupyter.gif"><br>
      <em>Jupyter Lab</em>
    </td>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/ray.gif"><br>
      <em>Ray Job</em>
    </td>
  </tr>
  <tr>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/monitor.gif"><br>
      <em>Monitor</em>
    </td>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/datasets.gif"><br>
      <em>Models</em>
    </td>
  </tr>
</table>

To install or use the complete Crater project, visit the [Crater Official Documentation](https://raids-lab.github.io/crater/en/docs/admin/).

## Development

The development specification — local setup, `make` targets, API/routing/error conventions, the GORM database workflow, the storage server, and VSCode debugging — lives in [CONTRIBUTING.md](./CONTRIBUTING.md). For local startup topology, see the Local Run and Storage Server sections there.

Repository-wide rules (global development rules, hooks, unified config, commit/PR conventions) live in the root [CONTRIBUTING.md](../CONTRIBUTING.md).
