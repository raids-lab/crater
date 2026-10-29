# Foundation Feature Matrix

| Capability                 | Entry                    | Backend API                                | Agent behavior                                                                            |
| -------------------------- | ------------------------ | ------------------------------------------ | ----------------------------------------------------------------------------------------- |
| Platform LLM configuration | Admin platform settings  | Existing `/api/v1/admin/system-config/llm` | Used when no personal configuration exists                                                |
| Personal LLM configuration | Portal personal settings | New `/api/v1/agent/llm-config`             | Overrides platform configuration for the same user ID                                     |
| Connection test            | Personal LLM card        | New `/api/v1/agent/llm-config/test`        | Validates endpoint, key, and model before persistence                                     |
| Basic conversation         | Global floating panel    | New `/api/v1/agent/chat`                   | Streams through AI SDK with no business tools                                             |
| Cancel generation          | Chat composer            | Request cancellation                       | Propagates the browser abort signal through Go and the Agent runtime                      |
| Platform guide             | Agent conversation       | No additional API                          | Selects the `platform-guide` Skill for onboarding and stable user-side platform knowledge |
| Scope enforcement          | Agent conversation       | No additional API                          | Uses invariant model instructions without language-specific keyword routing               |

## Active prompts

The `platform-guide` Skill combines the invariant safety boundary with curated,
version-controlled user knowledge based on the official Crater user guide and
current portal routes. It is selected for every foundation-stage conversation.
Every Skill declares an explicit capability contract and tool allowlist. The
model answers in the user's current language and briefly declines unrelated
requests under the invariant scope instructions. The current platform Skill has
no business tools and must not claim real-time platform access.
