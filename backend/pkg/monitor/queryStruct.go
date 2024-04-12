package monitor

type PodGPUAllocate struct {
	Node     string `json:"node"`
	Instance string `json:"instance"`
	Pod      string `json:"pod"`
	GPUCount int    `json:"GPU_count"`
}
