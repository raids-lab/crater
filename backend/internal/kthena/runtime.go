package kthena

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"
)

//go:embed launch.py
var launchScript string

func RuntimeCommand(engine, layout, modelPath, modelName string, port int64, role *RoleSpec, entry bool) []string {
	argv := []string{"python3", "-m", "vllm.entrypoints.openai.api_server", "--model", modelPath, "--host", "0.0.0.0"}
	if engine == "SGLang" {
		argv = []string{"python3", "-m", "sglang.launch_server", "--model-path", modelPath, "--host", "__POD_IP__", "--enable-metrics"}
	}
	argv = append(argv, "--port", strconv.FormatInt(port, 10), "--served-model-name", modelName)
	if engine == EngineVLLM {
		argv = append(
			argv,
			"--tensor-parallel-size",
			strconv.FormatInt(role.TensorParallel, 10),
			"--pipeline-parallel-size",
			strconv.FormatInt(role.PipelineParallel, 10),
		)
	} else {
		argv = append(argv, "--tp-size", strconv.FormatInt(role.TensorParallel, 10))
	}
	if role.Execution == ExecutionRay {
		argv = append(argv, "--distributed-executor-backend", ExecutionRay)
	}
	if layout == LayoutPD {
		if engine == EngineVLLM {
			mode := "kv_producer"
			if role.Name == "decode" {
				mode = "kv_consumer"
			}
			kv, _ := json.Marshal(map[string]string{"kv_connector": "NixlConnector", "kv_role": mode})
			argv = append(argv, "--kv-transfer-config", string(kv))
		} else {
			argv = append(argv, "--disaggregation-mode", role.Name, "--disaggregation-transfer-backend", "mooncake")
		}
	}
	keys := make([]string, 0, len(role.EngineOptions))
	for k := range role.EngineOptions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		argv = append(argv, "--"+k)
		if role.EngineOptions[k] != "" {
			argv = append(argv, role.EngineOptions[k])
		}
	}
	gpu := resource.MustParse(role.Entry.GPU) // Role profiles are validated before rendering.
	n := gpu.Value()
	spec, _ := json.Marshal(
		map[string]any{
			"argv":      argv,
			"execution": role.Execution,
			"entry":     entry,
			"nodes":     role.NodesPerInstance,
			"gpus":      n,
			"nixl":      engine == EngineVLLM && layout == LayoutPD,
		},
	)
	return []string{"python3", "-u", "-c", launchScript, string(spec)}
}
