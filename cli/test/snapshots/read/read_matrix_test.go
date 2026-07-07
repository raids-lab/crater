package read_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

func TestReadCommandMatrix(t *testing.T) {
	home := t.TempDir()
	baseEnv := snaptest.EnvMinimal(home, "en")
	bin := snaptest.CraterExecutable(t)

	commands := [][]string{
		{"node", "ls"}, {"node", "get"}, {"node", "pods"}, {"node", "gpu"},
		{"job", "ls"}, {"job", "get"}, {"job", "pods"}, {"job", "events"}, {"job", "yaml"},
		{"image", "ls"},
		{"account", "ls"}, {"account", "get"}, {"account", "members"}, {"account", "users-out"}, {"account", "billing", "config"}, {"account", "billing", "members"},
		{"resource", "ls"}, {"resource", "networks"}, {"resource", "vgpu"}, {"resource", "prices"},
		{"dataset", "ls"}, {"dataset", "get"}, {"dataset", "users"}, {"dataset", "queues"}, {"dataset", "users-out"}, {"dataset", "queues-out"},
		{"template", "ls"}, {"template", "get"},
		{"model-download", "ls"}, {"model-download", "get"}, {"model-download", "logs"},
		{"context", "prequeue"}, {"context", "quota"}, {"context", "resources"}, {"context", "billing"},
		{"billing", "status"}, {"billing", "summary"}, {"billing", "prices"}, {"billing", "jobs"}, {"billing", "job"},
		{"order", "ls"}, {"order", "get"}, {"order", "by-name"}, {"order", "submit"}, {"order", "edit"}, {"order", "cancel"},
		{"user", "get"}, {"user", "email-verified"},
		{"pod", "containers"}, {"pod", "events"}, {"pod", "logs"}, {"pod", "ingresses"}, {"pod", "nodeports"},
		{"admin", "system-config", "llm"}, {"admin", "system-config", "gpu-analysis"}, {"admin", "system-config", "prequeue"},
		{"admin", "queue-quotas"}, {"admin", "gpu-analyses"}, {"admin", "operation-logs"}, {"admin", "cronjobs"}, {"admin", "whitelist"},
		{"admin", "account", "ls"}, {"admin", "account", "get"}, {"admin", "account", "members"}, {"admin", "account", "users-out"}, {"admin", "account", "quota"}, {"admin", "account", "billing", "config"}, {"admin", "account", "billing", "members"},
		{"admin", "resource", "networks"}, {"admin", "resource", "vgpu"},
		{"admin", "dataset", "ls"},
		{"admin", "model-download", "ls"},
		{"admin", "billing", "status"}, {"admin", "billing", "jobs"},
		{"admin", "order", "ls"}, {"admin", "order", "get"}, {"admin", "order", "approve"}, {"admin", "order", "reject"}, {"admin", "order", "check"},
		{"admin", "user", "ls"}, {"admin", "user", "billing", "summary"}, {"admin", "user", "billing", "accounts"},
	}

	for _, command := range commands {
		args := append(append([]string{}, command...), "--help")
		result, err := snaptest.Run(bin, baseEnv, args)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("%v: help exit=%d stderr=%s", args, result.ExitCode, result.Stderr)
		}
	}

	apiCases := [][]string{
		{"node", "ls"}, {"job", "ls"}, {"image", "ls"},
		{"account", "ls"}, {"resource", "ls"}, {"dataset", "ls"}, {"template", "ls"}, {"model-download", "ls"},
		{"context", "prequeue"}, {"context", "quota"}, {"context", "resources"}, {"context", "billing"},
		{"billing", "status"}, {"billing", "summary"}, {"billing", "prices"}, {"billing", "jobs"},
		{"order", "ls"}, {"user", "email-verified"},
		{"admin", "system-config", "llm"}, {"admin", "system-config", "gpu-analysis"}, {"admin", "system-config", "prequeue"},
		{"admin", "queue-quotas"}, {"admin", "gpu-analyses"}, {"admin", "operation-logs"}, {"admin", "cronjobs"}, {"admin", "whitelist"},
		{"admin", "account", "ls"}, {"admin", "dataset", "ls"}, {"admin", "model-download", "ls"},
		{"admin", "billing", "status"}, {"admin", "billing", "jobs"}, {"admin", "order", "ls"}, {"admin", "user", "ls"}, {"admin", "user", "billing", "summary"},
	}
	env404 := append(baseEnv, "CRATER_TEST_SANDBOX_HTTP=error404")
	for _, command := range apiCases {
		args := append(append([]string{}, command...), "--json", "--no-interactive")
		result, err := snaptest.Run(bin, env404, args)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if result.ExitCode != 4 {
			t.Fatalf("%v: expected api error exit=4, got exit=%d stderr=%s", args, result.ExitCode, result.Stderr)
		}
	}

	if os.Getenv("CRATER_MATRIX_DEBUG") != "" {
		t.Logf("checked %d help commands and %d api commands", len(commands), len(apiCases))
	}
}
