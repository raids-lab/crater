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
		{"job", "ls"}, {"job", "get"}, {"job", "pods"}, {"job", "events"}, {"job", "yaml"}, {"job", "template"}, {"job", "token"}, {"job", "secret"}, {"job", "ssh"}, {"job", "snapshot"}, {"job", "alert"}, {"job", "delete"},
		{"job", "create", "jupyter"}, {"job", "create", "webide"}, {"job", "create", "custom"}, {"job", "create", "tensorflow"}, {"job", "create", "pytorch"},
		{"image", "ls"}, {"image", "build", "ls"}, {"image", "build", "pip-apt"}, {"image", "build", "dockerfile"}, {"image", "build", "envd"}, {"image", "build", "remove"}, {"image", "build", "get"}, {"image", "build", "template"}, {"image", "build", "pod"},
		{"image", "upload"}, {"image", "delete"}, {"image", "delete-many"}, {"image", "description"}, {"image", "type"}, {"image", "tags"}, {"image", "arch"}, {"image", "valid"},
		{"image", "share", "ls"}, {"image", "share", "users"}, {"image", "share", "accounts"}, {"image", "share", "add"}, {"image", "share", "remove"},
		{"image", "cuda", "ls"}, {"image", "harbor", "info"}, {"image", "harbor", "credential"}, {"image", "quota", "get"}, {"image", "quota", "set"},
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
		{"admin", "image", "build-ls"}, {"admin", "image", "build-remove"}, {"admin", "image", "ls"}, {"admin", "image", "delete-many"}, {"admin", "image", "description"}, {"admin", "image", "type"}, {"admin", "image", "tags"}, {"admin", "image", "arch"}, {"admin", "image", "public"}, {"admin", "image", "cuda", "add"}, {"admin", "image", "cuda", "delete"},
		{"admin", "job", "ls"}, {"admin", "job", "delete"}, {"admin", "job", "lock"}, {"admin", "job", "unlock"}, {"admin", "job", "keep"},
		{"admin", "job", "clean", "waiting-jupyter"}, {"admin", "job", "clean", "waiting-custom"}, {"admin", "job", "clean", "long-running"}, {"admin", "job", "clean", "low-gpu"},
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
		{"node", "ls"}, {"job", "ls"}, {"image", "ls"}, {"image", "build", "ls"}, {"image", "cuda", "ls"}, {"image", "harbor", "info"}, {"image", "quota", "get"},
		{"account", "ls"}, {"resource", "ls"}, {"dataset", "ls"}, {"template", "ls"}, {"model-download", "ls"},
		{"context", "prequeue"}, {"context", "quota"}, {"context", "resources"}, {"context", "billing"},
		{"billing", "status"}, {"billing", "summary"}, {"billing", "prices"}, {"billing", "jobs"},
		{"order", "ls"}, {"user", "email-verified"},
		{"admin", "system-config", "llm"}, {"admin", "system-config", "gpu-analysis"}, {"admin", "system-config", "prequeue"},
		{"admin", "queue-quotas"}, {"admin", "gpu-analyses"}, {"admin", "operation-logs"}, {"admin", "cronjobs"}, {"admin", "whitelist"},
		{"admin", "account", "ls"}, {"admin", "dataset", "ls"}, {"admin", "model-download", "ls"},
		{"admin", "billing", "status"}, {"admin", "billing", "jobs"}, {"admin", "image", "ls"}, {"admin", "image", "build-ls"}, {"admin", "job", "ls"}, {"admin", "order", "ls"}, {"admin", "user", "ls"}, {"admin", "user", "billing", "summary"},
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
