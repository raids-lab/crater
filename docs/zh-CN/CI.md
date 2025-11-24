[English](../en/CI.md) | [简体中文](CI.md)

# CI 持续集成文档

本文档介绍 Crater 项目持续集成（CI）的设计与实现。

在开源至 GitHub 之前，本项目托管在实验室内部部署的 GitLab 中，使用 GitLab Pipeline 进行 CI/CD。迁移至 GitHub 后，我们取消了持续部署（CD）部分，仅保留持续集成（CI），并使用 GitHub Actions workflow 实现。

---

## 概述

Crater 项目的 CI 流程基于 GitHub Actions 构建，主要服务于代码质量保障和构建产物发布。与传统的 CI/CD 流程不同，我们仅保留了持续集成（CI）部分，将持续部署（CD）交由用户根据自身环境自行处理。这样的设计既保证了代码质量和构建产物的标准化，又给予了用户部署的灵活性。

CI 流程的输入主要包括仓库中的源代码、Dockerfile、文档文件和 Helm Chart 配置等文件；输出为构建产物，包括多平台 Docker 镜像、静态网站和 Helm Chart 包。所有 Docker 镜像和 Helm Chart 都保存在 GitHub Container Registry (GHCR) 中，文档网站部署在 GitHub Pages 上，用户可以通过相应的地址访问这些产物。

### 目标

Crater 的 CI 流程旨在通过自动化手段保障代码质量、标准化构建产物，并为用户提供可直接使用的镜像和 Chart。通过 PR 检查确保代码质量，通过自动化构建和发布减少人工干预，通过多平台支持满足不同环境需求，通过版本管理和清理策略保证产物的可追溯性和存储效率。

### 技术栈

Crater 的 CI 流程基于 GitHub Actions、Docker Buildx 和 GitHub Container Registry (GHCR) 构建。GitHub Actions 提供了与仓库深度集成的 CI 能力，无需额外配置第三方服务；Docker Buildx 通过 QEMU 模拟实现跨平台构建，能够同时构建 amd64 和 arm64 架构的镜像，满足不同硬件环境的需求；GHCR 作为容器镜像和 Helm Chart 的统一存储，与 GitHub 权限体系集成，支持 OCI 标准，并通过 `GITHUB_TOKEN` 实现自动化认证。

### CI 流程分类

Crater 的 CI 流程根据构建目标的不同，划分为四个主要类别，每个类别都有独立的触发条件和构建流程：

- **前端与后端** 是 CI 流程的核心，负责应用服务（Backend、Frontend、Storage）的代码质量检查和镜像构建发布。采用两阶段设计：PR 检查阶段进行代码风格检查（Lint）和构建验证，确保代码质量；构建发布阶段在代码合并后构建多平台镜像并推送到 GHCR，同时通过自动清理策略管理存储空间。

- **依赖镜像** 负责构建和推送构建工具相关的 Docker 镜像（buildx-client、envd-client、nerdctl-client），为应用构建提供必要的运行时环境。这些镜像同样支持多平台构建，并通过 GHCR 统一管理。

- **文档网站** 处理文档的构建、质量检查和自动化部署。PR 检查阶段验证文档构建成功并检查图片格式规范；部署阶段自动构建 Next.js 网站并部署到 GitHub Pages；同时通过自动修正和自动翻译机制，确保文档质量和多语言同步。

- **Helm Chart** 负责 Chart 的验证和发布。PR 检查阶段验证 Chart 语法、模板和版本号更新；发布阶段将 Chart 打包并推送到 GHCR OCI 仓库，为用户提供标准化的部署方案。

---

## 前端与后端

本章节介绍 Crater 前端与后端的 CI 配置。

需要特别说明的是，存储服务（storage-server）位于主仓库下的 `storage` 目录，其 CI 配置也包含在本章节中。存储服务采用与后端相同的 CI 模式，未来计划将其合并至后端。

### 概述

前端与后端的 CI 流程采用两阶段设计：PR 检查阶段和构建发布阶段。输入为源代码（Go 代码或前端资源），输出为多平台 Docker 镜像（linux/amd64 和 linux/arm64），产物保存在 GHCR 的 `ghcr.io/raids-lab/crater-backend`、`ghcr.io/raids-lab/crater-frontend` 和 `ghcr.io/raids-lab/storage-server` 仓库中。

PR 检查阶段在代码合并前执行，进行代码风格检查（Lint）和构建验证，只构建单平台以节省时间，不构建和推送镜像。构建发布阶段在代码合并后或创建版本标签时执行，构建多平台镜像并推送到 GHCR，同时自动清理旧镜像以控制存储空间。

