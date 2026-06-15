package read_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemRead = "read"

func TestReadSnapshotsEN(t *testing.T) {
	runReadSnapshots(t, "en")
}

func TestReadSnapshotsZhCN(t *testing.T) {
	runReadSnapshots(t, "zh-CN")
}

func runReadSnapshots(t *testing.T, lang string) {
	t.Helper()
	path := snaptest.GoldenFileT(t, "read", goldenStemRead, lang)
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, lang)
	bin := snaptest.CraterExecutable(t)
	cases := []snaptest.Case{
		{ID: "01-node-typo-json", Args: []string{"node", "list", "--json", "--no-interactive"}},
		{ID: "02-node-get-missing-json", Args: []string{"node", "get", "--json", "--no-interactive"}},
		{ID: "03-node-ls-404-json", Args: []string{"node", "ls", "--json", "--no-interactive"}},
		{ID: "04-node-ls-gpu-available-missing-gpu-json", Args: []string{"node", "ls", "--gpu-available", "--json", "--no-interactive"}},
		{ID: "05-job-get-missing-json", Args: []string{"job", "get", "--json", "--no-interactive"}},
		{ID: "06-job-ls-invalid-status-json", Args: []string{"job", "ls", "--status", "bad", "--json", "--no-interactive"}},
		{ID: "07-job-ls-conflict-json", Args: []string{"job", "ls", "--interactive", "--batch", "--json", "--no-interactive"}},
		{ID: "08-job-ls-404-json", Args: []string{"job", "ls", "--json", "--no-interactive"}},
		{ID: "09-image-ls-invalid-type-json", Args: []string{"image", "ls", "--type", "bad", "--json", "--no-interactive"}},
		{ID: "10-image-ls-invalid-visibility-json", Args: []string{"image", "ls", "--visibility", "bad", "--json", "--no-interactive"}},
		{ID: "11-image-ls-404-json", Args: []string{"image", "ls", "--json", "--no-interactive"}},
	}
	results := make([]*snaptest.Result, len(cases))
	for i := range cases {
		env := baseEnv
		switch cases[i].ID {
		case "03-node-ls-404-json", "08-job-ls-404-json", "11-image-ls-404-json":
			env = append(baseEnv, "CRATER_TEST_SANDBOX_HTTP=error404")
		}
		r, err := snaptest.Run(bin, env, cases[i].Args)
		if err != nil {
			t.Fatalf("case %s: %v", cases[i].ID, err)
		}
		results[i] = r
	}
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, lang, cases, results, update); err != nil {
		t.Fatal(err)
	}
}
