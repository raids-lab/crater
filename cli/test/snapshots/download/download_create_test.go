package download_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemCreate = "download_create"

func runDownloadCases(t *testing.T, bin string, baseEnv []string, cases []snaptest.Case) []*snaptest.Result {
	t.Helper()
	out := make([]*snaptest.Result, len(cases))
	for i := range cases {
		env := appendHTTPEnv(baseEnv, cases[i].ID)
		r, err := snaptest.Run(bin, env, cases[i].Args)
		if err != nil {
			t.Fatalf("case %s: %v", cases[i].ID, err)
		}
		out[i] = r
	}
	return out
}

func appendHTTPEnv(env []string, caseID string) []string {
	switch caseID {
	case "09-http-sim-timeout-nojson", "10-http-sim-timeout-json",
		"19-shortcut-model-timeout-nojson", "20-shortcut-model-timeout-json",
		"25-ls-timeout-json", "30-get-timeout-json", "33-logs-timeout-json",
		"36-pause-timeout-json", "39-resume-timeout-json", "42-retry-timeout-json",
		"46-rm-timeout-json":
		return append(env, "CRATER_TEST_SANDBOX_HTTP=timeout")
	case "11-http-sim-404-nojson", "12-http-sim-404-json":
		return append(env, "CRATER_TEST_SANDBOX_HTTP=error404")
	case "17-no-active-nojson", "18-no-active-json":
		return append(env, "CRATER_TEST_SANDBOX_SESSION_EMPTY=1")
	default:
		return env
	}
}

