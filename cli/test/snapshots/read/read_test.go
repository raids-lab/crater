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
		{ID: "05-node-ls-extra-arg-json", Args: []string{"node", "ls", "unexpected", "--json", "--no-interactive"}},
		{ID: "06-node-pods-missing-json", Args: []string{"node", "pods", "--json", "--no-interactive"}},
		{ID: "07-node-gpu-missing-json", Args: []string{"node", "gpu", "--json", "--no-interactive"}},
		{ID: "08-node-pods-404-json", Args: []string{"node", "pods", "missing-node", "--json", "--no-interactive"}},
		{ID: "09-job-get-missing-json", Args: []string{"job", "get", "--json", "--no-interactive"}},
		{ID: "10-job-pods-missing-json", Args: []string{"job", "pods", "--json", "--no-interactive"}},
		{ID: "11-job-events-missing-json", Args: []string{"job", "events", "--json", "--no-interactive"}},
		{ID: "12-job-yaml-missing-json", Args: []string{"job", "yaml", "--json", "--no-interactive"}},
		{ID: "13-job-ls-invalid-status-json", Args: []string{"job", "ls", "--status", "bad", "--json", "--no-interactive"}},
		{ID: "14-job-ls-invalid-days-json", Args: []string{"job", "ls", "--days", "-2", "--json", "--no-interactive"}},
		{ID: "15-job-ls-conflict-json", Args: []string{"job", "ls", "--interactive", "--batch", "--json", "--no-interactive"}},
		{ID: "16-job-ls-trimmed-status-404-json", Args: []string{"job", "ls", "--status", "Running ", "--json", "--no-interactive"}},
		{ID: "17-job-yaml-404-json", Args: []string{"job", "yaml", "missing-job", "--json", "--no-interactive"}},
		{ID: "18-image-ls-invalid-type-json", Args: []string{"image", "ls", "--type", "bad", "--json", "--no-interactive"}},
		{ID: "19-image-ls-invalid-visibility-json", Args: []string{"image", "ls", "--visibility", "bad", "--json", "--no-interactive"}},
		{ID: "20-image-ls-trimmed-type-404-json", Args: []string{"image", "ls", "--type", "jupyter ", "--json", "--no-interactive"}},
		{ID: "21-account-get-missing-json", Args: []string{"account", "get", "--json", "--no-interactive"}},
		{ID: "22-account-admin-flag-removed-json", Args: []string{"account", "ls", "--admin", "--json", "--no-interactive"}},
		{ID: "23-admin-account-ls-404-json", Args: []string{"admin", "account", "ls", "--json", "--no-interactive"}},
		{ID: "24-resource-networks-missing-json", Args: []string{"resource", "networks", "--json", "--no-interactive"}},
		{ID: "25-resource-admin-flag-removed-json", Args: []string{"resource", "networks", "1", "--admin", "--json", "--no-interactive"}},
		{ID: "26-dataset-get-invalid-id-json", Args: []string{"dataset", "get", "abc", "--json", "--no-interactive"}},
		{ID: "27-admin-dataset-ls-404-json", Args: []string{"admin", "dataset", "ls", "--json", "--no-interactive"}},
		{ID: "28-template-ls-404-json", Args: []string{"template", "ls", "--json", "--no-interactive"}},
		{ID: "29-model-download-logs-missing-json", Args: []string{"model-download", "logs", "--json", "--no-interactive"}},
		{ID: "30-context-resources-404-json", Args: []string{"context", "resources", "--json", "--no-interactive"}},
		{ID: "31-order-by-name-missing-json", Args: []string{"order", "by-name", "--json", "--no-interactive"}},
		{ID: "32-admin-order-ls-404-json", Args: []string{"admin", "order", "ls", "--json", "--no-interactive"}},
		{ID: "33-user-get-missing-json", Args: []string{"user", "get", "--json", "--no-interactive"}},
		{ID: "34-admin-user-ls-404-json", Args: []string{"admin", "user", "ls", "--json", "--no-interactive"}},
		{ID: "35-pod-logs-missing-json", Args: []string{"pod", "logs", "ns", "pod", "--json", "--no-interactive"}},
		{ID: "36-billing-jobs-404-json", Args: []string{"billing", "jobs", "--all", "--json", "--no-interactive"}},
		{ID: "37-admin-billing-jobs-404-json", Args: []string{"admin", "billing", "jobs", "--json", "--no-interactive"}},
		{ID: "38-aijob-removed-json", Args: []string{"aijob", "ls", "--json", "--no-interactive"}},
		{ID: "39-spjob-removed-json", Args: []string{"spjob", "yaml", "--json", "--no-interactive"}},
		{ID: "40-admin-operation-logs-404-json", Args: []string{"admin", "operation-logs", "--json", "--no-interactive"}},
		{ID: "41-admin-system-config-llm-404-json", Args: []string{"admin", "system-config", "llm", "--json", "--no-interactive"}},
	}
	results := make([]*snaptest.Result, len(cases))
	for i := range cases {
		env := baseEnv
		switch cases[i].ID {
		case "03-node-ls-404-json", "08-node-pods-404-json", "16-job-ls-trimmed-status-404-json", "17-job-yaml-404-json", "20-image-ls-trimmed-type-404-json",
			"23-admin-account-ls-404-json", "27-admin-dataset-ls-404-json", "28-template-ls-404-json", "30-context-resources-404-json", "32-admin-order-ls-404-json",
			"34-admin-user-ls-404-json", "36-billing-jobs-404-json", "37-admin-billing-jobs-404-json", "40-admin-operation-logs-404-json", "41-admin-system-config-llm-404-json":
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
