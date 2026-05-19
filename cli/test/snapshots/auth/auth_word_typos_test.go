package auth_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

// golden: auth_word_typos.{lang}.txtar — auth 下子命令拼写错误（unknown subcommand + 建议）。
const goldenStemWordTypos = "auth_word_typos"

func TestAuthWordTyposEN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "auth", goldenStemWordTypos, "en")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "en")
	bin := snaptest.CraterExecutable(t)

	cases := []snaptest.Case{
		{ID: "01-subcommand-typo-lss-nojson", Args: []string{"auth", "lss", "--no-interactive"}},
		{ID: "02-subcommand-typo-lss-json", Args: []string{"auth", "lss", "--no-interactive", "--json"}},
		{ID: "03-subcommand-typo-swtich-nojson", Args: []string{"auth", "swtich", "--no-interactive"}},
		{ID: "04-subcommand-typo-swtich-json", Args: []string{"auth", "swtich", "--no-interactive", "--json"}},
		{ID: "05-subcommand-typo-loguot-nojson", Args: []string{"auth", "loguot", "--no-interactive"}},
		{ID: "06-subcommand-typo-loguot-json", Args: []string{"auth", "loguot", "--no-interactive", "--json"}},
		{ID: "07-subcommand-typo-rmm-nojson", Args: []string{"auth", "rmm", "--no-interactive"}},
		{ID: "08-subcommand-typo-rmm-json", Args: []string{"auth", "rmm", "--no-interactive", "--json"}},
	}

	results := runSessionCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "en", cases, results, update); err != nil {
		t.Fatal(err)
	}
}

func TestAuthWordTyposZhCN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "auth", goldenStemWordTypos, "zh-CN")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "zh-CN")
	bin := snaptest.CraterExecutable(t)

	cases := []snaptest.Case{
		{ID: "01-subcommand-typo-lss-nojson", Args: []string{"auth", "lss", "--no-interactive"}},
		{ID: "02-subcommand-typo-lss-json", Args: []string{"auth", "lss", "--no-interactive", "--json"}},
		{ID: "03-subcommand-typo-swtich-nojson", Args: []string{"auth", "swtich", "--no-interactive"}},
		{ID: "04-subcommand-typo-swtich-json", Args: []string{"auth", "swtich", "--no-interactive", "--json"}},
		{ID: "05-subcommand-typo-loguot-nojson", Args: []string{"auth", "loguot", "--no-interactive"}},
		{ID: "06-subcommand-typo-loguot-json", Args: []string{"auth", "loguot", "--no-interactive", "--json"}},
		{ID: "07-subcommand-typo-rmm-nojson", Args: []string{"auth", "rmm", "--no-interactive"}},
		{ID: "08-subcommand-typo-rmm-json", Args: []string{"auth", "rmm", "--no-interactive", "--json"}},
	}

	results := runSessionCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "zh-CN", cases, results, update); err != nil {
		t.Fatal(err)
	}
}
