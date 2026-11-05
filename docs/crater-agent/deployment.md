# Crater-Agent Deployment

Crater-Agent is an internal service deployed together with Crater. It is not
exposed through an Ingress. Browser requests are authenticated by the Go
backend, forwarded to the Agent through its ClusterIP service, and model
requests return to the backend LLM proxy. Model API keys therefore remain in
the backend and are never injected into the Agent container.

## Helm deployment

The Crater chart keeps the Agent disabled by default. Enabling it requires an
externally managed Secret so credential generation, backup, and rotation remain
owned by the deployment environment:

```yaml
agent:
  enabled: true
  existingSecretName: crater-agent-internal
```

The referenced Secret must contain:

- `internal-token`: at least 32 characters, shared by backend and Agent for
  service authentication.
- `context-signing-key`: at least 32 characters, injected only into the backend.
- `encryption-key`: exactly 32 characters, injected only into the backend for
  personal model API keys.
- `previous-encryption-keys`: optional comma-separated previous encryption keys.

All three active keys must be distinct. The chart never generates these values.
Create and back up the Secret before enabling the Agent. After changing Secret
contents, also change `agent.secretRevision` so both Deployments roll together.
When rotating `encryption-key`, keep the old value in
`previous-encryption-keys` until every affected personal model configuration has
been saved again with the new key.

Every Helm install or upgrade runs the backend image's `./migrate` binary in an
init container before starting the backend. This automatically creates or
updates `user_llm_configs`, `agent_conversations`, `agent_runs`, and
`agent_messages` in the configured PostgreSQL database. Migrations are
idempotent and deployment stops instead of starting against an incompatible
schema if a migration fails.

Pull requests validate the Agent source, runtime image, and Helm templates.
After a change is merged, GitHub Actions publishes the multi-architecture Agent
image and the versioned Helm chart to GHCR. The repository intentionally does
not hold production cluster credentials; a GitOps controller or environment
deployment workflow must apply the published chart to a production cluster.

## Production-style local run with an external cluster

This mode builds the same Backend, Frontend, and Agent images used in
deployment, then starts them with Docker Compose. It mounts an existing backend
configuration and a flattened kubeconfig at runtime. No credentials are copied
into an image or committed to the repository.

```bash
KUBECONFIG=/path/to/kubeconfig \
CRATER_AGENT_ENCRYPTION_KEY=<the-32-character-key-used-by-the-target-database> \
CRATER_LOCAL_ALLOW_EXTERNAL_CLUSTER=true \
CRATER_LOCAL_RUN_MIGRATIONS=true \
make agent-local-up
```

The UI is available at `http://localhost:5180`. The backend continues to use the
PostgreSQL, Prometheus, registry, storage, and Kubernetes endpoints from
`backend/etc/debug-config.yaml`. Override that file when needed:

```bash
KUBECONFIG=/path/to/kubeconfig \
CRATER_AGENT_ENCRYPTION_KEY=<the-32-character-key-used-by-the-target-database> \
CRATER_LOCAL_BACKEND_CONFIG=/path/to/debug-config.yaml \
CRATER_LOCAL_ALLOW_EXTERNAL_CLUSTER=true \
make agent-local-up
```

Use `make agent-local-smoke` to repeat health checks and
`make agent-local-down` to stop the containers. Database migrations are skipped
unless `CRATER_LOCAL_RUN_MIGRATIONS=true` is explicitly set. The migration uses
the same idempotent binary as the Helm init container and writes schema changes
to the PostgreSQL instance from the selected backend configuration.

The Crater backend also runs Kubernetes reconcilers. Do not run this mode in
parallel with another development backend connected to the same cluster. The
explicit `CRATER_LOCAL_ALLOW_EXTERNAL_CLUSTER=true` acknowledgement prevents an
accidental start against a real cluster, but it does not make the backend
read-only.

If a development server already occupies the default ports, select different
host ports without changing the container topology:

```bash
KUBECONFIG=/path/to/kubeconfig \
CRATER_AGENT_ENCRYPTION_KEY=<the-32-character-key-used-by-the-target-database> \
CRATER_LOCAL_HTTP_PORT=5181 \
CRATER_LOCAL_BACKEND_PORT=8099 \
CRATER_LOCAL_ALLOW_EXTERNAL_CLUSTER=true \
make agent-local-up
```

## Isolated Helm verification

The Kind mode creates a disposable Kubernetes cluster, builds local images,
installs the Crater chart, runs migrations against a disposable PostgreSQL
instance, and verifies the three application Deployments:

```bash
make agent-kind-up
make agent-kind-smoke
make agent-kind-down
```

Kind verification does not access the external Crater cluster or database.
