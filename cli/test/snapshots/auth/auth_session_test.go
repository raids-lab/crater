package auth_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

// golden: auth_session.{lang}.txtar — ls / switch / logout / rm（依赖沙箱 fake session）。
const goldenStemSession = "auth_session"

func runSessionCases(t *testing.T, bin string, baseEnv []string, cases []snaptest.Case) []*snaptest.Result {
	t.Helper()
	out := make([]*snaptest.Result, len(cases))
	for i := range cases {
		r, err := snaptest.Run(bin, baseEnv, cases[i].Args)
		if err != nil {
			t.Fatalf("case %s: %v", cases[i].ID, err)
		}
		out[i] = r
	}
	return out
}

func TestAuthSessionSnapshotsEN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "auth", goldenStemSession, "en")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "en")
	bin := snaptest.CraterExecutable(t)

	cases := []snaptest.Case{
		{ID: "01-ls-nojson", Args: []string{"auth", "ls", "--no-interactive"}},
		{ID: "02-ls-json", Args: []string{"auth", "ls", "--no-interactive", "--json"}},
		{ID: "03-ls-filter-platform-nojson", Args: []string{"auth", "ls", "--no-interactive", "--platform", "https://example.invalid"}},
		{ID: "04-ls-filter-platform-json", Args: []string{"auth", "ls", "--no-interactive", "--json", "--platform", "https://example.invalid"}},
		{ID: "05-ls-invalid-mode-nojson", Args: []string{"auth", "ls", "--no-interactive", "--mode", "bad"}},
		{ID: "06-ls-invalid-mode-json", Args: []string{"auth", "ls", "--no-interactive", "--json", "--mode", "bad"}},
		{ID: "07-switch-multiple-matches-nojson", Args: []string{"auth", "switch", "--no-interactive"}},
		{ID: "08-switch-multiple-matches-json", Args: []string{"auth", "switch", "--no-interactive", "--json"}},
		{ID: "09-switch-single-match-nojson", Args: []string{"auth", "switch", "--no-interactive", "--platform", "https://staging.invalid"}},
		{ID: "10-switch-single-match-json", Args: []string{"auth", "switch", "--no-interactive", "--json", "--platform", "https://staging.invalid"}},
		{ID: "11-switch-invalid-mode-nojson", Args: []string{"auth", "switch", "--no-interactive", "--mode", "bad"}},
		{ID: "12-switch-invalid-mode-json", Args: []string{"auth", "switch", "--no-interactive", "--json", "--mode", "bad"}},
		{ID: "13-switch-no-matches-nojson", Args: []string{"auth", "switch", "--no-interactive", "--platform", "https://nope.invalid"}},
		{ID: "14-switch-no-matches-json", Args: []string{"auth", "switch", "--no-interactive", "--json", "--platform", "https://nope.invalid"}},
		{ID: "15-logout-missing-yes-nojson", Args: []string{"auth", "logout", "--no-interactive"}},
		{ID: "16-logout-missing-yes-json", Args: []string{"auth", "logout", "--no-interactive", "--json"}},
		{ID: "17-logout-yes-nojson", Args: []string{"auth", "logout", "--no-interactive", "--yes"}},
		{ID: "18-logout-yes-json", Args: []string{"auth", "logout", "--no-interactive", "--json", "--yes"}},
		{ID: "19-rm-missing-yes-nojson", Args: []string{"auth", "rm", "--no-interactive", "--platform", "https://example.invalid"}},
		{ID: "20-rm-missing-yes-json", Args: []string{"auth", "rm", "--no-interactive", "--json", "--platform", "https://example.invalid"}},
		{ID: "21-rm-invalid-mode-nojson", Args: []string{"auth", "rm", "--no-interactive", "--platform", "https://example.invalid", "--mode", "bad"}},
		{ID: "22-rm-invalid-mode-json", Args: []string{"auth", "rm", "--no-interactive", "--json", "--platform", "https://example.invalid", "--mode", "bad"}},
		{ID: "23-rm-yes-nojson", Args: []string{"auth", "rm", "--no-interactive", "--yes", "--platform", "https://example.invalid"}},
		{ID: "24-rm-yes-json", Args: []string{"auth", "rm", "--no-interactive", "--json", "--yes", "--platform", "https://example.invalid"}},
		{ID: "25-rm-not-found-nojson", Args: []string{"auth", "rm", "--no-interactive", "--yes", "--platform", "https://nope.invalid"}},
		{ID: "26-rm-not-found-json", Args: []string{"auth", "rm", "--no-interactive", "--json", "--yes", "--platform", "https://nope.invalid"}},
	}

	results := runSessionCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "en", cases, results, update); err != nil {
		t.Fatal(err)
	}
}