Backend、Frontend 和 Storage 三个组件采用相同的两阶段 CI 模式，但构建过程不同：Backend 和 Storage 编译生成二进制文件后打包到镜像中，Frontend 构建静态资源后通过 Web 服务器提供服务的镜像。

后续章节主要介绍构建发布阶段的详细流程和机制，PR 检查阶段将在最后简要说明。

### 触发与版本管理

构建发布阶段有两种触发方式：代码推送到 main 分支或创建版本标签。以 Backend 的 workflow 配置为例：

```yaml
on:
  push:
    branches: [main]
    paths:
      - "backend/**"
      - ".github/workflows/backend-build.yml"
  tags:
    - "v*.*.*"
```

在 `push` 事件中，`branches: [main]` 指定只监听 main 分支的推送，`paths` 参数进一步过滤路径，只有当 `backend/**` 目录下的文件或 workflow 文件本身发生变更时才会触发构建。也就是说，如果某次提交仅仅修改了前端的代码，而没有修改后端的代码，那么只有前端的镜像会被重新构建。

`tags` 事件配置为 `v*.*.*`，匹配所有符合语义化版本格式的标签（如 v1.2.3）。标签触发不使用路径过滤，无论路径如何都会触发所有组件的构建。这是因为版本发布需要确保所有组件都基于相同的代码版本构建，保证版本的一致性和完整性。即使某个组件在本次发布中没有代码变更，也会重新构建并打上对应的版本标签。

### 版本号注入

构建过程中会将版本信息注入到构建产物中，便于运行时查询和问题定位。版本信息包括版本号（AppVersion）、提交 SHA（CommitSHA）、构建类型（BuildType）和构建时间（BuildTime）。

版本信息的生成逻辑在 workflow 中通过脚本实现，以 Backend 为例：

```yaml
- name: Set version variables
  id: set-version
  run: |
    COMMIT_SHA="${{ github.sha }}"
    SHORT_SHA="${COMMIT_SHA:0:7}"

    # Check if triggered by tag
    if [[ "${{ github.ref_type }}" == "tag" ]]; then
      APP_VERSION="${{ github.ref_name }}"
      BUILD_TYPE="release"
    else
      APP_VERSION="$SHORT_SHA"
      BUILD_TYPE="development"
    fi

    echo "app_version=$APP_VERSION" >> $GITHUB_OUTPUT
    echo "commit_sha=$COMMIT_SHA" >> $GITHUB_OUTPUT
    echo "build_type=$BUILD_TYPE" >> $GITHUB_OUTPUT
    echo "build_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> $GITHUB_OUTPUT
```

脚本通过检查 `github.ref_type` 判断触发类型：标签触发时使用标签名称作为版本号，BUILD_TYPE 设置为 "release"；分支触发时使用 commit SHA 的前 7 位作为版本号，BUILD_TYPE 设置为 "development"。

对于 Backend 和 Storage 这类 Go 项目，版本信息通过 `ldflags` 注入到二进制文件中：

```yaml
go build -ldflags="-X main.AppVersion=${{ steps.set-version.outputs.app_version }} \
  -X main.CommitSHA=${{ steps.set-version.outputs.commit_sha }} \
  -X main.BuildType=${{ steps.set-version.outputs.build_type }} \
  -X main.BuildTime=${{ steps.set-version.outputs.build_time }} -w -s" \
  -o bin/linux_amd64/controller cmd/crater/main.go
```

`-X` 参数用于设置包变量的值，将版本信息编译到二进制文件中，运行时可以通过程序接口查询这些信息。

对于 Frontend 项目，版本信息通过环境变量注入到构建过程中：

```yaml
echo "VITE_APP_VERSION=$APP_VERSION" >> $GITHUB_ENV
echo "VITE_APP_COMMIT_SHA=$COMMIT_SHA" >> $GITHUB_ENV
echo "VITE_APP_BUILD_TYPE=$BUILD_TYPE" >> $GITHUB_ENV
echo "VITE_APP_BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> $GITHUB_ENV
```

这些环境变量在构建时会被 Vite 替换到前端代码中，用户可以在前端界面查看版本信息。

通常来说，前端和后端的最新版本应该是一致的，但是如果某个修改仅仅修改了其中之一，那么将会导致前后端最新版本不一致的情况出现。

在使用镜像部署时，建议用户同步前后端版本，目前项目不保证不同前后端版本之间的兼容性。

### 跨平台构建

