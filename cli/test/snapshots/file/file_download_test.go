package file_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemFileDownload = "file_download"

func TestFileDownloadSnapshotsEN(t *testing.T) {
	runFileDownloadSnapshots(t, "en")
}

func TestFileDownloadSnapshotsZhCN(t *testing.T) {
	runFileDownloadSnapshots(t, "zh-CN")
}

func runFileDownloadSnapshots(t *testing.T, language string) {
	t.Helper()
	path := snaptest.GoldenFileT(t, "file", goldenStemFileDownload, language)
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, language)
	binary := snaptest.CraterExecutable(t)
	localTarget := "snapshot-download-" + language + ".bin"
	cases := []snaptest.Case{
		{ID: "01-file-typo-json", Args: []string{"file", "get", "--json", "--no-interactive"}},
		{ID: "01b-file-typo-text", Args: []string{"file", "get", "--no-interactive"}},
		{ID: "02-file-download-missing-json", Args: []string{"file", "download", "--json", "--no-interactive"}},
		{ID: "03-file-download-extra-arg-json", Args: []string{"file", "download", "user/a.bin", "local.bin", "extra", "--json", "--no-interactive"}},
		{ID: "04-file-download-traversal-json", Args: []string{"file", "download", "user/../public/a.bin", "--json", "--no-interactive"}},
		{ID: "05-file-download-root-json", Args: []string{"file", "download", "user", "--json", "--no-interactive"}},
		{ID: "06-file-download-404-json", Args: []string{"file", "download", "user/实验 data/result.bin", localTarget, "--overwrite", "--json", "--no-interactive"}},
		{ID: "07-file-help", Args: []string{"file", "--help"}},
		{ID: "08-file-download-help", Args: []string{"file", "download", "--help"}},
	}

	results := make([]*snaptest.Result, len(cases))
	for index := range cases {
		environment := baseEnv
		if cases[index].ID == "06-file-download-404-json" {
			environment = append(baseEnv, "CRATER_TEST_SANDBOX_HTTP=error404")
		}
		result, err := snaptest.Run(binary, environment, cases[index].Args)
		if err != nil {
			t.Fatalf("case %s: %v", cases[index].ID, err)
		}
		results[index] = result
	}
	assertNoSnapshotDownloadArtifacts(t, localTarget)

	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, language, cases, results, update); err != nil {
		t.Fatal(err)
	}
}

func assertNoSnapshotDownloadArtifacts(t *testing.T, target string) {
	t.Helper()
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("snapshot target was not cleaned up: %s (err=%v)", target, err)
	}
	matches, err := filepath.Glob("." + filepath.Base(target) + ".crater-*")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("snapshot temporary files remain: %#v", matches)
	}
}
