package kthena

import (
	"encoding/json"
	"testing"
)

func TestRuntimeGPUQuantitiesAgreeWithValidation(t *testing.T) {
	for _, quantity := range []string{"1", "1e0", "1000m"} {
		t.Run(quantity, func(t *testing.T) {
			profile := PodProfile{Image: VLLMImage, CPU: "1", Memory: "1Gi", GPU: quantity, GPUModel: "nvidia.com/gpu"}
			worker := profile
			worker.GPU = "1"
			roles := []RoleSpec{
				{
					Name:             "server",
					Instances:        1,
					Execution:        ExecutionRay,
					NodesPerInstance: 2,
					TensorParallel:   1,
					PipelineParallel: 2,
					Entry:            profile,
					Worker:           &worker,
				},
			}
			if err := ValidateRoles(EngineVLLM, LayoutCombined, 1, roles); err != nil {
				t.Fatal(err)
			}
			cmd := RuntimeCommand(EngineVLLM, LayoutCombined, "/model", "test", 8000, &roles[0], true)
			var launch struct {
				GPUs int64 `json:"gpus"`
			}
			if err := json.Unmarshal([]byte(cmd[4]), &launch); err != nil {
				t.Fatal(err)
			}
			if launch.GPUs != 1 {
				t.Fatalf("gpus=%d", launch.GPUs)
			}
		})
	}
	for _, q := range []string{"500m", "-1", "invalid"} {
		p := PodProfile{Image: VLLMImage, CPU: "1", Memory: "1Gi", GPU: q}
		if validateProfile(&p) == nil {
			t.Fatalf("accepted %s", q)
		}
	}
}