构建发布阶段支持同时构建 linux/amd64 和 linux/arm64 两个平台的镜像，满足不同硬件架构的需求。跨平台构建分为两个阶段：先为不同平台编译二进制文件，然后使用 Docker Buildx 构建多平台镜像。

对于 Backend 和 Storage 这类需要编译的项目，使用 GitHub Actions 的 matrix strategy 并行构建不同平台的二进制文件：

```yaml
build_backend:
  strategy:
    matrix:
      platform:
        - goos: linux
          goarch: amd64
          image_platform: linux/amd64
        - goos: linux
          goarch: arm64
          image_platform: linux/arm64
  steps:
    - name: Build backend binaries
      run: |
        go build -ldflags="..." -o bin/${{ matrix.platform.image_platform }}/controller cmd/crater/main.go
      env:
        GOOS: ${{ matrix.platform.goos }}
        GOARCH: ${{ matrix.platform.goarch }}
```

通过设置 `GOOS` 和 `GOARCH` 环境变量，Go 编译器会为目标平台生成对应的二进制文件。构建完成后，各平台的二进制文件会被上传为构建产物（artifact），供后续镜像构建使用。

镜像构建阶段使用 Docker Buildx 和 QEMU 实现跨平台镜像构建：

```yaml
- name: Set up QEMU
  uses: docker/setup-qemu-action@v3

- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Build and push multi-platform image
  uses: docker/build-push-action@v6
  with:
    context: ./backend
    file: ./backend/Dockerfile
    platforms: linux/amd64,linux/arm64
    push: true
```

QEMU 通过 CPU 模拟实现跨平台构建，允许在 amd64 架构的构建机器上构建 arm64 镜像。Docker Buildx 是 Docker 的扩展构建工具，支持多平台构建和镜像清单（manifest）管理。构建完成后，Docker Buildx 会创建一个包含多个平台镜像的 manifest，用户拉取镜像时会自动选择匹配的平台版本。

对于 Frontend 这类不需要编译的项目，构建产物是静态资源文件（HTML、CSS、JavaScript 等），而非二进制可执行文件，因此不需要交叉编译。前端构建过程不依赖目标平台架构，只需将构建产物打包到不同平台的基础镜像中即可，Docker Buildx 会根据目标平台选择合适的基础镜像（如 Nginx）来提供静态文件服务。

### 镜像标签策略

构建完成后，镜像会被打上多个标签，便于用户根据不同的使用场景选择合适的版本。标签生成使用 `docker/metadata-action`，配置如下：

```yaml
- name: Docker meta
  id: meta
  uses: docker/metadata-action@v5
  with:
    images: ${{ env.REGISTRY }}/${{ env.IMAGE_REPO }}
    tags: |
      type=ref,event=branch
      type=semver,pattern={{version}}
      type=semver,pattern={{major}}.{{minor}}
      type=semver,pattern={{major}}
      type=raw,value=latest,enable={{is_default_branch}}
      type=sha
```

`images` 参数指定镜像的基础名称，由 `${{ env.REGISTRY }}` 和 `${{ env.IMAGE_REPO }}` 组成，完整的镜像地址将在下一节"镜像推送与清理"中详细说明。

`tags` 参数下的每一行都是一个独立的标签生成规则，这些规则会根据触发条件并行生成对应的标签。例如，当创建版本标签 `v1.2.3` 时，会同时生成多个标签：`v1.2.3`、`1.2`、`1` 和 SHA 标签；当推送到 main 分支时，会生成 `main`、`latest` 和 SHA 标签。

各标签规则的参数说明：

- **`type=ref,event=branch`**：`type=ref` 表示基于 Git 引用生成标签，`event=branch` 指定仅在分支推送事件时生效。当推送到分支时，使用分支名作为标签。
- **语义化版本标签**（`type=semver`）：基于语义化版本生成标签，仅在创建版本标签时生效。包含以下三种模式：
  - `pattern={{version}}`：使用完整版本号（如 `v1.2.3`）
  - `pattern={{major}}.{{minor}}`：使用主次版本号（如 `1.2`），当创建 `v1.2.3` 时会生成 `1.2` 标签，该标签会指向该版本系列的最新版本
  - `pattern={{major}}`：使用主版本号（如 `1`），当创建 `v1.2.3` 时会生成 `1` 标签，该标签会指向该主版本系列的最新版本
- **`type=raw,value=latest,enable={{is_default_branch}}`**：`type=raw` 表示使用原始值作为标签，`value=latest` 指定标签值为 `latest`，`enable={{is_default_branch}}` 表示仅在默认分支（main）时生成此标签。
- **`type=sha`**：基于 commit SHA 生成标签，格式为 `sha-<SHA前7位>`，所有构建都会生成此标签，便于精确追踪构建来源。

