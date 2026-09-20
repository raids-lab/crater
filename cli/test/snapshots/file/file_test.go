package file_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemFile = "file"

func TestFileSnapshotsEN(t *testing.T) {
	runFileSnapshots(t, "en")
}

func TestFileSnapshotsZhCN(t *testing.T) {
	runFileSnapshots(t, "zh-CN")
}

func runFileSnapshots(t *testing.T, language string) {
	t.Helper()
	path := snaptest.GoldenFileT(t, "file", goldenStemFile, language)
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, language)
	binary := snaptest.CraterExecutable(t)
	cases := []snaptest.Case{
		{ID: "01-file-typo-json", Args: []string{"file", "list", "--json", "--no-interactive"}},
		{ID: "02-file-ls-extra-arg-json", Args: []string{"file", "ls", "user", "extra", "--json", "--no-interactive"}},
		{ID: "03-file-ls-traversal-json", Args: []string{"file", "ls", "user/../public", "--json", "--no-interactive"}},
		{ID: "04-file-ls-invalid-root-json", Args: []string{"file", "ls", "admin/secret", "--json", "--no-interactive"}},
		{ID: "05-file-ls-root-404-json", Args: []string{"file", "ls", "--json", "--no-interactive"}},
		{ID: "06-file-ls-unicode-404-json", Args: []string{"file", "ls", "user/实验 data", "--json", "--no-interactive"}},
		{ID: "07-file-help", Args: []string{"file", "--help"}},
		{ID: "08-file-ls-help", Args: []string{"file", "ls", "--help"}},
	}

	results := make([]*snaptest.Result, len(cases))
	for index := range cases {
		environment := baseEnv
		switch cases[index].ID {
		case "05-file-ls-root-404-json", "06-file-ls-unicode-404-json":
			environment = append(baseEnv, "CRATER_TEST_SANDBOX_HTTP=error404")
		}
		result, err := snaptest.Run(binary, environment, cases[index].Args)
		if err != nil {
			t.Fatalf("case %s: %v", cases[index].ID, err)
		}
		results[index] = result
	}

	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, language, cases, results, update); err != nil {
		t.Fatal(err)
	}
}
