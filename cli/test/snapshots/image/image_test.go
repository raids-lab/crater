package image_test

import (
	"os"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemImage = "image"

func TestImageSnapshotsEN(t *testing.T) {
	runImageSnapshots(t, "en")
}

func TestImageSnapshotsZhCN(t *testing.T) {
	runImageSnapshots(t, "zh-CN")
}

func TestImageUnknownSubcommandText(t *testing.T) {
	home := t.TempDir()
	result, err := snaptest.Run(snaptest.CraterExecutable(t), snaptest.EnvMinimal(home, "en"), []string{"image", "list", "--no-interactive"})
	if err != nil {
		t.Fatalf("run image list: %v", err)
	}
	if result.ExitCode != 2 {
		t.Fatalf("exit code = %d, want 2", result.ExitCode)
	}
	if !strings.Contains(result.Stderr, `unknown command "list" for "crater image"`) {
		t.Fatalf("stderr missing unknown-command error: %q", result.Stderr)
	}
}

func runImageSnapshots(t *testing.T, lang string) {
	t.Helper()
	path := snaptest.GoldenFileT(t, "image", goldenStemImage, lang)
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, lang)
	bin := snaptest.CraterExecutable(t)
	cases := []snaptest.Case{
		{ID: "01-image-typo-json", Args: []string{"image", "list", "--json", "--no-interactive"}},
		{ID: "02-build-pip-apt-missing-json", Args: []string{"image", "build", "pip-apt", "--json", "--no-interactive"}},
		{ID: "03-build-dockerfile-missing-content-json", Args: []string{"image", "build", "dockerfile", "--name", "img", "--tag", "v1", "--json", "--no-interactive"}},
		{ID: "04-build-envd-invalid-source-json", Args: []string{"image", "build", "envd", "--name", "img", "--tag", "v1", "--envd", "x", "--build-source", "bad", "--json", "--no-interactive"}},
		{ID: "05-upload-missing-image-json", Args: []string{"image", "upload", "--json", "--no-interactive"}},
		{ID: "06-delete-invalid-id-json", Args: []string{"image", "delete", "abc", "--json", "--no-interactive"}},
		{ID: "07-delete-many-invalid-ids-json", Args: []string{"image", "delete-many", "--ids", "1,x", "--json", "--no-interactive"}},
		{ID: "08-share-add-missing-ids-json", Args: []string{"image", "share", "add", "1", "--json", "--no-interactive"}},
		{ID: "09-share-remove-missing-target-json", Args: []string{"image", "share", "remove", "1", "--json", "--no-interactive"}},
		{ID: "10-harbor-credential-confirm-json", Args: []string{"image", "harbor", "credential", "--json", "--no-interactive"}},
		{ID: "11-cuda-add-missing-json", Args: []string{"image", "cuda", "add", "--json", "--no-interactive"}},
		{ID: "12-valid-missing-links-json", Args: []string{"image", "valid", "--json", "--no-interactive"}},
		{ID: "13-build-get-missing-json", Args: []string{"image", "build", "get", "--json", "--no-interactive"}},
		{ID: "14-build-pod-invalid-json", Args: []string{"image", "build", "pod", "bad", "--json", "--no-interactive"}},
		{ID: "15-admin-build-remove-missing-json", Args: []string{"admin", "image", "build-remove", "--json", "--no-interactive"}},
		{ID: "16-admin-type-invalid-json", Args: []string{"admin", "image", "type", "1", "--type", "all", "--json", "--no-interactive"}},
		{ID: "17-image-ls-404-json", Args: []string{"image", "ls", "--json", "--no-interactive"}},
		{ID: "18-image-available-404-json", Args: []string{"image", "ls", "--available", "--json", "--no-interactive"}},
		{ID: "19-build-pip-apt-404-json", Args: []string{"image", "build", "pip-apt", "--name", "img", "--tag", "v1", "--image", "base:latest", "--json", "--no-interactive"}},
		{ID: "20-share-ls-404-json", Args: []string{"image", "share", "ls", "1", "--json", "--no-interactive"}},
		{ID: "21-admin-image-ls-404-json", Args: []string{"admin", "image", "ls", "--json", "--no-interactive"}},
		{ID: "22-build-dockerfile-mutually-exclusive-json", Args: []string{"image", "build", "dockerfile", "--name", "img", "--tag", "v1", "--dockerfile", "FROM scratch", "--file", "Dockerfile", "--json", "--no-interactive"}},
		{ID: "23-image-arch-missing-json", Args: []string{"image", "arch", "1", "--json", "--no-interactive"}},
		{ID: "24-admin-cuda-add-missing-json", Args: []string{"admin", "image", "cuda", "add", "--json", "--no-interactive"}},
		{ID: "25-image-help", Args: []string{"image", "--help"}},
		{ID: "26-image-upload-help", Args: []string{"image", "upload", "--help"}},
		{ID: "27-image-build-pip-apt-help", Args: []string{"image", "build", "pip-apt", "--help"}},
	}
	results := make([]*snaptest.Result, len(cases))
	for i := range cases {
		env := baseEnv
		switch cases[i].ID {
		case "17-image-ls-404-json", "18-image-available-404-json", "19-build-pip-apt-404-json", "20-share-ls-404-json", "21-admin-image-ls-404-json":
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