### 镜像推送与清理

镜像构建完成后，需要推送到镜像仓库并清理旧镜像以控制存储空间。镜像推送使用 GHCR（GitHub Container Registry）作为仓库，完整的镜像地址格式为 `${{ env.REGISTRY }}/${{ env.IMAGE_REPO }}`，即 `ghcr.io/raids-lab/crater-backend`、`ghcr.io/raids-lab/crater-frontend` 和 `ghcr.io/raids-lab/storage-server`。

推送前需要先登录到 GHCR，配置如下：

```yaml
- name: Login to GHCR
  uses: docker/login-action@v3
  with:
    registry: ${{ env.REGISTRY }}
    username: ${{ github.repository_owner }}
    password: ${{ secrets.GITHUB_TOKEN }}
```

使用 `GITHUB_TOKEN` 作为认证凭据，该 token 由 GitHub Actions 自动提供，无需额外配置。`github.repository_owner` 是仓库所有者（组织或用户名），对于本项目为 `raids-lab`。

镜像构建和推送配置如下：

```yaml
- name: Build and push multi-platform image
  uses: docker/build-push-action@v6
  with:
    context: ./backend
    file: ./backend/Dockerfile
    platforms: linux/amd64,linux/arm64
    push: true
    tags: ${{ steps.meta.outputs.tags }}
```

`tags` 参数使用上一节中 `docker/metadata-action` 生成的标签列表，构建完成后会将镜像推送到 GHCR，并打上所有生成的标签。用户可以通过 `docker pull ghcr.io/raids-lab/crater-backend:<tag>` 拉取镜像。推送的镜像可以在 GitHub 仓库的 Packages 页面中查看，包括所有标签和版本信息。

构建完成后会自动清理旧镜像，配置如下：

```yaml
- uses: quartx-analytics/ghcr-cleaner@v1
  with:
    owner-type: org
    token: ${{ secrets.PAT_TOKEN }}
    repository-owner: ${{ github.repository_owner }}
    package-name: crater-backend
    delete-untagged: true
    keep-at-most: 2
    skip-tags: v*
```

清理规则说明：

- `delete-untagged: true`：删除未标记的镜像层（dangling images），这些是构建过程中产生的中间层，不再被任何标签引用。
- `keep-at-most: 2`：每个包最多保留 2 个未标记镜像，超出数量的旧镜像会被删除。
- `skip-tags: v*`：跳过以 `v` 开头的标签（版本标签），保护所有版本镜像不被删除，确保用户可以访问历史版本。

### PR Check

除了构建发布阶段，CI 还为 Pull Request 设置了检查流程，防止坏代码进入主分支。PR Check 的触发机制与构建发布阶段一致，使用同样的路径过滤，只有当相关代码或 workflow 文件变更时才会触发对应组件的检查。

PR Check 包含两个阶段：Lint Check 和 Build Check，使用与构建 workflow 一致的流程以进行检查。与构建发布阶段不同，PR Check 只构建单平台（linux/amd64）以节省构建时间，且不推送镜像，仅验证构建是否成功。

但需要特别注意的是，在目前的 workflow 配置下，我们无法在 GitHub 分支保护规则 *Require status checks to pass* 中要求这些 PR Check 必须通过。因为没有被路径触发的 workflow 不会被视为通过，而是一直处于 Pending 状态，这将永久阻塞 PR 合并。

---

## 依赖镜像

依赖镜像包括 `buildx-client`、`envd-client` 和 `nerdctl-client` 三个构建工具相关的 Docker 镜像。`buildx-client` 和 `envd-client` 用于支持平台的镜像制作功能，`nerdctl-client` 用于将运行中的容器快照（commit）为镜像，支持 Jupyter 和 Custom job 类型的容器快照功能。

这些镜像虽作为后端功能的依赖，但由于构建流程与后端显著不同，因此单独说明。

### 概述

依赖镜像的 CI 流程采用变更检测机制，只有当某个镜像的 Dockerfile 或相关文件发生变更时，才会触发该镜像的构建。输入为 Dockerfile 和相关文件（位于 `hack/depend-image-dockerfile/` 目录），输出为多平台 Docker 镜像，产物保存在 GHCR 的 `ghcr.io/raids-lab/buildx-client`、`ghcr.io/raids-lab/envd-client` 和 `ghcr.io/raids-lab/nerdctl-client` 仓库中，构建完成后会自动清理旧镜像以控制存储空间。每个镜像的构建流程独立，包括多平台镜像构建、标签生成和推送。

