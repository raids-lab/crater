# Crater-Agent

Crater-Agent is the AI SDK runtime for Crater. The Go backend authenticates
users and retains all model credentials; this process receives only a short-lived
context token and streams AI SDK UI messages back through the backend.

## Local development

Use the same random service token for the backend and this process. The context
signing and encryption keys belong only to the backend:

```bash
export CRATER_AGENT_INTERNAL_TOKEN="$(openssl rand -hex 32)"
export CRATER_AGENT_CONTEXT_SIGNING_KEY="$(openssl rand -hex 32)"
export SYSTEM_SECRET_KEY="$(openssl rand -hex 16)"
export CRATER_BACKEND_INTERNAL_URL="http://localhost:8088"
pnpm install
pnpm dev
```

Set `CRATER_AGENT_URL=http://localhost:8000` and all three security variables in
`backend/.debug.env` before starting the Go backend. Only
`CRATER_AGENT_INTERNAL_TOKEN` is provided to the Agent process.

## Kubernetes deployment

The Crater Helm chart deploys this runtime as an internal-only `ClusterIP`
service. The browser communicates with the Go backend, and the backend forwards
authenticated requests to Crater-Agent. Model credentials remain encrypted in
the Crater database and are never mounted into this container.

For production, prefer `agent.existingSecretName`. The referenced Secret must
contain `internal-token`, `context-signing-key`, and a 32-byte `encryption-key`.
When rotating the encryption key, retain the old key in the optional
`previous-encryption-keys` entry as a comma-separated value, update the Secret,
and change `agent.secretRevision` in the same Helm release. This rolls both
Deployments and lets the backend re-encrypt stored model credentials at startup.
After the rollout succeeds, remove the previous key and change
`agent.secretRevision` again.

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
