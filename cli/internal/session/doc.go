// Package session centralizes access to local session state and credentials.
//
// Terminology (important):
//
// - state: The contents of the CLI local state file (state.json), managed by internal/state.
//   It contains AuthInfos, ActiveContext, and Language.
//
// - active context: A triple (platform_url, username, method) identifying the currently
//   active saved credentials. In code this is state.ActiveContext.
//
// - token: The access token returned by the platform login API. It is stored in the OS
//   keyring via internal/credential (not in state.json).
//
// - context.Context: Go's request-scoped context for cancellation/deadlines. This is not
//   the same as "active context" above.
//
// The cmd layer should depend on this package rather than directly touching
// HOME-derived paths or the OS keyring.
package session