### 触发

依赖镜像的构建仅在代码推送到 main 分支时触发，且只监听 `hack/depend-image-dockerfile/**` 目录的变更。与前端后端不同，依赖镜像不支持标签触发，这是因为依赖镜像的版本由环境变量固定（如 `BUILDX_VERSION`、`ENVD_VERSION`、`NERDCTL_VERSION`），不需要通过标签来管理版本。

workflow 的触发配置如下：

```yaml
on:
  push:
    branches: [main]
    paths:
      - ".github/workflows/depend-build.yml"
      - "hack/depend-image-dockerfile/**"
```

workflow 使用 `detect-changes` job 检测哪些镜像需要构建，通过 `dorny/paths-filter` 检查各个镜像目录的变更情况。只有当某个镜像的 Dockerfile 或相关文件发生变更时，对应的构建 job 才会执行，避免了不必要的构建开销。变更检测的配置如下：

```yaml
detect-changes:
  runs-on: ubuntu-latest
  outputs:
    buildx-client: ${{ steps.changes.outputs.buildx-client }}
    envd-client: ${{ steps.changes.outputs.envd-client }}
    nerdctl-client: ${{ steps.changes.outputs.nerdctl-client }}
  steps:
    - name: Detect changes
      uses: dorny/paths-filter@v3
      id: changes
      with:
        filters: |
          buildx-client:
            - 'hack/depend-image-dockerfile/buildx-client/**'
          envd-client:
            - 'hack/depend-image-dockerfile/envd-client/**'
          nerdctl-client:
            - 'hack/depend-image-dockerfile/nerdctl-client/**'
```

每个构建 job 通过 `needs: detect-changes` 和条件判断 `if: needs.detect-changes.outputs.buildx-client == 'true'` 来决定是否执行。

### 镜像标签策略

镜像标签策略与前端后端类似，但版本号标签使用环境变量中定义的固定版本号，而不是从 Git 标签解析。标签生成配置如下：

```yaml
- name: Docker meta for buildx-client
  id: meta
  uses: docker/metadata-action@v5
  with:
    images: ${{ env.REGISTRY }}/${{ env.REPOSITORY }}/${{ env.BUILDX_CLIENT }}
    tags: |
      type=ref,event=branch
      type=sha
      type=raw,value=latest,enable={{is_default_branch}}
      type=raw,value=${{ env.BUILDX_VERSION }}
```

`images` 参数由 `${{ env.REGISTRY }}`（`ghcr.io`）、`${{ env.REPOSITORY }}`（`raids-lab`）和镜像名称（如 `buildx-client`）组成，完整的镜像地址将在下一节说明。

标签规则说明：

- **`type=ref,event=branch`**：基于分支名生成标签，推送到 main 分支时生成 `main` 标签。
- **`type=sha`**：基于 commit SHA 生成标签，格式为 `sha-<SHA前7位>`，所有构建都会生成此标签。
- **`type=raw,value=latest,enable={{is_default_branch}}`**：仅在默认分支（main）时生成 `latest` 标签。
- **`type=raw,value=${{ env.BUILDX_VERSION }}`**：使用环境变量中定义的版本号（如 `v0.25.0`）作为标签，这是依赖镜像特有的标签类型，用于标记依赖工具的固定版本。

每个镜像都有对应的版本环境变量：`BUILDX_VERSION`、`ENVD_VERSION` 和 `NERDCTL_VERSION`，这些版本号在 workflow 的环境变量中定义，与依赖工具的发布版本保持一致。

### 镜像构建

依赖镜像的构建使用 Docker Buildx 进行多平台构建，与前端后端类似，通过 QEMU 模拟实现跨平台支持。不同镜像的平台支持情况不同：`buildx-client` 和 `nerdctl-client` 支持 `linux/amd64` 和 `linux/arm64` 两个平台，而 `envd-client` 仅支持 `linux/amd64` 平台。

与前端后端不同，依赖镜像的构建过程更简单：直接使用 Dockerfile 构建，无需先编译源代码。依赖镜像的 Dockerfile 位于 `hack/depend-image-dockerfile/` 目录下，每个镜像都有独立的 Dockerfile，用于安装和配置对应的工具（如 `buildx`、`envd`、`nerdctl`）。

镜像构建的配置如下（以 `buildx-client` 为例）：

```yaml
- name: Set up QEMU
  uses: docker/setup-qemu-action@v3

- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Build and push buildx-client image
  uses: docker/build-push-action@v6
  with:
    context: hack/depend-image-dockerfile/buildx-client
    file: hack/depend-image-dockerfile/buildx-client/Dockerfile
    platforms: linux/amd64,linux/arm64
    push: true
    tags: ${{ steps.meta.outputs.tags }}
```

