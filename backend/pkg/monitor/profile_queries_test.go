package monitor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func testProfileExpressions(t *testing.T, interval string, input []map[string]string, expressions []map[string]any) {
	t.Helper()
	if _, err := exec.LookPath("promtool"); err != nil {
		t.Fatal("promtool is required to evaluate the generated PromQL")
	}
	fixture, err := yaml.Marshal(map[string]any{
		"evaluation_interval": interval,
		"tests": []map[string]any{{
			"interval": interval, "input_series": input, "promql_expr_test": expressions,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	fixturePath := filepath.Join(t.TempDir(), "profile.yml")
	if err := os.WriteFile(fixturePath, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "promtool", "test", "rules", fixturePath).CombinedOutput()
	if err != nil {
		t.Fatalf("PromQL evaluation failed: %v\n%s", err, output)
	}
}

func TestProfileResourceQuery(t *testing.T) {
	input := []map[string]string{
		{"series": `kube_pod_container_resource_requests{namespace="workspace",pod="master.0",container="train",resource="cpu",instance="a"}`, "values": "0 3"},
		{"series": `kube_pod_container_resource_requests{namespace="workspace",pod="master.0",container="train",resource="cpu",instance="b"}`, "values": "0 3"},
		{"series": `kube_pod_container_resource_requests{namespace="workspace",pod="worker-0",container="train",resource="cpu"}`, "values": "0 4"},
		{"series": `kube_pod_container_resource_requests{namespace="workspace",pod="masterx0",container="train",resource="cpu"}`, "values": "0 99"},
	}
	query := profileResourceQuery("kube_pod_container_resource_requests", "cpu", "workspace",
		profilePodPattern([]string{"master.0", "worker-0"}), "1m")
	testProfileExpressions(t, "1m", input, []map[string]any{
		{"expr": query, "eval_time": "1m", "exp_samples": []map[string]any{{"labels": "{}", "value": 7}}},
	})
}
