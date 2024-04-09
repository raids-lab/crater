package payload

type Quota struct {
	JobReq int `json:"jobReq"`
	Job    int `json:"job"`

	NodeReq int `json:"nodeReq"`
	Node    int `json:"node"`

	CPUReq int `json:"cpuReq"`
	CPU    int `json:"cpu"`

	GPUReq int `json:"gpuReq"`
	GPU    int `json:"gpu"`

	MemReq int `json:"memReq"`
	Mem    int `json:"mem"`

	GPUMemReq int `json:"gpuMemReq"`
	GPUMem    int `json:"gpuMem"`

	Storage int     `json:"storage"`
	Extra   *string `json:"extra"`
}