构建上下文设置为各个镜像的 Dockerfile 所在目录，通过 `platforms` 参数指定需要构建的平台。

`tags` 参数使用上一节中 `docker/metadata-action` 生成的标签列表（`${{ steps.meta.outputs.tags }}`），用于为构建的镜像打标签。

虽然构建和推送在同一个 step（`docker/build-push-action@v6`）中完成，但逻辑上是分离的，本节主要介绍构建过程，推送相关内容将在下一节说明。

### 镜像推送与清理

依赖镜像的推送机制与前端后端相同，使用 GHCR 作为仓库，镜像地址为 `ghcr.io/raids-lab/buildx-client`、`ghcr.io/raids-lab/envd-client` 和 `ghcr.io/raids-lab/nerdctl-client`。镜像推送在上一节的 `docker/build-push-action@v6` 步骤中完成（通过 `push: true` 参数启用），构建完成后会自动将镜像推送到 GHCR，并打上所有生成的标签。推送的镜像可以在 GitHub 仓库的 Packages 页面中查看。

镜像清理使用 `ghcr-cleaner`，配置与前端后端类似，但参数有所不同：

```yaml
- uses: quartx-analytics/ghcr-cleaner@v1
  with:
    package-name: ${{ env.BUILDX_CLIENT }}
    delete-untagged: true
    keep-at-most: 5
    skip-tags: latest
```

主要区别：

- `keep-at-most: 5`：相比前端后端（保留 2 个），依赖镜像保留更多未标记镜像，因为依赖镜像的更新频率相对较低。
- `skip-tags: latest`：跳过 `latest` 标签，保护最新版本镜像，而前端后端使用 `skip-tags: v*` 保护所有版本标签。

三个依赖镜像都会执行相同的清理步骤，分别清理各自的镜像仓库。

---

## 文档网站

文档网站基于 Next.js 构建，部署在 GitHub Pages 上，包含 Crater 部署和使用的教程。

### 概述

文档网站的 CI/CD 流程包括 CI（持续集成）和 CD（持续部署）两部分。CI 部分包括 PR 检查、自动修正和自动翻译：PR 检查阶段在代码合并前执行，进行构建验证和图片格式检查；自动修正阶段在 PR 时自动修正文档格式；自动翻译阶段在代码合并后自动翻译文档并创建翻译 PR。CD 部分包括文档部署：在代码合并后自动构建 Next.js 网站并部署到 GitHub Pages。输入为文档源代码（Markdown、MDX、配置文件等），输出为部署在 GitHub Pages 上的静态网站。

与前后端和依赖镜像不同，文档网站不仅包含 CI（构建和验证），还包含 CD（自动部署），实现了从代码变更到生产环境的全自动化流程。

以创建一个更新文档网站的 PR 为例，执行流程如下：

1. **PR 创建时**：当 PR 修改了 `website/src/**` 或 `website/content/**` 目录下的文件时，会同时触发两个 workflow：
   - **PR 检查**（`docs-build.yml`）：构建 Next.js 网站验证构建是否成功，检查新增或修改的图片是否为 WebP 格式
   - **自动修正**（`docs-autocorrect.yml`）：自动修正文档格式，对于内部 PR 会直接提交修正，对于 Fork PR 会在 PR 评论中报告问题

2. **PR 合并后**：当 PR 合并到 main 分支后，会触发两个 workflow：
   - **文档部署**（`docs-deploy.yml`）：构建包含 Orama 搜索索引的 Next.js 网站，并部署到 GitHub Pages，文档网站自动更新
   - **自动翻译**（`docs-autotranslate.yml`）：根据文件变更类型和 PR 标签智能过滤需要翻译的文件，运行翻译脚本，并创建包含翻译结果的 PR（分支名为 `feature/auto-translate-{run_id}`）

3. **翻译 PR 创建后**：自动翻译创建的 PR 会触发 PR 检查和自动修正 workflow，但由于翻译 PR 的 commit 消息以 `chore(i18n):` 开头，自动翻译 workflow 的防循环机制会跳过该 PR，避免无限循环。

4. **翻译 PR 合并后**：翻译 PR 合并到 main 分支后，会触发文档部署 workflow，更新文档网站。由于 commit 消息以 `chore(i18n):` 开头，不会再次触发自动翻译 workflow。

需要注意的是，目前自动翻译流程可靠性仍较弱，还有待完善。

### PR 检查

