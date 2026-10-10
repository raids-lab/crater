# Crater-Agent

Crater-Agent is the AI SDK runtime for Crater. The Go backend authenticates
users and retains all model credentials; this process receives only a short-lived
context token and streams AI SDK UI messages back through the backend.

## Local development

Use the same random internal token for the backend and this process:

```bash
export CRATER_AGENT_INTERNAL_TOKEN="$(openssl rand -hex 32)"
export CRATER_BACKEND_INTERNAL_URL="http://localhost:8088"
pnpm install
pnpm dev
```

Set `CRATER_AGENT_URL=http://localhost:8000` and the same
`CRATER_AGENT_INTERNAL_TOKEN` in `backend/.debug.env` before starting the Go
backend.

## Kubernetes deployment

The Crater Helm chart deploys this runtime as an internal-only `ClusterIP`
service. The browser communicates with the Go backend, and the backend forwards
authenticated requests to Crater-Agent. Model credentials remain encrypted in
the Crater database and are never mounted into this container.

For a local end-to-end deployment with Docker Desktop, Kind, and Helm, run from
the repository root:

```bash
make agent-kind-up
```

The local environment is isolated in the `kind-crater-agent-dev` context and is
available at `http://crater.localhost:8080`. Run `make agent-kind-smoke` to repeat
the service checks and `make agent-kind-down` to delete the local cluster.

The Kind environment installs the Volcano API definitions required by the
backend, but not the Volcano scheduler and controllers. It validates the Agent,
authentication, database, frontend, and backend deployment path; use the shared
development cluster for workload scheduling tests.
