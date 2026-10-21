# Crater-Agent Feature Matrix

| Capability                 | Entry                    | Backend API                                | Agent behavior                                                                            |
| -------------------------- | ------------------------ | ------------------------------------------ | ----------------------------------------------------------------------------------------- |
| Platform LLM configuration | Admin platform settings  | Existing `/api/v1/admin/system-config/llm` | Used when no personal configuration exists                                                |
| Personal LLM configuration | Portal personal settings | New `/api/v1/agent/llm-config`             | Overrides platform configuration for the same user ID                                     |
| Connection test            | Personal LLM card        | New `/api/v1/agent/llm-config/test`        | Validates endpoint, key, and model before persistence                                     |
| Basic conversation         | Global floating panel    | New `/api/v1/agent/chat`                   | Streams through AI SDK with explicit Skill and tool boundaries                            |
| Cancel generation          | Chat composer            | Request cancellation                       | Propagates the browser abort signal through Go and the Agent runtime                      |
| Platform guide             | Agent conversation       | No additional API                          | Selects the `platform-guide` Skill for onboarding and stable user-side platform knowledge |
| Scope enforcement          | Agent conversation       | No additional API                          | Rejects prompt overrides deterministically and constrains other requests to the Crater scope |
| Job discovery              | Agent conversation       | Internal Agent job-list API                | Lists only jobs authorized for the current user or administrator identity                 |
| Job diagnosis              | Job detail or conversation | Internal diagnostic APIs                 | Reads bounded events, termination states, metrics, and scheduling evidence                |
| Cluster capacity           | Agent conversation       | Internal node and capacity APIs            | Reports current read-only node and aggregate resource snapshots                           |
| Form assistance            | Supported creation forms | Browser page context                       | Proposes allowlisted field patches and applies them only after user confirmation           |
| Page navigation            | Agent conversation       | Browser navigation                         | Proposes identity-specific allowlisted destinations and opens them only after confirmation |

## Active prompts

The `platform-guide` Skill combines the invariant safety boundary with curated,
version-controlled user knowledge based on the official Crater user guide and
current portal routes. It is selected deterministically for onboarding and
platform-related questions. Every Skill declares an explicit capability contract
and tool allowlist. Prompt-override requests receive a fixed response from the
Agent service. Other requests remain constrained by the platform capability
contract. Read tools may report only the live facts returned by authorized
backend APIs; unsupported state and write operations must be described as
unavailable.