PR 检查在创建 Pull Request 时触发，监听 `website/src/**`、`website/content/**`、`website/package.json` 和 `website/pnpm-lock.yaml` 的变更。检查流程包括构建验证和图片格式检查两个步骤。

构建验证使用 Next.js 构建网站，确保文档可以正常构建：

```yaml
- name: Build website
  run: pnpm exec next build
```

图片格式检查确保 PR 中新增或修改的图片使用 WebP 格式，检查范围包括 `src/` 和 `content/` 目录。如果发现非 WebP 格式的图片（如 PNG、JPG、JPEG、GIF、BMP），PR 检查会失败，提示用户将图片转换为 WebP 格式。

### 文档构建和部署

文档部署在代码推送到 main 分支时触发，监听 `website/src/**`、`website/content/**` 和 `website/package.json` 的变更。部署流程包括构建和部署两个阶段。

构建阶段使用 Next.js 构建网站，包含 Orama 搜索索引（需要 `ORAMA_PRIVATE_API_KEY` 和 `ORAMA_INDEX_NAME` 环境变量），并创建 `.nojekyll` 文件以防止 GitHub Pages 使用 Jekyll 处理：

```yaml
- name: Build website
  env:
    ORAMA_PRIVATE_API_KEY: ${{ secrets.ORAMA_PRIVATE_API_KEY }}
    ORAMA_INDEX_NAME: ${{ secrets.ORAMA_INDEX_NAME }}
  run: pnpm build

- name: Create .nojekyll file
  run: touch ./out/.nojekyll
```

部署阶段使用 `actions/deploy-pages@v4` 将构建产物部署到 GitHub Pages，需要 `pages: write` 和 `id-token: write` 权限，并使用 `github-pages` 环境：

```yaml
- name: Deploy to GitHub Pages
  uses: actions/deploy-pages@v4
```

部署完成后，文档网站会自动更新，用户可以通过 GitHub Pages 的 URL 访问最新版本的文档。

### 自动修正

自动修正使用 `autocorrect` 工具自动修正文档格式，在创建 Pull Request 时触发，监听 `website/src/**` 和 `website/content/**` 的变更。根据 PR 来源的不同，采用不同的处理策略。

对于内部 PR（来自同一仓库），自动修正会直接修复文件并提交更改：

```yaml
- name: AutoCorrect and Fix (for internal PRs)
  uses: huacnlee/autocorrect-action@v2
  with:
    args: --fix ${{ steps.internal_files.outputs.files }}

- name: Commit changes (for internal PRs)
  uses: stefanzweifel/git-auto-commit-action@v5
```

对于 Fork PR（来自外部仓库），由于权限限制，无法直接提交更改，因此使用 Reviewdog 在 PR 评论中报告格式问题，由贡献者自行修正。

自动修正会排除 `*.*.mdx` 文件（多语言文件），避免影响翻译文件。

### 自动翻译

自动翻译在代码推送到 main 分支时触发，监听 `website/content/docs/**`、`website/messages/**` 和 `website/src/i18n/config.ts` 的变更。翻译流程使用 GitHub App 进行身份认证，通过智能过滤机制确定需要翻译的文件，然后运行 Python 脚本进行翻译，最后创建包含翻译结果的 PR。

防循环机制通过检查 commit 消息来避免无限循环：跳过以 `chore(i18n):` 开头或包含 `from feature/auto-translate-` 的提交。

智能过滤机制基于文件变更类型和 PR 标签：

- **新增文件**：始终翻译
- **修改文件**：检查来源 PR 的标签，只有带 `run-translation` 标签的 PR 修改的文件才会翻译
- **跳过条件**：如果 PR 有 `no-translation` 标签，整个 workflow 会跳过

翻译完成后，会自动创建翻译 PR，分支名为 `feature/auto-translate-{run_id}`，PR 标题为 `🌐 [Auto-Translate] 同步多语言文件`，标签为 `i18n, automated`。PR 创建后会自动删除分支，避免分支积累。

---

## Helm Chart

Helm Chart 用于将 Crater 平台部署到 Kubernetes 集群，提供了一键部署和配置管理能力。Helm Chart 的 CI 流程包括 Chart 验证和 Chart 发布两个阶段，确保 Chart 质量和自动化发布。

### 概述

Helm Chart 的 CI 流程采用两阶段设计：Chart 验证阶段在 PR 时执行，进行语法验证、模板验证和版本号检查；Chart 发布阶段在代码合并到 main 分支或创建 Release 时执行，打包 Chart 并推送到 GHCR OCI 仓库。输入为 Chart 源代码（位于 `charts/crater/` 目录），输出为打包后的 Helm Chart（`.tgz` 文件），产物保存在 GHCR 的 `ghcr.io/raids-lab/crater` OCI 仓库中。

