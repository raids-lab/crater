package session

import "github.com/raids-lab/crater/cli/internal/state"

// Thin re-exports to keep cmd depending only on internal/session.
// The underlying persisted shape remains the same (internal/state).
type (
	State         = state.State
	AuthInfo      = state.AuthInfo
	ActiveContext = state.ActiveContext
)

