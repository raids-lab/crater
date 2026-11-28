# Agent Feature Matrix

| Capability                 | Entry                    | Backend API                                | Agent behavior                                                                            |
| -------------------------- | ------------------------ | ------------------------------------------ | ----------------------------------------------------------------------------------------- |
| Platform LLM configuration | Admin platform settings  | Existing `/api/v1/admin/system-config/llm` | Used when no personal configuration exists                                                |
| Personal LLM configuration | Portal personal settings | New `/api/v1/agent/llm-config`             | Overrides platform configuration for the same user ID                                     |
| Connection test            | Personal LLM card        | New `/api/v1/agent/llm-config/test`        | Validates endpoint, key, and model before persistence                                     |
| Conversation and tools     | Global panel or page entry | `/api/v1/agent/chat`                     | Streams answers with role-scoped read tools and confirmation cards                       |
| Cancel generation          | Chat composer            | Request cancellation and run cancellation | Stops the stream and bounded tool queries, releases the durable run                       |
| Platform guide             | Agent conversation       | No additional API                          | Selects the `platform-guide` Skill for onboarding and stable user-side platform knowledge |
| Scope enforcement          | Agent conversation       | No additional API                          | Uses invariant model instructions without language-specific keyword routing               |
| Job and node reads         | Any Agent entry          | Internal `/api/agent/jobs`, `/nodes`, `/capacity` | Authorized job list, diagnostics, scheduling estimates and cluster capacity        |
| Browser actions            | Confirmation cards       | `/api/v1/agent/conversations/{id}/tool-output` | Role-scoped navigation and allowlisted form patches; no form submission             |
| Administrative audit       | Admin Agent page         | `/api/v1/admin/agent/subjects`             | User/identity groups, conversations, messages and read-only tool records                    |

## Active prompts

The `platform-guide` Skill combines the invariant safety boundary with curated,
version-controlled user knowledge based on the official Crater user guide and
current routes. All entry points share the same role-scoped read capabilities.
Every Skill declares an explicit capability contract and tool allowlist. The
model answers in the user's current language and briefly declines unrelated
requests under the invariant scope instructions. Live claims require successful
tool evidence. The model cannot read raw logs or mutate jobs. Form assistance
adds a typed proposal tool only for a registered form, bound to that form instance.
After confirmed navigation to a registered form, the panel stays open and
continues the same conversation with bounded GPU and image options loaded by
that form. A proposed option is checked against the live form before applying.
Job lists fetch the requested page without a full count unless the user asks for
an exact total; the response always indicates whether another page exists.
Prompts and knowledge remain version-controlled files; editable runtime prompt
configuration is not part of this migration.