Chart 验证确保 Chart 的正确性和完整性，包括语法检查、模板渲染验证和版本号更新检查。Chart 发布将验证通过的 Chart 打包并推送到 GHCR，用户可以通过 `helm install crater oci://ghcr.io/raids-lab/crater --version <version>` 安装 Chart。

### Chart 验证

Chart 验证在创建 Pull Request 时触发，监听 `charts/**` 目录的变更。验证流程包括语法验证、模板验证、版本号检查和打包测试四个步骤。

语法验证使用 `helm lint` 检查 Chart 的语法、依赖和模板等：

```yaml
- name: Validate Chart Syntax
  run: |
    cd charts
    helm lint crater/
    helm template crater crater/ --dry-run
```

`helm lint` 检查 Chart 的语法错误、依赖关系和最佳实践；`helm template --dry-run` 验证模板能否正确渲染，确保模板语法正确且所有必需的值都已提供。

版本号检查确保每次 PR 都更新了 Chart 版本号，通过比较当前分支和基础分支的版本号来实现：

```bash
CURRENT_VERSION=$(helm show chart charts/crater/ | grep '^version:' | awk '{print $2}')
BASE_VERSION=$(git show "origin/$BASE_BRANCH:charts/crater/Chart.yaml" | grep '^version:' | awk '{print $2}')

if [ "$CURRENT_VERSION" = "$BASE_VERSION" ]; then
  echo "⚠️  Chart version has not been updated"
  exit 1
fi
```

如果版本号未更新，PR 检查会失败，提示用户更新版本号并遵循语义化版本规范。

打包测试使用 `helm package` 测试 Chart 能否正常打包，打包完成后会删除打包文件，仅用于验证：

```yaml
- name: Package Chart (Test)
  run: |
    cd charts
    helm lint crater/
    helm package crater/
    rm -f crater-*.tgz
```

### Chart 发布

Chart 发布在代码推送到 main 分支、创建 Release 或手动触发时执行，监听 `charts/**` 目录的变更。发布流程包括打包和推送两个步骤。

打包阶段使用 `helm package` 将 Chart 打包成 `.tgz` 文件，并从 `Chart.yaml` 中读取版本号：

```yaml
- name: Package and Push Helm Chart
  run: |
    cd charts
    helm package crater/
    CHART_VERSION=$(helm show chart crater/ | grep '^version:' | awk '{print $2}')
```

推送阶段使用 `helm push` 将打包的 Chart 推送到 GHCR OCI 仓库：

```yaml
- name: Login to GHCR
  uses: docker/login-action@v3
  with:
    registry: ${{ env.REGISTRY }}
    username: ${{ github.repository_owner }}
    password: ${{ secrets.GITHUB_TOKEN }}

- name: Package and Push Helm Chart
  run: |
    helm push crater-${CHART_VERSION}.tgz oci://${{ env.REGISTRY }}/${{ env.REPOSITORY }}
```

Chart 推送到 `ghcr.io/raids-lab/crater` OCI 仓库后，用户可以通过以下命令安装：

```bash
helm registry login ghcr.io
helm install crater oci://ghcr.io/raids-lab/crater --version <version>
```

发布完成后会自动清理旧版本 Chart，使用 `ghcr-cleaner` 清理未标记的 Chart，保留最多 10 个版本，并跳过 `latest` 标签：

```yaml
- uses: quartx-analytics/ghcr-cleaner@v1
  with:
    package-name: crater
    delete-untagged: true
    keep-at-most: 10
    skip-tags: latest
```

### 版本管理

Helm Chart 使用语义化版本（Semantic Versioning）管理版本号，版本号定义在 `charts/crater/Chart.yaml` 的 `version` 字段中。版本号格式为 `MAJOR.MINOR.PATCH`（如 `0.1.1`），遵循以下规则：

- **MAJOR**：不兼容的 API 变更
- **MINOR**：向后兼容的功能新增
- **PATCH**：向后兼容的问题修复

PR 检查阶段会强制要求更新版本号，确保每次 Chart 变更都有对应的版本号更新。这有助于用户追踪 Chart 的变更历史，并在升级时选择合适的版本。

Chart 发布时，版本号会作为标签推送到 GHCR，用户可以通过版本号安装特定版本的 Chart。清理机制会保留最多 10 个版本的 Chart，确保用户可以访问历史版本，同时控制存储空间。