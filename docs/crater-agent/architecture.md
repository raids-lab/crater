# Crater-Agent Foundation

## Scope

This milestone provides authenticated, streaming model conversation and two-level
LLM credentials. It intentionally contains no Crater business tools.

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

- `agent_conversations` stores owner-scoped titles. User deletion permanently
  removes the conversation, its messages, and its run records.
- `agent_runs` stores one execution boundary per user turn, including terminal
  status and errors.
- `agent_messages` stores ordered, versioned AI SDK UI message parts as JSONB.

Structured assistant parts are preserved instead of being flattened to text.
User input currently accepts text parts only; URL-backed file parts are rejected
at both the backend and Agent runtime boundaries. Future trusted part types can
be added explicitly without replacing the conversation schema. Editing an
earlier user message marks the later branch as superseded rather than destroying
its audit history.

## Configuration precedence

1. A complete personal configuration (`baseUrl`, `modelName`, and `apiKey`) is
   selected for that user ID on both portal and admin surfaces.
2. Otherwise, the existing platform configuration is selected.
3. If neither configuration is complete, chat fails immediately with a service
   configuration error.

Personal endpoints must be public HTTPS addresses. Private, loopback, link-local,
and unresolved destinations are rejected to prevent server-side network access.

## Extension boundary

The current runtime always selects the `platform-guide` Skill. It combines
invariant safety instructions with curated, version-controlled user
documentation and asks the model to answer in the user's language. Scope and
prompt-injection handling are instruction based rather than keyword based, so
valid multilingual platform questions are not rejected by a brittle regular
expression. Business capabilities are added through the typed tool registry in
`crater-agent/src/tools/registry.ts`. Authorization and data ownership remain
enforced by the Go backend.

## Deployment

The Helm chart deploys Crater-Agent as an internal `ClusterIP` service. It has no
Ingress. The backend and Agent share an internal service token, while only the
backend receives the separate context-signing key and credential-encryption key.
Agent callbacks exposed through the backend require both the internal token and
a short-lived signed user context. The backend also applies per-user model proxy
rate and concurrency limits. An Agent NetworkPolicy limits ingress to the
backend and egress to the backend and cluster DNS.

The existing backend migration init container creates and upgrades the Agent
tables before the backend starts. The same migration binary is used by local
production-style deployment and Helm deployment, so no manual SQL step is
required.

Pull requests validate TypeScript and the container image; chart changes are
validated by the repository's Helm chart workflow.
After a change is merged, GitHub Actions publishes the multi-architecture Agent
image and the Crater Helm chart to GHCR. The repository deliberately does not
hold production cluster credentials. Production environments should consume the
published chart through a pull-based GitOps controller such as Argo CD and pin
images by immutable version or commit digest.

The local Kind deployment under `hack/kind` exercises the same container and
Helm templates used in production. It creates a dedicated Kubernetes context so
that local smoke tests cannot accidentally target an existing cluster.