func TestDownloadCreateSnapshotsEN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "download", goldenStemCreate, "en")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "en")
	bin := snaptest.CraterExecutable(t)

	cases := []snaptest.Case{
		{ID: "01-missing-name-nojson", Args: []string{"download", "create", "--no-interactive", "--category", "model"}},
		{ID: "02-missing-name-json", Args: []string{"download", "create", "--no-interactive", "--json", "--category", "model"}},
		{ID: "03-missing-category-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct"}},
		{ID: "04-missing-category-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct"}},
		{ID: "05-invalid-name-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "bad", "--category", "model"}},
		{ID: "06-invalid-name-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "bad", "--category", "model"}},
		{ID: "07-invalid-source-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "bad"}},
		{ID: "08-invalid-source-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "bad"}},
		{ID: "09-http-sim-timeout-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "hf"}},
		{ID: "10-http-sim-timeout-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "hf"}},
		{ID: "11-http-sim-404-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "AI-ModelScope/alpaca-gpt4-data-zh", "--category", "dataset", "--source", "ms"}},
		{ID: "12-http-sim-404-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "AI-ModelScope/alpaca-gpt4-data-zh", "--category", "dataset", "--source", "ms"}},
		{ID: "13-invalid-category-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "bad"}},
		{ID: "14-invalid-category-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "bad"}},
		{ID: "15-multiple-issues-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "bad", "--category", "bad", "--source", "bad"}},
		{ID: "16-multiple-issues-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "bad", "--category", "bad", "--source", "bad"}},
		{ID: "17-no-active-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "hf"}},
		{ID: "18-no-active-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "hf"}},
		{ID: "19-shortcut-model-timeout-nojson", Args: []string{"download", "model", "qwen/Qwen2.5-Coder-7B-Instruct", "--no-interactive", "--source", "hf"}},
		{ID: "20-shortcut-model-timeout-json", Args: []string{"download", "model", "qwen/Qwen2.5-Coder-7B-Instruct", "--no-interactive", "--json", "--source", "hf"}},
		{ID: "21-shortcut-dataset-invalid-name-nojson", Args: []string{"download", "dataset", "bad", "--no-interactive", "--source", "ms"}},
		{ID: "22-shortcut-dataset-invalid-name-json", Args: []string{"download", "dataset", "bad", "--no-interactive", "--json", "--source", "ms"}},
		{ID: "23-token-source-conflict-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--token", "a", "--token-env", "HF_TOKEN"}},
		{ID: "24-token-source-conflict-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--token", "a", "--token-env", "HF_TOKEN"}},
		{ID: "25-ls-timeout-json", Args: []string{"download", "ls", "--no-interactive", "--json", "--category", "model"}},
		{ID: "26-ls-invalid-category-nojson", Args: []string{"download", "ls", "--no-interactive", "--category", "bad"}},
		{ID: "27-ls-extra-arg-json", Args: []string{"download", "ls", "unexpected", "--no-interactive", "--json"}},
		{ID: "28-get-missing-id-nojson", Args: []string{"download", "get", "--no-interactive"}},
		{ID: "29-get-missing-id-json", Args: []string{"download", "get", "--no-interactive", "--json"}},
		{ID: "30-get-timeout-json", Args: []string{"download", "get", "123", "--no-interactive", "--json"}},
		{ID: "31-get-invalid-id-nojson", Args: []string{"download", "get", "bad", "--no-interactive"}},
		{ID: "32-logs-follow-json", Args: []string{"download", "logs", "123", "--follow", "--no-interactive", "--json"}},
		{ID: "33-logs-timeout-json", Args: []string{"download", "logs", "123", "--no-interactive", "--json"}},
		{ID: "34-rm-missing-yes-nojson", Args: []string{"download", "rm", "123", "--no-interactive"}},
		{ID: "35-rm-missing-yes-json", Args: []string{"download", "rm", "123", "--no-interactive", "--json"}},
		{ID: "36-pause-timeout-json", Args: []string{"download", "pause", "123", "--no-interactive", "--json"}},
		{ID: "37-pause-invalid-id-nojson", Args: []string{"download", "pause", "bad", "--no-interactive"}},
		{ID: "38-resume-invalid-id-nojson", Args: []string{"download", "resume", "bad", "--no-interactive"}},
		{ID: "39-resume-timeout-json", Args: []string{"download", "resume", "123", "--no-interactive", "--json"}},
		{ID: "40-retry-invalid-id-nojson", Args: []string{"download", "retry", "bad", "--no-interactive"}},
		{ID: "41-rm-invalid-id-nojson", Args: []string{"download", "rm", "bad", "--no-interactive", "--yes"}},
		{ID: "42-retry-timeout-json", Args: []string{"download", "retry", "123", "--no-interactive", "--json"}},
		{ID: "43-token-env-missing-nojson", Args: []string{"download", "model", "qwen/Qwen2.5-Coder-7B-Instruct", "--no-interactive", "--source", "hf", "--token-env", "HF_TOKEN_NOT_SET"}},
		{ID: "44-token-env-missing-json", Args: []string{"download", "model", "qwen/Qwen2.5-Coder-7B-Instruct", "--no-interactive", "--json", "--source", "hf", "--token-env", "HF_TOKEN_NOT_SET"}},
		{ID: "45-token-stdin-empty-json", Args: []string{"download", "model", "qwen/Qwen2.5-Coder-7B-Instruct", "--no-interactive", "--json", "--source", "hf", "--token-stdin"}},
		{ID: "46-rm-timeout-json", Args: []string{"download", "rm", "123", "--no-interactive", "--json", "--yes"}},
	}

	results := runDownloadCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "en", cases, results, update); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadCreateSnapshotsZhCN(t *testing.T) {
	path := snaptest.GoldenFileT(t, "download", goldenStemCreate, "zh-CN")
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "zh-CN")
	bin := snaptest.CraterExecutable(t)

	cases := []snaptest.Case{
		{ID: "01-missing-name-nojson", Args: []string{"download", "create", "--no-interactive", "--category", "model"}},
		{ID: "02-missing-name-json", Args: []string{"download", "create", "--no-interactive", "--json", "--category", "model"}},
		{ID: "03-missing-category-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct"}},
		{ID: "04-missing-category-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct"}},
		{ID: "05-invalid-name-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "bad", "--category", "model"}},
		{ID: "06-invalid-name-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "bad", "--category", "model"}},
		{ID: "07-invalid-source-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "bad"}},
		{ID: "08-invalid-source-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "bad"}},
		{ID: "09-http-sim-timeout-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "hf"}},
		{ID: "10-http-sim-timeout-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "hf"}},
		{ID: "11-http-sim-404-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "AI-ModelScope/alpaca-gpt4-data-zh", "--category", "dataset", "--source", "ms"}},
		{ID: "12-http-sim-404-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "AI-ModelScope/alpaca-gpt4-data-zh", "--category", "dataset", "--source", "ms"}},
		{ID: "13-invalid-category-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "bad"}},
		{ID: "14-invalid-category-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "bad"}},
		{ID: "15-multiple-issues-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "bad", "--category", "bad", "--source", "bad"}},
		{ID: "16-multiple-issues-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "bad", "--category", "bad", "--source", "bad"}},
		{ID: "17-no-active-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "hf"}},
		{ID: "18-no-active-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--source", "hf"}},
		{ID: "19-shortcut-model-timeout-nojson", Args: []string{"download", "model", "qwen/Qwen2.5-Coder-7B-Instruct", "--no-interactive", "--source", "hf"}},
		{ID: "20-shortcut-model-timeout-json", Args: []string{"download", "model", "qwen/Qwen2.5-Coder-7B-Instruct", "--no-interactive", "--json", "--source", "hf"}},
		{ID: "21-shortcut-dataset-invalid-name-nojson", Args: []string{"download", "dataset", "bad", "--no-interactive", "--source", "ms"}},
		{ID: "22-shortcut-dataset-invalid-name-json", Args: []string{"download", "dataset", "bad", "--no-interactive", "--json", "--source", "ms"}},
		{ID: "23-token-source-conflict-nojson", Args: []string{"download", "create", "--no-interactive", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--token", "a", "--token-env", "HF_TOKEN"}},
		{ID: "24-token-source-conflict-json", Args: []string{"download", "create", "--no-interactive", "--json", "--name", "qwen/Qwen2.5-Coder-7B-Instruct", "--category", "model", "--token", "a", "--token-env", "HF_TOKEN"}},
		{ID: "25-ls-timeout-json", Args: []string{"download", "ls", "--no-interactive", "--json", "--category", "model"}},
		{ID: "26-ls-invalid-category-nojson", Args: []string{"download", "ls", "--no-interactive", "--category", "bad"}},
		{ID: "27-ls-extra-arg-json", Args: []string{"download", "ls", "unexpected", "--no-interactive", "--json"}},
		{ID: "28-get-missing-id-nojson", Args: []string{"download", "get", "--no-interactive"}},
		{ID: "29-get-missing-id-json", Args: []string{"download", "get", "--no-interactive", "--json"}},
		{ID: "30-get-timeout-json", Args: []string{"download", "get", "123", "--no-interactive", "--json"}},
		{ID: "31-get-invalid-id-nojson", Args: []string{"download", "get", "bad", "--no-interactive"}},
		{ID: "32-logs-follow-json", Args: []string{"download", "logs", "123", "--follow", "--no-interactive", "--json"}},
		{ID: "33-logs-timeout-json", Args: []string{"download", "logs", "123", "--no-interactive", "--json"}},
		{ID: "34-rm-missing-yes-nojson", Args: []string{"download", "rm", "123", "--no-interactive"}},
		{ID: "35-rm-missing-yes-json", Args: []string{"download", "rm", "123", "--no-interactive", "--json"}},
		{ID: "36-pause-timeout-json", Args: []string{"download", "pause", "123", "--no-interactive", "--json"}},
		{ID: "37-pause-invalid-id-nojson", Args: []string{"download", "pause", "bad", "--no-interactive"}},
		{ID: "38-resume-invalid-id-nojson", Args: []string{"download", "resume", "bad", "--no-interactive"}},
		{ID: "39-resume-timeout-json", Args: []string{"download", "resume", "123", "--no-interactive", "--json"}},
		{ID: "40-retry-invalid-id-nojson", Args: []string{"download", "retry", "bad", "--no-interactive"}},
		{ID: "41-rm-invalid-id-nojson", Args: []string{"download", "rm", "bad", "--no-interactive", "--yes"}},
		{ID: "42-retry-timeout-json", Args: []string{"download", "retry", "123", "--no-interactive", "--json"}},
		{ID: "43-token-env-missing-nojson", Args: []string{"download", "model", "qwen/Qwen2.5-Coder-7B-Instruct", "--no-interactive", "--source", "hf", "--token-env", "HF_TOKEN_NOT_SET"}},
		{ID: "44-token-env-missing-json", Args: []string{"download", "model", "qwen/Qwen2.5-Coder-7B-Instruct", "--no-interactive", "--json", "--source", "hf", "--token-env", "HF_TOKEN_NOT_SET"}},
		{ID: "45-token-stdin-empty-json", Args: []string{"download", "model", "qwen/Qwen2.5-Coder-7B-Instruct", "--no-interactive", "--json", "--source", "hf", "--token-stdin"}},
		{ID: "46-rm-timeout-json", Args: []string{"download", "rm", "123", "--no-interactive", "--json", "--yes"}},
	}

	results := runDownloadCases(t, bin, baseEnv, cases)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, "zh-CN", cases, results, update); err != nil {
		t.Fatal(err)
	}
}
