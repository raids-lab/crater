package handler

import (
	"testing"

	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/pkg/config"
)

func TestServingProfileControlsRuntimeImages(t *testing.T) {
	cfg := config.GetConfig()
	old := cfg.Kthena.RuntimeImages
	t.Cleanup(func() { cfg.Kthena.RuntimeImages = old })
	cfg.Kthena.RuntimeImages = map[string][]string{
		"vLLM/combined/single": {"registry.example/vllm:v0.10.0", "registry.example/vllm:latest"},
	}
	req := &CreateKthenaReq{
		BackendType: kthena.EngineVLLM,
		Layout:      kthena.LayoutCombined,
		Roles: []kthena.RoleSpec{
			{Name: "server", Execution: kthena.ExecutionSingle, Entry: kthena.PodProfile{Image: "registry.example/vllm:v0.10.0"}},
		},
	}
	if err := validateServingCapability(req); err != nil {
		t.Fatal(err)
	}
	for _, image := range []string{kthena.VLLMImage, "registry.example/vllm:latest", "registry.example/vllm:v0.9.0"} {
		req.Roles[0].Entry.Image = image
		if validateServingCapability(req) == nil {
			t.Fatalf("unapproved image accepted: %s", image)
		}
	}
	if servingProfileConnector(kthena.EngineVLLM, kthena.LayoutPD) != "NixlConnector" ||
		servingProfileConnector(kthena.EngineSGLang, kthena.LayoutPD) != "mooncake" {
		t.Fatal("incorrect KV contract")
	}
}
