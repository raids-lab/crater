package auth_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

// golden: auth_login.{lang}.txtar
const goldenStemLogin = "auth_login"

func TestAuthLoginSnapshotsEN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "auth", goldenStemLogin, "en")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "en")
	bin := snaptest.CraterExecutable(t)

	cases := []snaptest.Case{
		{ID: "01-missing-platform-nojson", Args: []string{"auth", "login", "--no-interactive", "--username", "u", "--password", "p"}},
		{ID: "02-missing-platform-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--username", "u", "--password", "p"}},
		{ID: "03-missing-username-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--password", "p"}},
		{ID: "04-missing-username-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--password", "p"}},
		{ID: "05-missing-password-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--username", "u"}},
		{ID: "06-missing-password-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--username", "u"}},
		{ID: "07-invalid-mode-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "bad"}},
		{ID: "08-invalid-mode-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "bad"}},
		{ID: "09-http-sim-timeout-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "ldap"}},
		{ID: "10-http-sim-timeout-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "ldap"}},
		{ID: "11-http-sim-404-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "ldap"}},
		{ID: "12-http-sim-404-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "ldap"}},
		{ID: "13-subcommand-typo-logni-nojson", Args: []string{"auth", "logni", "--no-interactive"}},
		{ID: "14-subcommand-typo-logni-json", Args: []string{"auth", "logni", "--no-interactive", "--json"}},
		{ID: "15-missing-user-pass-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example"}},
		{ID: "16-missing-user-pass-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example"}},
		{ID: "17-invalid-mode-missing-pass-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--username", "u", "--mode", "bad"}},
		{ID: "18-invalid-mode-missing-pass-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--username", "u", "--mode", "bad"}},
	}

	results := runAuthCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "en", cases, results, update); err != nil {
		t.Fatal(err)
	}
}

func TestAuthLoginSnapshotsZhCN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "auth", goldenStemLogin, "zh-CN")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "zh-CN")
	bin := snaptest.CraterExecutable(t)

	cases := []snaptest.Case{
		{ID: "01-missing-platform-nojson", Args: []string{"auth", "login", "--no-interactive", "--username", "u", "--password", "p"}},
		{ID: "02-missing-platform-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--username", "u", "--password", "p"}},
		{ID: "03-missing-username-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--password", "p"}},
		{ID: "04-missing-username-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--password", "p"}},
		{ID: "05-missing-password-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--username", "u"}},
		{ID: "06-missing-password-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--username", "u"}},
		{ID: "07-invalid-mode-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "bad"}},
		{ID: "08-invalid-mode-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "bad"}},
		{ID: "09-http-sim-timeout-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "ldap"}},
		{ID: "10-http-sim-timeout-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "ldap"}},
		{ID: "11-http-sim-404-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "ldap"}},
		{ID: "12-http-sim-404-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--username", "u", "--password", "p", "--mode", "ldap"}},
		{ID: "13-subcommand-typo-logni-nojson", Args: []string{"auth", "logni", "--no-interactive"}},
		{ID: "14-subcommand-typo-logni-json", Args: []string{"auth", "logni", "--no-interactive", "--json"}},
		{ID: "15-missing-user-pass-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example"}},
		{ID: "16-missing-user-pass-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example"}},
		{ID: "17-invalid-mode-missing-pass-nojson", Args: []string{"auth", "login", "--no-interactive", "--platform", "http://example", "--username", "u", "--mode", "bad"}},
		{ID: "18-invalid-mode-missing-pass-json", Args: []string{"auth", "login", "--no-interactive", "--json", "--platform", "http://example", "--username", "u", "--mode", "bad"}},
	}

	results := runAuthCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "zh-CN", cases, results, update); err != nil {
		t.Fatal(err)
	}
}
