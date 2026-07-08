[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

# Contributing to Crater Backend

This guide covers `backend/`, including the `internal/storage/` service integrated in the same Go module. Read the root [CONTRIBUTING.md](../CONTRIBUTING.md) first for the repository workflow, branch rules, PR requirements, and cross-module constraints.

Use this file when you change handlers, services, DAO/model code, database migrations, storage-server code, backend configuration, or external API behavior.

## Local Environment

### Toolchain

- **Go**: `backend/go.mod` currently requires `1.25.4`. Before running backend build, test, or local-run targets, check `go version`. You can use [gvm](https://github.com/moovweb/gvm) (recommended `v1.0.22`) and run `gvm applymod` in the directory containing `go.mod`, or install Go directly.
- **kubectl**: required, recommended `v1.33`. Running the project needs a Kubernetes cluster; Kind or Minikube is fine for test/learning.

When installing Go directly you may need:

```bash
export GOROOT=/usr/local/go        # change to your Go install path
export PATH=$PATH:$GOROOT/bin
go env -w GOPROXY=https://goproxy.cn,direct
```

## Local Debugging

### Runtime Config

Prefer the root unified config targets (`make config-link`, `make config-status`, `make config-unlink`, `make config-restore`) for local config. Real local-run config may require administrator-provided Kubernetes, database, network, or integration settings. Do not commit private config.

The backend is normally debugged together with the frontend while connected to existing test-cluster dependencies through config. You usually do not need to start a local PostgreSQL instance, recreate Kubernetes, or run every Crater component on your machine. For a full local frontend-backend experience, run both the main backend (`make run`) and the storage server (`make run-storage`). The storage server is required for file browsing, uploads/downloads, datasets, models, and any frontend request proxied through `/api/ss`. If your current task does not touch storage-related pages or APIs, you may skip `make run-storage` and run only the main backend plus frontend.

Backend uses:

- **`kubeconfig`**: Kubernetes client config. The backend first reads `KUBECONFIG`; if absent, it reads `kubeconfig` in the current directory.
- **`./etc/debug-config.yaml`**: application config for server ports, metrics/profiling, PostgreSQL, workspace namespaces/storage/ingress, image registry, SMTP, scheduler flags, and job-type flags. Example: [`etc/example-config.yaml`](https://github.com/raids-lab/crater-backend/blob/main/etc/example-config.yaml).
- **`.debug.env`**: git-ignored personal config created by `make run`; currently used for service port selection:

  ```env
  CRATER_BE_PORT=:8088
  ```

### Local Run

```bash
make run
```

Once running, open Swagger UI: `http://localhost:<port>/swagger/index.html#/`.

For full local frontend-backend debugging, use three terminals:

```bash
# Terminal 1: main backend API
cd backend
make run

# Terminal 2: storage server, required for storage-related UI/API flows
cd backend
make run-storage

# Terminal 3: frontend dev server
cd frontend
make run
```

`make run` is an environment-dependent local verification step, not a required default Agent check. `make build` or tests may pass even when `make run` fails because config, Kubernetes access, database connectivity, or network access is missing. If a failure points to missing config, credentials, cluster access, or administrator-only setup, stop and tell the developer what to inspect instead of repeatedly retrying.

Common targets:

| Command | Description |
|---------|-------------|
| `make run` | Run the backend locally |
| `make lint` / `make vet` / `make imports` | Lint and formatting |
| `make migrate` | Run database migration |
| `make curd` | Generate GORM CRUD code |
| `make docs` | Generate swag docs |
| `make pre-commit-check` | Run pre-commit checks manually |
| `make build` / `make build-migrate` | Build main / migration binary |
| `make run-storage` | Run the storage server locally, default port `7320` |
| `make build-storage` | Build the storage server binary without starting it |

## API And Handler Rules

- **Route by identity**: admin APIs register to the `Admin` route; user APIs register to the `Protected` route. Do not mix them.
- **Name by identity**: admin API functions take an `Admin` prefix; user API functions take a `User` prefix.
- **Document external APIs**: changing an external API must update its `swag` annotations; if no annotation change is needed, say so explicitly.
- **Keep handlers thin**: handlers dispatch requests and responses. Move complex business logic to the Service layer.

## Job Template Compatibility

Clone-job flows depend on the job template captured with the job at creation time. The current template payload is the frontend import/export JSON (`version`, `type`, `data`) produced from `frontend/src/components/form/types.ts` metadata and stored by the backend as an opaque template string.

When changing job creation configuration fields, request/response schema, or template serialization:

- Treat the job template payload as a versioned schema. If the change should block old templates or exported configs from being imported/cloned, bump the corresponding frontend `MetadataForm*` version so the block is explicit and intentional.
- If old templates should remain usable, add the needed migration/compatibility handling instead of silently accepting mismatched data.
- Keep template creation, clone/replay, and public API behavior aligned; update `swag` annotations and frontend job creation / clone handling when the payload changes.
- Verify current-version clone/import succeeds, and obsolete versions are either migrated or clearly blocked.

## Errors And Security

Backend errors are part of the user-facing API contract. They must give the frontend and CLI enough structured facts to tell users what failed and enough stable facts for administrators to troubleshoot.

- Treat the implementation definitions as the live reference for the error contract: `backend/internal/bizerr/groups.go` defines business error groups and codes; `backend/internal/resputil/handle.go` maps those groups to HTTP status codes; `backend/internal/resputil/response.go` defines the `code` / `data` / `msg` response envelope. The generated frontend constants come from `frontend/src/services/generator.py` and `frontend/src/services/error_code.ts`. Do not copy long error-code tables into documentation; mention concrete codes only as examples.
- Use RESTful HTTP status codes that match the failure semantics in `resputil.HandleError`. Do not collapse all failures into `500` or the legacy `Error()` helper. Examples: invalid caller input belongs in the `400xx` group, state or dependency conflicts belong in `409xx`, and platform or dependency failures belong in `5xx`. These are examples; check the definition files above before choosing or adding a code.
- For new code, return `bizerr` errors and send them with `resputil.HandleError`; keep `backend/internal/resputil/code.go` only for legacy compatibility. Pick an existing `bizerr` group before adding a new one.
- Define new business error codes in `backend/internal/bizerr/groups.go` only when the frontend, CLI, or an external caller needs a stable machine-readable reason. The group must stay aligned with the HTTP status selected by `resputil.HandleError`.
- Error messages returned to clients must be clear, accurate English. Prefer actionable messages that name the invalid field, missing resource, conflicting state, required permission, or unavailable dependency. Do not return generic messages such as `failed`, `invalid`, or raw Go errors when the caller can act on a more specific explanation.
- Keep user-facing messages safe: do not include secrets, tokens, internal IPs, kubeconfigs, SQL fragments, stack traces, or private infrastructure details. Wrap lower-level errors with `bizerr.*.Wrap` so server logs keep the cause while the API response stays safe.
- When adding or changing an API error path, verify how the frontend and CLI will present it. The response envelope remains `code`, `data`, and `msg`; clients may show `msg` directly and expose `http_status` / business `code` for support. If a page needs special behavior, document which business code it handles and why.
- Never concatenate SQL strings in `backend/internal/storage/` or the DAO layer; use parameterized queries.
- Never hardcode secrets, tokens, passwords, internal IPs, kubeconfigs, or production credentials.

## Database Changes

Use GORM + gormigrate + GORM Gen for schema changes.

1. Update model structs in `dao/model/*.go`.
2. Add a migration item in `cmd/gorm-gen/models/migrate.go`.
3. Use migration IDs in `YYYYMMDDHHmm` format.
4. Implement both `Migrate` and `Rollback`.
5. Run `make migrate`.
6. Run `make curd` to regenerate CRUD helpers.
7. Commit the model, migration, generated code, and business code together.

Pulling code that updates `migrate.go` means you should run `make migrate`. A brand-new database is initialized automatically by `make migrate`; Helm deployments run migration automatically through an InitContainer.

Detailed workflow: [`cmd/gorm-gen/README.md`](./cmd/gorm-gen/README.md).

## Storage Server

The storage service is built from `cmd/storage-server/main.go` inside the backend Go module. Use `make run-storage` to start it locally. Use `make build-storage` only when you need to verify or package the storage server binary.

```bash
make run-storage
make build-storage
```

Runtime environment variables:

- `CRATER_STORAGE_PORT` (preferred, fallback `PORT`, default `7320`)
- `CRATER_STORAGE_ROOT` (preferred, fallback `ROOTDIR`, default `/crater`)

For local debug, put these in `backend/.debug.env` and run `make run-storage`. If `make run-storage` fails because required config, filesystem permissions, Kubernetes access, or administrator-provided settings are unavailable, stop and ask the developer to inspect the environment instead of repeatedly retrying.

## Debugging With VSCode

The repo root `.vscode/launch.json` provides a "Backend Debug Server" configuration. Open the project at the root, set breakpoints, press `F5`, and select it. Its `cwd` is `${workspaceFolder}/backend`, `program` points to `backend/cmd/crater/main.go`, and it connects to the cluster via `KUBECONFIG`.

## Before Submitting Backend Changes

- Run the relevant `make` target, usually `make pre-commit-check` for final checks.
- If API behavior changed, confirm `swag` annotations, frontend usage, CLI behavior, and error presentation are aligned.
- If job template payloads changed, confirm the versioning / compatibility decision.
- If database schema changed, include migration and generated code.
- Record exact checks and any developer manual verification for the PR description.
