[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

# 为 Crater Backend 贡献

本文档覆盖 `backend/`，包括集成在同一个 Go 模块中的 `internal/storage/` 服务。请先阅读仓库根 [CONTRIBUTING](../docs/zh-CN/CONTRIBUTING.md)，了解全仓库流程、分支规则、PR 要求和跨模块约束。

修改 handler、service、DAO/model、数据库迁移、storage-server、后端配置或外部 API 行为时，使用本文档。

## 本地环境

### 工具链

- **Go**：`backend/go.mod` 当前要求 `1.25.4`。执行后端构建、测试或本地运行 target 前，先运行 `go version`。可用 [gvm](https://github.com/moovweb/gvm)（推荐 `v1.0.22`）在 `go.mod` 所在目录执行 `gvm applymod`，也可直接安装 Go。
- **kubectl**：必需，推荐 `v1.33`。运行项目需要 Kubernetes 集群；测试 / 学习可用 Kind 或 Minikube。

直接安装 Go 时可能需要：

```bash
export GOROOT=/usr/local/go        # 改成你的 Go 安装路径
export PATH=$PATH:$GOROOT/bin
go env -w GOPROXY=https://goproxy.cn,direct
```

## 本地调试

### 运行配置

推荐通过仓库根的统一配置 target（`make config-link`、`make config-status`、`make config-unlink`、`make config-restore`）管理本地配置。真实本地运行配置可能需要管理员提供 Kubernetes、数据库、网络或外部集成设置。不要提交私有配置。

后端通常需要和前端一起调试，并通过配置连接测试集群中已有的依赖服务。一般不需要在本机启动 PostgreSQL、复刻 Kubernetes，也不需要运行 Crater 的全部组件。如果需要完整的本地前后端体验，应同时运行主后端（`make run`）和 storage server（`make run-storage`）。文件浏览、上传 / 下载、数据集、模型，以及所有经前端 `/api/ss` 代理的请求都需要 storage server。如果当前任务不涉及存储相关页面或 API，可以暂时跳过 `make run-storage`，只运行主后端和前端。

Backend 使用：

- **`kubeconfig`**：Kubernetes 客户端配置。Backend 先读 `KUBECONFIG`；不存在则读当前目录的 `kubeconfig`。
- **`./etc/debug-config.yaml`**：服务端口、metrics/profiling、PostgreSQL、工作区命名空间 / 存储 / ingress、镜像仓库、SMTP、调度器和作业类型开关等应用配置。示例见 [`etc/example-config.yaml`](https://github.com/raids-lab/crater-backend/blob/main/etc/example-config.yaml)。
- **`.debug.env`**：执行 `make run` 时创建，被 git 忽略，用于个人配置；当前用于服务端口：

  ```env
  CRATER_BE_PORT=:8088
  ```

### 本地运行

```bash
make run
```

服务正常后打开 Swagger UI：`http://localhost:<端口>/swagger/index.html#/`。

完整本地前后端调试可使用三个终端：

```bash
# 终端 1：主后端 API
cd backend
make run

# 终端 2：storage server，存储相关 UI/API 流程需要它
cd backend
make run-storage

# 终端 3：前端开发服务
cd frontend
make run
```

`make run` 是依赖本地环境的验证步骤，不是 Agent 默认必须执行的检查。`make build` 或测试可以通过，但 `make run` 仍可能因配置、Kubernetes 访问、数据库连接或网络访问缺失而失败。如果失败指向缺少配置、凭据、集群访问或管理员才能处理的环境问题，应停下来告知开发者需要检查什么，不要反复尝试。

常用 target：

| 命令 | 说明 |
|------|------|
| `make run` | 本地运行后端 |
| `make lint` / `make vet` / `make imports` | 代码检查与格式 |
| `make migrate` | 执行数据库迁移 |
| `make curd` | 生成 GORM CRUD 代码 |
| `make docs` | 生成 swag 文档 |
| `make pre-commit-check` | 手动运行 pre-commit 检查 |
| `make build` / `make build-migrate` | 构建主程序 / 迁移程序 |
| `make run-storage` | 本地运行 storage server，默认端口为 `7320` |
| `make build-storage` | 只构建 storage server 二进制，不启动服务 |

## API 与 Handler 规则

- **按身份分路由**：管理员接口注册到 `Admin` 路由，用户接口注册到 `Protected` 路由，二者不可混用。
- **按身份命名**：管理员接口函数名加 `Admin` 前缀，用户接口函数名加 `User` 前缀。
- **同步外部 API 文档**：变更外部 API 必须同步 `swag` 注释；若判断无需修改注释，要明确说明。
- **保持 Handler 瘦身**：Handler 只负责请求分发与响应，复杂业务逻辑放到 Service 层。

## 作业模板兼容性

克隆作业依赖创建作业时保存的作业模板。当前模板 payload 是前端基于 `frontend/src/components/form/types.ts` metadata 生成的导入 / 导出 JSON（`version`、`type`、`data`），后端将它作为不透明 template 字符串保存。

修改作业创建配置字段、请求 / 响应结构或模板序列化方式时：

- 将作业模板 payload 视为带版本 schema。如果本次改动应阻断旧模板或旧导出配置继续导入 / 克隆，必须提升对应前端 `MetadataForm*` version，让阻断显式且有意。
- 如果旧模板仍应可用，补充必要的迁移 / 兼容处理，而不是静默接受不匹配的数据。
- 保持模板创建、克隆 / 回放与公开 API 行为一致；payload 变化时同步更新 `swag` 注释和前端作业创建 / 克隆处理。
- 验证当前版本克隆 / 导入成功，废弃版本要么被迁移，要么被清晰阻断。

## 错误处理与安全

后端错误是面向用户的 API 契约的一部分。错误响应必须给前端和 CLI 足够的结构化事实，让用户知道哪里出了问题，也要给管理员留下稳定的排查依据。

- 将实现定义文件作为错误契约的实时参考：`backend/internal/bizerr/groups.go` 定义业务错误 group 和业务码；`backend/internal/resputil/handle.go` 定义这些 group 到 HTTP 状态码的映射；`backend/internal/resputil/response.go` 定义 `code` / `data` / `msg` 响应信封。前端生成常量来自 `frontend/src/services/generator.py` 和 `frontend/src/services/error_code.ts`。不要在文档中复制长错误码表；具体 code 只作为示例出现。
- 使用符合 `resputil.HandleError` 语义的 RESTful HTTP 状态码，不要把所有失败都压成 `500` 或旧的 `Error()` helper。示例：调用方输入非法属于 `400xx` group，状态或依赖冲突属于 `409xx`，平台或依赖故障属于 `5xx`。这些只是示例；选择或新增 code 前先查上面的定义文件。
- 新代码应返回 `bizerr` 错误并通过 `resputil.HandleError` 输出；`backend/internal/resputil/code.go` 仅保留为旧代码兼容入口。新增错误前先优先选择已有 `bizerr` group。
- 只有当前端、CLI 或外部调用方确实需要稳定的机器可读原因时，才在 `backend/internal/bizerr/groups.go` 中新增业务错误码。业务码所属 group 必须和 `resputil.HandleError` 选择的 HTTP 状态保持一致。
- 返回给客户端的错误信息必须是清晰、准确的英文。优先给出可行动的信息，说明具体非法字段、缺失资源、冲突状态、所需权限或不可用依赖。调用方可以据此修正时，不要返回 `failed`、`invalid` 这类泛化描述，也不要直接返回 Go 原始错误。
- 用户可见信息必须安全：不得包含密钥、Token、内网 IP、kubeconfig、SQL 片段、堆栈或私有基础设施细节。底层错误用 `bizerr.*.Wrap` 包装，让服务端日志保留 cause，同时保证 API 响应安全。
- 新增或修改 API 错误路径时，必须确认前端和 CLI 将如何展示。响应信封保持 `code`、`data`、`msg`；客户端可以直接展示 `msg`，并暴露 `http_status` / 业务 `code` 作为支持排查事实。如果页面需要特殊交互，说明处理哪个业务码以及原因。
- 严禁在 `backend/internal/storage/` 或 DAO 层拼接 SQL 字符串，必须使用参数化查询。
- 严禁硬编码密钥、Token、密码、内网 IP、kubeconfig 或生产凭据。

## 数据库变更

数据库结构变更使用 GORM + gormigrate + GORM Gen。

1. 修改 `dao/model/*.go` 中的模型结构体。
2. 在 `cmd/gorm-gen/models/migrate.go` 添加迁移项。
3. 迁移 ID 使用 `YYYYMMDDHHmm` 格式。
4. 同时实现 `Migrate` 与 `Rollback`。
5. 运行 `make migrate`。
6. 运行 `make curd` 重新生成 CRUD helper。
7. 将模型、迁移、生成代码和业务代码一起提交。

拉取代码后如果发现 `migrate.go` 更新，应运行 `make migrate`。全新数据库会由 `make migrate` 自动初始化；Helm 部署生产环境时通过 InitContainer 自动迁移。

详细流程见 [`cmd/gorm-gen/README.md`](./cmd/gorm-gen/README.md)。

## Storage Server

存储服务由同一后端 Go 模块中的 `cmd/storage-server/main.go` 构建。本地启动使用 `make run-storage`；只有需要验证或打包 storage server 二进制时才使用 `make build-storage`。

```bash
make run-storage
make build-storage
```

运行时环境变量：

- `CRATER_STORAGE_PORT`（优先，回退 `PORT`，默认 `7320`）
- `CRATER_STORAGE_ROOT`（优先，回退 `ROOTDIR`，默认 `/crater`）

本地调试可将这些变量写入 `backend/.debug.env` 后运行 `make run-storage`。如果 `make run-storage` 因所需配置、文件系统权限、Kubernetes 访问或管理员提供的设置不可用而失败，应停下来要求开发者检查环境，不要反复尝试。

## VSCode 调试

仓库根 `.vscode/launch.json` 提供「Backend Debug Server」配置。在根目录打开项目、设置断点、按 `F5` 选择该配置即可。其 `cwd` 是 `${workspaceFolder}/backend`，`program` 指向 `backend/cmd/crater/main.go`，并通过 `KUBECONFIG` 连接集群。

## 提交 Backend 改动前

- 运行相关 `make` target，最终检查通常使用 `make pre-commit-check`。
- 如果 API 行为变化，确认 `swag` 注释、前端调用、CLI 行为和错误展示已对齐。
- 如果作业模板 payload 变化，确认版本和兼容性决策。
- 如果数据库结构变化，包含迁移和生成代码。
- 记录实际检查和开发者人工验证，供 PR 描述使用。
