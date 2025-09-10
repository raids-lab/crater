#  Storage server

Chinese Version : [README_CN.md](README_CN.md)

Crater is a Kubernetes-based GPU cluster management system providing a comprehensive solution for GPU resource orchestration.



## 💻 Development Guide

Before getting started with development, please ensure your environment has the following tools installed:

- **Go**: Version `v1.22.1` is recommended  
  📖 [Go Installation Guide](https://go.dev/doc/install)

- **Kubectl**: Version `v1.22.1` is recommended  
  📖 [Kubectl Installation Guide](https://kubernetes.io/docs/tasks/tools/)


### 📐 Code Style & Linting

This project uses [`golangci-lint`](https://golangci-lint.run/) to enforce Go code conventions and best practices. To avoid running it manually, we recommend setting up a Git pre-commit hook to automatically check the code before each commit.

After installation, you might need to add your GOPATH to the system PATH so that golangci-lint can be used in the terminal. For example, on Linux:

```bash
# Check your GOPATH
go env GOPATH
# /Users/your-username/go

# Add the path to .bashrc or .zshrc
export PATH="/Users/your-username/go/bin:$PATH"

# Reload the shell and verify
golangci-lint --version
# golangci-lint has version 1.61.0
```
#### Setting Up Git Pre-Commit Hook

Copy the `.githook/pre-commit` script to your Git hooks directory and make it executable:

**Linux/macOS:**
```bash
cp .githook/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```
Windows:

* Copy the script to .git/hooks/pre-commit

* Modify the script to replace golangci-lint with golangci-lint.exe if needed, or adapt it into a .bat file.

With the hook in place, golangci-lint will automatically run on staged files before each commit.



#### 🛠️ Database Code Generation
The project uses GORM Gen to generate boilerplate code for database CRUD operations.

Generation scripts and documentation can be found in: [`gorm_gen`](./cmd/gorm-gen/README.md)

Please regenerate the code after modifying database models or schema definitions, while CI pipeline will automatically make database migrations.

###  Project Configuration
Install dependencies and plugins:
```bash
go mod download
```
Sure! Here's the **English version** of the **"Running the Code"** section for your `README.md`, divided into two parts: **Local Development** and **Kubernetes Deployment**, with a recommendation for the latter.

---

## 🚀 Running the Code

This project supports two ways to run: **Local Development** and **Deployment on a Kubernetes Cluster**. We **recommend using the Kubernetes deployment** for full functionality and behavior closer to production.

---

### 🧑‍💻 Local Development

> Suitable for quick testing and development phases.


#### 📄 Configuration:

Make sure you have a [config.yaml](./etc/config.yaml) file with the correct database settings. 

Create a `.env` file at the root directory to customize local ports. This file is ignored by Git:

```env
PORT=xxxx
ROOTDIR="/crater"
```

#### 📁 Directory Setup:

**Create a folder named `crater` in the project root directory to simulate file handling behavior.**

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

### 🚀 Quik Deployment
To deploy Crater Project in a production environment, we provide a Helm Chart available at: [Crater Helm Chart](https://github.com/raids-lab/crater).

Please refer to the main documentation for detailed deployment instructions.