func TestAuthSessionSnapshotsZhCN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "auth", goldenStemSession, "zh-CN")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "zh-CN")
	bin := snaptest.CraterExecutable(t)

	cases := []snaptest.Case{
		{ID: "01-ls-nojson", Args: []string{"auth", "ls", "--no-interactive"}},
		{ID: "02-ls-json", Args: []string{"auth", "ls", "--no-interactive", "--json"}},
		{ID: "03-ls-filter-platform-nojson", Args: []string{"auth", "ls", "--no-interactive", "--platform", "https://example.invalid"}},
		{ID: "04-ls-filter-platform-json", Args: []string{"auth", "ls", "--no-interactive", "--json", "--platform", "https://example.invalid"}},
		{ID: "05-ls-invalid-mode-nojson", Args: []string{"auth", "ls", "--no-interactive", "--mode", "bad"}},
		{ID: "06-ls-invalid-mode-json", Args: []string{"auth", "ls", "--no-interactive", "--json", "--mode", "bad"}},
		{ID: "07-switch-multiple-matches-nojson", Args: []string{"auth", "switch", "--no-interactive"}},
		{ID: "08-switch-multiple-matches-json", Args: []string{"auth", "switch", "--no-interactive", "--json"}},
		{ID: "09-switch-single-match-nojson", Args: []string{"auth", "switch", "--no-interactive", "--platform", "https://staging.invalid"}},
		{ID: "10-switch-single-match-json", Args: []string{"auth", "switch", "--no-interactive", "--json", "--platform", "https://staging.invalid"}},
		{ID: "11-switch-invalid-mode-nojson", Args: []string{"auth", "switch", "--no-interactive", "--mode", "bad"}},
		{ID: "12-switch-invalid-mode-json", Args: []string{"auth", "switch", "--no-interactive", "--json", "--mode", "bad"}},
		{ID: "13-switch-no-matches-nojson", Args: []string{"auth", "switch", "--no-interactive", "--platform", "https://nope.invalid"}},
		{ID: "14-switch-no-matches-json", Args: []string{"auth", "switch", "--no-interactive", "--json", "--platform", "https://nope.invalid"}},
		{ID: "15-logout-missing-yes-nojson", Args: []string{"auth", "logout", "--no-interactive"}},
		{ID: "16-logout-missing-yes-json", Args: []string{"auth", "logout", "--no-interactive", "--json"}},
		{ID: "17-logout-yes-nojson", Args: []string{"auth", "logout", "--no-interactive", "--yes"}},
		{ID: "18-logout-yes-json", Args: []string{"auth", "logout", "--no-interactive", "--json", "--yes"}},
		{ID: "19-rm-missing-yes-nojson", Args: []string{"auth", "rm", "--no-interactive", "--platform", "https://example.invalid"}},
		{ID: "20-rm-missing-yes-json", Args: []string{"auth", "rm", "--no-interactive", "--json", "--platform", "https://example.invalid"}},
		{ID: "21-rm-invalid-mode-nojson", Args: []string{"auth", "rm", "--no-interactive", "--platform", "https://example.invalid", "--mode", "bad"}},
		{ID: "22-rm-invalid-mode-json", Args: []string{"auth", "rm", "--no-interactive", "--json", "--platform", "https://example.invalid", "--mode", "bad"}},
		{ID: "23-rm-yes-nojson", Args: []string{"auth", "rm", "--no-interactive", "--yes", "--platform", "https://example.invalid"}},
		{ID: "24-rm-yes-json", Args: []string{"auth", "rm", "--no-interactive", "--json", "--yes", "--platform", "https://example.invalid"}},
		{ID: "25-rm-not-found-nojson", Args: []string{"auth", "rm", "--no-interactive", "--yes", "--platform", "https://nope.invalid"}},
		{ID: "26-rm-not-found-json", Args: []string{"auth", "rm", "--no-interactive", "--json", "--yes", "--platform", "https://nope.invalid"}},
	}

	results := runSessionCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "zh-CN", cases, results, update); err != nil {
		t.Fatal(err)
	}
}
