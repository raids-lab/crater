// Package session centralizes access to local session state and credentials.
//
// Terminology (important):
//
//   - state: The contents of the CLI local state file (state.json), managed by internal/state.
//     It contains AuthInfos (including both access and refresh tokens), ActiveContext, and Language.
//
//   - active context: A triple (platform_url, username, method) identifying the currently
//     active saved credentials. In code this is state.ActiveContext.
//
//   - token and refresh_token: Credentials returned by the platform auth API, stored on the
//     matching AuthInfo in state.json. Command output must strip both fields.
//
//   - context.Context: Go's request-scoped context for cancellation/deadlines. This is not
//     the same as "active context" above.
//
// The cmd layer should depend on this package rather than directly touching
// HOME-derived paths or state.json.
package session
