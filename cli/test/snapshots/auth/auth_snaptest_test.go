package auth_test

import (
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

// runAuthCases runs subprocesses; for login HTTP sim cases, appends CRATER_TEST_SANDBOX_HTTP.
func runAuthCases(t *testing.T, bin string, baseEnv []string, cases []snaptest.Case) []*snaptest.Result {
	t.Helper()
	out := make([]*snaptest.Result, len(cases))
	for i := range cases {
		env := appendHTTPEnvForLogin(baseEnv, cases[i].ID)
		r, err := snaptest.Run(bin, env, cases[i].Args)
		if err != nil {
			t.Fatalf("case %s: %v", cases[i].ID, err)
		}
		out[i] = r
	}
	return out
}

func appendHTTPEnvForLogin(env []string, caseID string) []string {
	switch caseID {
	case "09-http-sim-timeout-nojson", "10-http-sim-timeout-json":
		return append(env, "CRATER_TEST_SANDBOX_HTTP=timeout")
	case "11-http-sim-404-nojson", "12-http-sim-404-json":
		return append(env, "CRATER_TEST_SANDBOX_HTTP=error404")
	default:
		return env
	}
}
