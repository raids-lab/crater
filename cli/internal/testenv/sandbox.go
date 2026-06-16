package testenv

import (
	"os"
	"strings"
)

// SandboxEnabled returns true when the CLI should avoid touching developer environment:
// - local state storage on disk
// - OS keyring
// - real network (typically used together with HTTP simulation)
//
// It is controlled by CRATER_TEST_SANDBOX and/or feature-specific flags.
func SandboxEnabled() bool {
	return truthyEnv(os.Getenv("CRATER_TEST_SANDBOX"))
}

func SandboxHTTPMode() string {
	// When sandboxing HTTP, set CRATER_TEST_SANDBOX_HTTP:
	// - "1"/"true"/... => default "timeout"
	// - "timeout"|"hang"|"error404"|... => explicit mode (case-insensitive)
	raw := strings.TrimSpace(os.Getenv("CRATER_TEST_SANDBOX_HTTP"))
	if truthyEnv(raw) {
		return "timeout"
	}
	// If sandbox is enabled but no explicit/valid mode is given, fallback to timeout.
	if SandboxEnabled() && raw == "" {
		return "timeout"
	}

	mode := strings.ToLower(raw)
	switch mode {
	case "", "timeout", "hang", "error404", "404":
		// ok ("" already handled above)
	default:
		if SandboxEnabled() {
			return "timeout"
		}
	}
	return mode
}

func SandboxSessionEnabled() bool {
	// Session sandbox can be enabled standalone or under the global sandbox.
	return SandboxEnabled() || truthyEnv(os.Getenv("CRATER_TEST_SANDBOX_SESSION"))
}

func SandboxSessionEmpty() bool {
	return truthyEnv(os.Getenv("CRATER_TEST_SANDBOX_SESSION_EMPTY"))
}

func truthyEnv(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
