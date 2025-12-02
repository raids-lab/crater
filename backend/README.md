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

This document is the development guide for Crater Backend. If you want to install or use the complete Crater project, you can visit the [Crater Official Documentation](https://raids-lab.github.io/crater/en/docs/admin/) for more information.

## 🚀 Running Crater Backend Locally

### Installing Required Software

The following software and recommended versions are suggested.

- **gvm**: Optional, recommended version `v1.0.22`: [gvm - GitHub](https://github.com/moovweb/gvm)
- **Kubectl**: Required, recommended version `v1.33`: [Kubectl Installation Guide](https://kubernetes.io/docs/tasks/tools/)

gvm is used to easily and quickly install multiple Go versions and switch between them flexibly. Using gvm allows you to quickly install the Go version used by Crater and switch quickly when upgrading Go versions.

You can install gvm using the following command:

```bash
# Linux/macOS
bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
```

After gvm is successfully installed, you can quickly install the corresponding Go version in the backend directory (i.e., the directory where `go.mod` is located) using the following command:

```bash
# Linux/macOS
gvm applymod
```

Of course, you can also install Go directly without using gvm.

- **Go**: Recommended version `v1.25.4`: [Go Installation Guide](https://go.dev/doc/install)

In this case, you may also need to set environment variables to ensure that programs installed via `go install` can run directly.

```bash
# Linux/macOS

# Set GOROOT to your Go installation directory
export GOROOT=/usr/local/go  # Change this path to your actual Go installation location

# Add Go to PATH
export PATH=$PATH:$GOROOT/bin
```

You can add these contents to your shell configuration file, such as `.zshrc`.

Regardless of how you install Go, you may also need to configure the Go proxy, which can be set by running a single command without adding it to the shell configuration.

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

### Preparing Configuration Files

#### `kubeconfig`

To run the project, you need at least one Kubernetes cluster and have Kubectl installed.

For testing or learning environments, you can quickly obtain a cluster through open-source projects such as Kind, MiniKube, etc.

`kubeconfig` is a configuration file used by Kubernetes clients and tools to access and manage Kubernetes clusters. It contains cluster connection details, user credentials, and context information.

Crater Backend will first try to read the `kubeconfig` corresponding to the `KUBECONFIG` environment variable. If it doesn't exist, it will read the `kubeconfig` file in the current directory.

```makefile
# Makefile
KUBECONFIG_PATH := $(if $(KUBECONFIG),$(KUBECONFIG),${PWD}/kubeconfig)
```

#### `./etc/debug-config.yaml`

The `etc/debug-config.yaml` file contains the application configuration for the Crater backend service. This configuration file defines various settings, including:

- **Service Configuration**: Server port, metrics endpoints, and profiling settings
- **Database Connection**: PostgreSQL connection parameters and credentials
- **Workspace Settings**: Kubernetes namespaces, storage PVCs, and ingress configuration
- **External Integrations**: Raids Lab system authentication (not required for non-Raids Lab environments), image registry, SMTP email notification service, etc.
- **Feature Flags**: Scheduler and job type enablement settings昂

You can find example files and corresponding descriptions in [`etc/example-config.yaml`](https://github.com/raids-lab/crater-backend/blob/main/etc/example-config.yaml).

#### `.debug.env`

When you run the `make run` command, we will help you create a `.debug.env` file, which will be ignored by git and can store personalized configuration.

Currently, it only contains one configuration to specify the port number used by the service. If your team is developing on the same node, you can coordinate through it to avoid port conflicts.

```env
CRATER_BE_PORT=:8088  # Backend port
```

In development mode, we proxy the service through Crater Frontend's Vite Server, so you don't need to worry about CORS and other issues.

### Running Crater Backend

After completing the above setup, you can use the `make` command to run the project. If `make` is not yet installed, it is recommended to install it.

```bash
make run
```

If the server is running and accessible on your configured port, you can open Swagger UI for verification:

```bash
http://localhost:<your backend port>/swagger/index.html#/
```

![Swagger UI](./docs/image/swag.png)

You can run the `make help` command to view the complete list of related commands:

```bash
➜  crater-backend git:(main) ✗ make help 

Usage:
  make <target>

General
  help                Display this help.
  show-kubeconfig     Display current KUBECONFIG path
  prepare             Prepare development environment with updated configs

Development
  vet                 Run go vet.
  imports             Run goimports on all go files.
  import-check        Check if goimports is needed.
  lint                Lint go files.
  curd                Generate Gorm CURD code.
  migrate             Migrate database.
  docs                Generate docs docs.
  run                 Run a controller from your host.
  pre-commit-check    Run pre-commit hook manually.

Build
  build               Build manager binary.
  build-migrate       Build migration binary.

Development Tools
  golangci-lint       Install golangci-lint
  goimports           Install goimports
  swaggo              Install swaggo

Git Hooks
  pre-commit          Install git pre-commit hook.
```

## 🛠️ Database Code Generation (If Needed)

The project uses GORM Gen to generate boilerplate code for database CRUD operations. Go Migrate is used to generate database tables for objects.

Generation scripts and documentation can be found at: [`gorm_gen`](./cmd/gorm-gen/README.md)

After modifying database models or schema definitions, please regenerate the code.

If you installed Crater via Helm, database migration will be performed automatically after deploying a new version. The related logic can be found in InitContainer.

## 🐞 Debugging with VSCode (If Needed)

You can use VSCode to start the backend in debug mode by pressing F5 (Start Debugging). You can set breakpoints and interactively step through the code.

### Quick Start

The project has provided a pre-configured debug launch configuration in the root directory `.vscode/launch.json`. You only need to:

1. Open the project root directory in VSCode (`crater`, the root directory containing `backend` and `frontend`)
2. Set breakpoints (click on the left side of the line number)
3. Press `F5` to start debugging and select the "Backend Debug Server" configuration

> This debug configuration was migrated from the original backend repository (`backend/.vscode/launch.json`) to the project root directory. If you need to use the original debug configuration, you can directly open the `backend` directory in VSCode and use the configuration in `backend/.vscode/launch.json`.

### Debug Configuration Explanation

The `.vscode/launch.json` in the project root directory contains the following configuration:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Backend Debug Server",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/backend/cmd/crater/main.go",
            "cwd": "${workspaceFolder}/backend",
            "env": {
                "KUBECONFIG": "${workspaceFolder}/backend/kubeconfig",
                "NO_PROXY": "k8s.cluster.master"
            }
        }
    ]
}
```

Where:

- **`cwd`**: Set to `${workspaceFolder}/backend`, which ensures the program can correctly find configuration files with relative paths (such as `./etc/debug-config.yaml`)
- **`program`**: Main program entry file, pointing to `backend/cmd/crater/main.go`
- **Automatic Configuration File Discovery**: The program automatically searches for `./etc/debug-config.yaml` in debug mode (relative to `cwd`), **no need** to pass `--config-file` parameter through `args`
- **`KUBECONFIG`**: Uses the `kubeconfig` configuration file in the backend repository to connect to the cluster
