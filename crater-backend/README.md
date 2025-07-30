# ![Crater Backend](./docs/image/icon.png) Crater Backend
Crater is a Kubernetes-based GPU cluster management system providing a comprehensive solution for GPU resource orchestration.


## 🚀 Run Crater Backend

### Install Essential Software

It is recommended to install the following software in the suggested versions.

- **Go**: Version `v1.24.4` is recommended  
  📖 [Go Installation Guide](https://go.dev/doc/install)
- **Kubectl**: Version `v1.33` is recommended  
  📖 [Kubectl Installation Guide](https://kubernetes.io/docs/tasks/tools/)

Next, set the required environment variables.

```bash
# Linux/macOS

# Set GOROOT to your Go installation directory
export GOROOT=/usr/local/go  # Change this path to your actual Go installation location

# Add Go to PATH
export PATH=$PATH:$GOROOT/bin

# Set proxy for Go
export GOPROXY=https://goproxy.cn,direct
```

You may add this into your shell config such as `.zshrc`.

For Go proxy, you can also set with a single command running, instead of add a command into shell config.

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

### Prepare Config Files

To run the project, you also need to contact your administrator to obtain some configuration files.

```
kubeconfig
etc/debug-config.yaml
.debug.env
```

Crater backend cannot run properly if any of these three files are missing.

#### kubeconfig

`kubeconfig` is a configuration file used by Kubernetes clients and tools to access and manage your Kubernetes cluster. It contains cluster connection details, user credentials, and context information. Please obtain the correct `kubeconfig` file from your administrator to ensure proper access to the cluster.

#### debug-config.yaml

The `etc/debug-config.yaml` file contains the application configuration for the Crater backend service. This configuration file defines various settings including:

- **Service Configuration**: Server ports, metrics endpoints, and profiling settings
- **Database Connection**: PostgreSQL connection parameters and credentials
- **Workspace Settings**: Kubernetes namespace, storage PVCs, and ingress configurations
- **External Integrations**: ACT system authentication (no need for non-act environments), image registry, SMTP, and OpenAPI endpoints
- **Feature Flags**: Scheduler and job type enablement settings

You can find an example in `etc/example-config.yaml`. 

For adminstrator, you need to fill in relative values based on your deployment helm charts values, and provide it to your members.

#### .debug.env

The `.debug.env` file specifies the port numbers used by the services. If your team is developing on the same node, you need to coordinate to avoid port conflicts.

```env
CRATER_FE_PORT=xxxx  # Frontend
CRATER_BE_PORT=xxxx  # Backend
CRATER_MS_PORT=xxxx  # Microservice
CRATER_HP_PORT=xxxx  # Health Probe
CRATER_SS_TARGET="http://localhost:7320"
```

CRATER_SS_TARGET is the destination address for forwarding requests to the storage service. If your development does not involve the storage service, you can **skip** setting this environment variable.

### Run Crater Backend

After completing the above setup, you can run the project using the `make` command. If you don't have `make` installed yet, it is recommended to install it.

```bash
make run
```

If the server is running and accessible at your configured port, you can open the Swagger UI to verify:

```bash
http://localhost:<your-backend-port>/swagger/index.html#/
```

![Swagger UI](./docs/image/swag.png)


## 💻 Development Guide

If you want to develop and contribute your code, you need to perform some additional steps on top of the above.

### Development Tools

The project uses swag to generate API documentation and golangci-lint for code style checking. Some hooks are also configured, but you don't need to install them manually. You can simply use the `make` command to install all required tools and hooks with one command.

```bash
make setup
```

You don't have to run this command manually; the required tools will be installed into the `bin` directory by `make` when needed. 

However, please note that `make` does not check the versions of tools you have already installed. If the project updates the versions of these tools, you need to delete your local copies and let `make` reinstall them for you.

### 🛠️ Database Code Generation (If Needed)
The project uses GORM Gen to generate boilerplate code for database CRUD operations.

Generation scripts and documentation can be found in: [`gorm_gen`](./cmd/gorm-gen/README.md)

Please regenerate the code after modifying database models or schema definitions, while CI pipeline will automatically make database migrations.

### Manual Check before commit (If Needed)

If you have completed the previous steps, some checks will be automatically performed by `git` before you commit. If these checks fail, your commit will be rejected. You can also manually trigger these checks with the following command.

```bash
make pre-commit-check
```

### 🐞 Debugging with VSCode (If Needed)

You can start the backend in debug mode using VSCode by pressing F5 (Start Debugging). You can set breakpoints and step through the code interactively.

Example launch configuration:

```json
 {
            "name": "Debug Server",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/main.go",
            "env": {
                "KUBECONFIG": "${workspaceFolder}/kubeconfig",
                "CRATER_DEBUG_CONFIG_PATH": "${workspaceFolder}/etc/example-config.yaml",
            }
        }
```
