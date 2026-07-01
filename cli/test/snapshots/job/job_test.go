package job_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemJob = "job"

func runJobCases(t *testing.T, bin string, baseEnv []string, cases []snaptest.Case) []*snaptest.Result {
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

func jobCases() []snaptest.Case {
	return []snaptest.Case{
		{ID: "01-unknown-nojson", Args: []string{"job", "wat", "--no-interactive"}},
		{ID: "02-unknown-json", Args: []string{"job", "wat", "--no-interactive", "--json"}},
		{ID: "03-get-missing-name-nojson", Args: []string{"job", "get", "--no-interactive"}},
		{ID: "04-get-missing-name-json", Args: []string{"job", "get", "--no-interactive", "--json"}},
		{ID: "05-ls-invalid-days-nojson", Args: []string{"job", "ls", "--no-interactive", "--days", "-2"}},
		{ID: "06-ls-invalid-status-json", Args: []string{"job", "ls", "--no-interactive", "--json", "--status", "bad"}},
		{ID: "07-create-jupyter-multi-usage-nojson", Args: []string{"job", "create", "jupyter", "--no-interactive", "--name", "demo", "--cpu", "-1", "--memory", "-2Gi", "--gpu", "-1"}},
		{ID: "08-create-jupyter-multi-usage-json", Args: []string{"job", "create", "jupyter", "--no-interactive", "--json", "--name", "demo", "--cpu", "-1", "--memory", "-2Gi", "--gpu", "-1"}},
		{ID: "09-create-custom-missing-working-dir-json", Args: []string{"job", "create", "custom", "--no-interactive", "--json", "--name", "demo", "--image", "example/image:tag", "--memory", "2Gi", "--working-dir", ""}},
		{ID: "10-admin-lock-missing-duration-nojson", Args: []string{"admin", "job", "lock", "job-123", "--no-interactive"}},
		{ID: "11-admin-clean-low-gpu-invalid-json", Args: []string{"admin", "job", "clean", "low-gpu", "--no-interactive", "--json", "--time-range", "0", "--wait-time", "-1"}},
		{ID: "12-ls-network-timeout-json", Args: []string{"job", "ls", "--no-interactive", "--json"}},
	}
}

func TestJobSnapshotsEN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "job", goldenStemJob, "en")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "en")
	baseEnv = append(baseEnv, "CRATER_TEST_SANDBOX_HTTP=timeout")
	bin := snaptest.CraterExecutable(t)
	cases := jobCases()
	results := runJobCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "en", cases, results, update); err != nil {
		t.Fatal(err)
	}
}

func TestJobSnapshotsZhCN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "job", goldenStemJob, "zh-CN")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "zh-CN")
	baseEnv = append(baseEnv, "CRATER_TEST_SANDBOX_HTTP=timeout")
	bin := snaptest.CraterExecutable(t)
	cases := jobCases()
	results := runJobCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "zh-CN", cases, results, update); err != nil {
		t.Fatal(err)
	}
}
