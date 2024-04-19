package monitor

type PodGPUAllocate struct {
	Node     string `json:"node"`
	Instance string `json:"instance"`
	Pod      string `json:"pod"`
	GPUCount int    `json:"GPU_count"`
}

type NodeGPUUtil struct {
	Hostname  string  `json:"hostname"`
	UUID      string  `json:"uuid"`
	Container string  `json:"container"`
	Device    string  `json:"device"`
	Endpoint  string  `json:"endpoint"`
	Gpu       string  `json:"gpu"`
	Instance  string  `json:"instance"`
	Job       string  `json:"job"`
	ModelName string  `json:"model_name"`
	Namespace string  `json:"namespace"`
	Pod       string  `json:"pod"`
	Service   string  `json:"service"`
	Util      float32 `json:"util"`
}
