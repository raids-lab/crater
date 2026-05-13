package config_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

// goldenStem: language_test.go -> language.{lang}.txtar
const goldenStem = "language"

// 快照默认经 snaptest.EnvMinimal 启用 CRATER_TEST_SANDBOX=1（存储隔离）。
// 01–14：错误与边界（用法、非法值、未知子命令等）。
// 15–16：成功路径（沙箱下不落盘，输出稳定）。

func TestConfigLanguageSnapshotsEN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "config", goldenStem, "en")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "en")
	bin := snaptest.CraterExecutable(t)

	cases := []snaptest.Case{
		{ID: "01-missing-language-arg-nojson", Args: []string{"config", "language", "--no-interactive"}},
		{ID: "02-missing-language-arg-json", Args: []string{"config", "language", "--no-interactive", "--json"}},
		{ID: "03-invalid-lang-nojson", Args: []string{"config", "language", "--no-interactive", "not-a-lang"}},
		{ID: "04-invalid-lang-json", Args: []string{"config", "language", "--no-interactive", "--json", "not-a-lang"}},
		{ID: "05-lang-typo-nojson", Args: []string{"config", "language", "--no-interactive", "englush"}},
		{ID: "06-lang-typo-json", Args: []string{"config", "language", "--no-interactive", "--json", "englush"}},
		{ID: "07-no-interactive-flag-typo-nojson", Args: []string{"config", "language", "--no-interactiv"}},
		{ID: "08-no-interactive-flag-typo-json", Args: []string{"config", "language", "--no-interactiv", "--json"}},
		{ID: "09-no-interactive-bool-invalid-nojson", Args: []string{"config", "language", "--no-interactive=maybe"}},
		{ID: "10-no-interactive-bool-invalid-json", Args: []string{"config", "language", "--no-interactive=maybe", "--json"}},
		{ID: "11-subcommand-typo-langauge-nojson", Args: []string{"config", "langauge", "--no-interactive"}},
		{ID: "12-subcommand-typo-langauge-json", Args: []string{"config", "langauge", "--no-interactive", "--json"}},
		{ID: "13-too-many-args-nojson", Args: []string{"config", "language", "--no-interactive", "en", "extra"}},
		{ID: "14-too-many-args-json", Args: []string{"config", "language", "--no-interactive", "--json", "en", "extra"}},
		{ID: "15-set-language-success-nojson", Args: []string{"config", "language", "--no-interactive", "en"}},
		{ID: "16-set-language-success-json", Args: []string{"config", "language", "--no-interactive", "--json", "en"}},
	}

	results := runConfigLanguageCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "en", cases, results, update); err != nil {
		t.Fatal(err)
	}
}

func TestConfigLanguageSnapshotsZhCN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "config", goldenStem, "zh-CN")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "zh-CN")
	bin := snaptest.CraterExecutable(t)

	cases := []snaptest.Case{
		{ID: "01-missing-language-arg-nojson", Args: []string{"config", "language", "--no-interactive"}},
		{ID: "02-missing-language-arg-json", Args: []string{"config", "language", "--no-interactive", "--json"}},
		{ID: "03-invalid-lang-nojson", Args: []string{"config", "language", "--no-interactive", "not-a-lang"}},
		{ID: "04-invalid-lang-json", Args: []string{"config", "language", "--no-interactive", "--json", "not-a-lang"}},
		{ID: "05-lang-typo-nojson", Args: []string{"config", "language", "--no-interactive", "englush"}},
		{ID: "06-lang-typo-json", Args: []string{"config", "language", "--no-interactive", "--json", "englush"}},
		{ID: "07-no-interactive-flag-typo-nojson", Args: []string{"config", "language", "--no-interactiv"}},
		{ID: "08-no-interactive-flag-typo-json", Args: []string{"config", "language", "--no-interactiv", "--json"}},
		{ID: "09-no-interactive-bool-invalid-nojson", Args: []string{"config", "language", "--no-interactive=maybe"}},
		{ID: "10-no-interactive-bool-invalid-json", Args: []string{"config", "language", "--no-interactive=maybe", "--json"}},
		{ID: "11-subcommand-typo-langauge-nojson", Args: []string{"config", "langauge", "--no-interactive"}},
		{ID: "12-subcommand-typo-langauge-json", Args: []string{"config", "langauge", "--no-interactive", "--json"}},
		{ID: "13-too-many-args-nojson", Args: []string{"config", "language", "--no-interactive", "zh-CN", "extra"}},
		{ID: "14-too-many-args-json", Args: []string{"config", "language", "--no-interactive", "--json", "zh-CN", "extra"}},
		{ID: "15-set-language-success-nojson", Args: []string{"config", "language", "--no-interactive", "zh-CN"}},
		{ID: "16-set-language-success-json", Args: []string{"config", "language", "--no-interactive", "--json", "zh-CN"}},
	}

	results := runConfigLanguageCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "zh-CN", cases, results, update); err != nil {
		t.Fatal(err)
	}
}

func runConfigLanguageCases(t *testing.T, bin string, baseEnv []string, cases []snaptest.Case) []*snaptest.Result {
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
