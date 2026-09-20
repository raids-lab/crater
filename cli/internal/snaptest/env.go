package snaptest

import (
	"os"
	"strings"

	"github.com/raids-lab/crater/cli/internal/testutil"
)

// EnvMinimal returns a snapshot-friendly environment: isolated user config
// paths, fixed language-related locale, and PATH inherited when set so the
// real binary can run.
func EnvMinimal(home, craterLang string) []string {
	out := append(testutil.UserConfigEnv(home),
		"CRATER_LANG="+craterLang,
		"CRATER_TEST_SANDBOX=1",
		"LANG=C",
		"LC_ALL=C",
		"TERM=dumb",
	)
	if p := os.Getenv("PATH"); p != "" {
		out = append(out, "PATH="+p)
	}
	return out
}

// ArgvLine is a stable, human-readable argv line for golden txtar (first token is always "crater").
func ArgvLine(args []string) string {
	return "crater " + strings.Join(args, " ")
}
