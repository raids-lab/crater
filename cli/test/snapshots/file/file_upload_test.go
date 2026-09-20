package file_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemFileUpload = "file_upload"

func TestFileUploadSnapshotsEN(t *testing.T) {
	runFileUploadSnapshots(t, "en")
}

func TestFileUploadSnapshotsZhCN(t *testing.T) {
	runFileUploadSnapshots(t, "zh-CN")
}

func runFileUploadSnapshots(t *testing.T, language string) {
	t.Helper()
	path := snaptest.GoldenFileT(t, "file", goldenStemFileUpload, language)
	home := t.TempDir()
	baseEnv := append(snaptest.EnvMinimal(home, language), "CRATER_TEST_SANDBOX_HTTP=error404")
	binary := snaptest.CraterExecutable(t)
	localFixture := ".snapshot-upload-" + language + ".bin"
	if err := os.WriteFile(localFixture, []byte{0x00, 0xff, 'C', 'L', 'I'}, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(localFixture); err != nil {
			t.Error(err)
		}
	})

	cases := []snaptest.Case{
		{ID: "01-file-typo-json", Args: []string{"file", "get", "--json", "--no-interactive"}},
		{ID: "01b-file-typo-text", Args: []string{"file", "get", "--no-interactive"}},
		{ID: "02-file-upload-missing-local-json", Args: []string{"file", "upload", "--json", "--no-interactive"}},
		{ID: "03-file-upload-missing-remote-json", Args: []string{"file", "upload", localFixture, "--json", "--no-interactive"}},
		{ID: "04-file-upload-extra-arg-json", Args: []string{"file", "upload", localFixture, "user/a.bin", "extra", "--json", "--no-interactive"}},
		{ID: "05-file-upload-traversal-json", Args: []string{"file", "upload", localFixture, "user/../public/a.bin", "--json", "--no-interactive"}},
		{ID: "06-file-upload-root-json", Args: []string{"file", "upload", localFixture, "user", "--json", "--no-interactive"}},
		{ID: "07-file-upload-directory-json", Args: []string{"file", "upload", ".", "user/a.bin", "--json", "--no-interactive"}},
		{ID: "08-file-upload-404-json", Args: []string{"file", "upload", localFixture, "user/实验 data/result.bin", "--json", "--no-interactive"}},
		{ID: "09-file-help", Args: []string{"file", "--help"}},
		{ID: "10-file-upload-help", Args: []string{"file", "upload", "--help"}},
	}

	results := make([]*snaptest.Result, len(cases))
	for index := range cases {
		result, err := snaptest.Run(binary, baseEnv, cases[index].Args)
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
