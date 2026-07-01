package order_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemOrder = "order"

func TestOrderSnapshotsEN(t *testing.T) {
	runOrderSnapshots(t, "en")
}

func TestOrderSnapshotsZhCN(t *testing.T) {
	runOrderSnapshots(t, "zh-CN")
}

func runOrderSnapshots(t *testing.T, lang string) {
	t.Helper()
	path := snaptest.GoldenFileT(t, "order", goldenStemOrder, lang)
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, lang)
	bin := snaptest.CraterExecutable(t)
	cases := []snaptest.Case{
		{ID: "01-submit-missing-json", Args: []string{"order", "submit", "--json", "--no-interactive"}},
		{ID: "02-submit-invalid-type-json", Args: []string{"order", "submit", "--name", "job-a", "--type", "bad", "--reason", "need more time", "--json", "--no-interactive"}},
		{ID: "03-edit-invalid-status-json", Args: []string{"order", "edit", "1", "--status", "Approved", "--json", "--no-interactive"}},
		{ID: "04-cancel-confirm-required-json", Args: []string{"order", "cancel", "1", "--json", "--no-interactive"}},
		{ID: "05-admin-reject-missing-notes-json", Args: []string{"admin", "order", "reject", "1", "--json", "--no-interactive"}},
		{ID: "06-admin-approve-lock-no-duration-json", Args: []string{"admin", "order", "approve", "1", "--lock", "--json", "--no-interactive"}},
		{ID: "07-admin-approve-lock-negative-json", Args: []string{"admin", "order", "approve", "1", "--lock", "--days", "-1", "--json", "--no-interactive"}},
		{ID: "08-admin-check-confirm-required-json", Args: []string{"admin", "order", "check", "--json", "--no-interactive"}},
		{ID: "09-submit-404-json", Args: []string{"order", "submit", "--name", "job-a", "--type", "job", "--reason", "need more time", "--json", "--no-interactive"}},
		{ID: "10-edit-404-json", Args: []string{"order", "edit", "1", "--reason", "updated reason", "--json", "--no-interactive"}},
		{ID: "11-cancel-404-json", Args: []string{"order", "cancel", "1", "--yes", "--json", "--no-interactive"}},
		{ID: "12-admin-approve-404-json", Args: []string{"admin", "order", "approve", "1", "--json", "--no-interactive"}},
		{ID: "13-admin-reject-404-json", Args: []string{"admin", "order", "reject", "1", "--review-notes", "no", "--json", "--no-interactive"}},
		{ID: "14-admin-check-404-json", Args: []string{"admin", "order", "check", "--yes", "--json", "--no-interactive"}},
	}
	results := make([]*snaptest.Result, len(cases))
	for i := range cases {
		env := baseEnv
		switch cases[i].ID {
		case "09-submit-404-json", "10-edit-404-json", "11-cancel-404-json", "12-admin-approve-404-json", "13-admin-reject-404-json", "14-admin-check-404-json":
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
