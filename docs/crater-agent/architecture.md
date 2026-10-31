# Crater-Agent Foundation

## Scope

This milestone provides authenticated streaming conversations, two-level LLM
credentials, page-aware form assistance, and bounded read-only Crater tools.

## Request path

```text
React useChat
  -> Go POST /api/v1/agent/chat (JWT authentication)
  -> PostgreSQL saves the user message and creates an Agent run
  -> Crater-Agent /chat (AI SDK ToolLoopAgent)
  -> Go /api/agent/llm/chat/completions (short-lived signed context)
  -> selected OpenAI-compatible endpoint
  -> Crater-Agent reports the structured assistant message to Go
  -> PostgreSQL completes the run and saves the assistant message
```

The Go backend is the only component that decrypts an API key. The browser and
the TypeScript Agent runtime never receive model credentials.

## Durable conversation state

PostgreSQL is the single source of truth for conversation history. The browser
only keeps display preferences and the active conversation for the lifetime of
the current page; it does not persist messages in local storage.

- `agent_conversations` stores owner-scoped titles and soft deletion state.
- `agent_runs` stores one execution boundary per user turn, including terminal
  status and errors.
- `agent_messages` stores ordered, versioned AI SDK UI message parts as JSONB.

Structured parts are preserved instead of being flattened to text. Future tool
calls, confirmations, tables, files, and artifacts can therefore be added as
new part types without replacing the conversation schema. Editing an earlier
user message marks the later branch as superseded rather than destroying its
audit history.

## Configuration precedence

1. A complete personal configuration (`baseUrl`, `modelName`, and `apiKey`) is
   selected for that user ID on both portal and admin surfaces.
2. Otherwise, the existing platform configuration is selected.
3. If neither configuration is complete, chat fails immediately with a service
   configuration error.

Personal endpoints must be public HTTPS addresses. Private, loopback, link-local,
and unresolved destinations are rejected to prevent server-side network access.

## Extension boundary

The runtime selects an explicit Skill from the user conversation. Every Skill
declares its capabilities, limitations, and active tools; that contract is added
to model instructions for every request. The platform and diagnosis Skills can
query authorized job summaries, bounded diagnostic snapshots, node capacity,
and aggregate cluster capacity through the Go backend. Navigation and form
patch tools only create proposals; the browser executes them after explicit
user confirmation. Business capabilities are registered in the typed tool
registry in `crater-agent/src/tools/registry.ts`, and only tools selected by the
active Skill are exposed. Authorization and data ownership remain enforced by
the Go backend, which keeps database and Kubernetes credentials out of the Agent
runtime.

## Deployment

The Helm chart deploys Crater-Agent as an internal `ClusterIP` service. It has no
Ingress and cannot be called directly from outside the cluster. The backend and
Agent receive the same internal token from a Kubernetes Secret for service
request authentication. Only the backend receives the separate context-signing
key, which it uses to sign short-lived user context tokens.

The existing backend migration init container creates and upgrades the Agent
tables before the backend starts. The same migration binary is used by local
production-style deployment and Helm deployment, so no manual SQL step is
required.

Pull requests validate TypeScript, the container image, and Helm templates.
After a change is merged, GitHub Actions publishes the multi-architecture Agent
image and the Crater Helm chart to GHCR. The repository deliberately does not
hold production cluster credentials. Production environments should consume the
published chart through a pull-based GitOps controller such as Argo CD and pin
images by immutable version or commit digest.

The local Kind deployment under `hack/kind` exercises the same container and
Helm templates used in production. It creates a dedicated Kubernetes context so
that local smoke tests cannot accidentally target an existing cluster.
