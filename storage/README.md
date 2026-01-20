[English](README.md) | [简体中文](README_CN.md)

# Storage server

Crater is a Kubernetes-based GPU cluster management system providing a comprehensive solution for GPU resource orchestration.



## 💻 Development Guide

Before getting started with development, please ensure your environment has the following tools installed:

- **Go**: Version `v1.25.4` is recommended  
  📖 [Go Installation Guide](https://go.dev/doc/install)

- **Kubectl**: Version `v1.33` is recommended  
  📖 [Kubectl Installation Guide](https://kubernetes.io/docs/tasks/tools/)

For specific installation methods, please refer to the backend repository [README](../backend/README.md).


### 📐 Code Style & Linting

This project uses [`golangci-lint`](https://golangci-lint.run/) to enforce Go code conventions and best practices. To avoid running it manually, we have added a `pre-commit-check` target in the Makefile to perform related operations. This target will be used by the Git pre-commit hook in the Crater main repository. To install this hook, please refer to the Crater main repository [README](../README.md).

The `pre-commit-check` target in this repository prioritizes using golangci-lint installed in the backend repository. If unavailable, it will try to use your locally installed golangci-lint. You can install it locally (**not recommended**) on Linux:

```bash
# Check your GOPATH
go env GOPATH
# /Users/your-username/go

# Add the path to .bashrc or .zshrc
export PATH="/Users/your-username/go/bin:$PATH"

# The binary will be installed to $(go env GOPATH)/bin/golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.6.2

# Reload the shell and verify
golangci-lint --version
# golangci-lint has version 2.6.2
```

### 🛠️ Database Development Guide

The project uses **GORM** as the ORM framework, manages database version migrations through **gormigrate**, and automatically generates type-safe CRUD code using **GORM Gen**.

#### Core Concepts

- **Model Definition** (`dao/model/*.go`): Defines Go structs corresponding to database tables
- **Migration Scripts** (`cmd/gorm-gen/models/migrate.go`): Records database structure change history, supporting versioned migrations and rollbacks
- **Query Code Generation** (`cmd/gorm-gen/curd/generate.go`): Automatically generates type-safe database operation code based on model definitions

#### Database Change Development Workflow

When you need to modify the database structure (such as adding fields, creating new tables, etc.), please follow this workflow:

1. **Modify Model Definition**: Modify the corresponding struct definition in `dao/model/*.go`, adding or modifying fields and their GORM tags.

2. **Write Migration Script**: Add a new migration item to the migration list in `cmd/gorm-gen/models/migrate.go`. The migration ID uses a timestamp format `YYYYMMDDHHmm` (year, month, day, hour, minute), for example `202512091200`, ensuring uniqueness and guaranteeing execution in chronological order. Each migration item needs to include `Migrate` (upgrade) and `Rollback` (rollback) functions.

3. **Execute Database Migration**: Run the following command to apply changes to the database:
   ```bash
   go run cmd/gorm-gen/models/migrate.go
   ```
   This command checks executed migration records, executes all unexecuted migrations in order, and updates the migration record table. Migrations are idempotent, so repeated execution will not cause errors. If the database is brand new, the migration script will first execute `InitSchema` to create all tables and initialize default data (such as default accounts).

4. **Generate CRUD Code**: Run the following command to generate or update `dao/query/*.gen.go` files based on the latest model definitions:
   ```bash
   go run cmd/gorm-gen/curd/generate.go
   ```
   This command reads all model definitions and generates type-safe query methods.

5. **Write Business Code**: Use the generated `query` methods to operate on the database in business logic.

#### Common Commands

| Command | Description |
|---------|-------------|
| `go run cmd/gorm-gen/models/migrate.go` | Execute database migration, applying model changes to the database |
| `go run cmd/gorm-gen/curd/generate.go` | Generate database CRUD operation code |

#### Important Notes

1. **Migration Order**: Migrations are executed in ID order, ensure timestamps are incremental
2. **Team Collaboration**: After pulling code, if you see updates to `migrate.go`, remember to run the migration command
3. **Rollback with Caution**: Before rolling back in production, be sure to backup data
4. **First Deployment**: If the database is brand new, the migration script will automatically create all tables and initialize default data
5. **Database Connection**: Before executing migrations and code generation, please ensure the database connection configuration in `migrate.go` and `generate.go` is correct, or connect to the database in the cluster through port forwarding

#### Related Documentation

Generation scripts and detailed documentation can be found at: [`cmd/gorm-gen/README`](./cmd/gorm-gen/README)

### Project Configuration

Install dependencies and plugins:
```bash
go mod download
```

---

## 🚀 Running the Code

This project supports two ways to run: **Local Development** and **Deployment on a Kubernetes Cluster**. We **recommend using the Kubernetes deployment** for full functionality and behavior closer to production.

---

### 🧑‍💻 Local Development

> Suitable for quick testing and development phases.


#### 📄 Configuration:

**Note:** It is recommended to manage these configuration files (`config.yaml` and `.env`) through the unified configuration management system in the main repository. For details, please refer to the main repository README.

Make sure you have a [config.yaml](./etc/config.yaml) file with the correct database settings. 

Create a `.env` file at the root directory to customize local ports. This file is ignored by Git:

```env
PORT=xxxx
ROOTDIR="/crater"
```

#### 📁 Directory Setup:

**Create a folder named `crater` (or any other name) in a directory you are familiar with to simulate file handling behavior.**

**Alternatively, you can modify the `ROOTDIR` in the .env file and use it as the root directory for your testing.**

```bash
mkdir crater
```

This directory will act as the root for file processing.


#### 🚀 Run the Application:

```bash
make run
```

The service will start and listen on `localhost:port` by default.


---

### ☸️ Deploying to Kubernetes 

#### ✅ Prerequisites:

- Docker
- Access to a Kubernetes cluster (`kubectl`)
- A PVC named `crater-rw-storage` already created (for persistent file storage)

#### 📦 Build and Push the Docker Image:

```bash
docker build -t your-registry/crater-webdav:latest .
docker push your-registry/crater-webdav:latest
```

> Replace `your-registry` with your actual container registry.

#### 🚀 Deploy to Kubernetes:

Make sure the following files exist in your current directory:

- `Dockerfile`
- `deployment.yaml`
- `service.yaml` (if applicable)

You can find these files in https://github.com/raids-lab/crater/tree/main/charts/crater/templates/storage-server

Apply the manifests:

```bash
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

> Ensure that the `deployment.yaml` correctly references the image and mounts the PVC `crater-rw-storage`.

### 🚀 Quick Deployment

To deploy Crater Project in a production environment, we provide a Helm Chart available at: [Crater Helm Chart](https://github.com/raids-lab/crater).

Please refer to the main documentation for detailed deployment instructions.
