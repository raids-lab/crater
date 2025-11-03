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

type ProfileData struct {
	// CPU and Memory
	CPURequest *float32 `json:"cpu_request,omitempty"`
	CPULimit   *float32 `json:"cpu_limit,omitempty"`
	MemRequest *float32 `json:"mem_request,omitempty"`
	MemLimit   *float32 `json:"mem_limit,omitempty"`

	CPUUsageAvg *float32 `json:"cpu_usage_avg,omitempty"`
	CPUUsageMax *float32 `json:"cpu_usage_max,omitempty"`
	CPUUsageStd *float32 `json:"cpu_usage_std,omitempty"`

	CPUMemAvg *float32 `json:"cpu_mem_avg,omitempty"`
	CPUMemMax *float32 `json:"cpu_mem_max,omitempty"`
	CPUMemStd *float32 `json:"cpu_mem_std,omitempty"`

	// GPU
	GPUUtilAvg *float32 `json:"gpu_util_avg,omitempty"`
	GPUUtilMax *float32 `json:"gpu_util_max,omitempty"`
	GPUUtilStd *float32 `json:"gpu_util_std,omitempty"`

	SMActiveAvg *float32 `json:"sm_active_avg,omitempty"`
	SMActiveMax *float32 `json:"sm_active_max,omitempty"`
	SMActiveStd *float32 `json:"sm_active_std,omitempty"`

	SMOccupancyAvg *float32 `json:"sm_occupancy_avg,omitempty"`
	SMOccupancyMax *float32 `json:"sm_occupancy_max,omitempty"`
	SMOccupancyStd *float32 `json:"sm_occupancy_std,omitempty"`

	DramUtilAvg *float32 `json:"dram_util_avg,omitempty"`
	DramUtilMax *float32 `json:"dram_util_max,omitempty"`
	DramUtilStd *float32 `json:"dram_util_std,omitempty"`

	MemCopyUtilAvg *float32 `json:"mem_copy_util_avg,omitempty"`
	MemCopyUtilMax *float32 `json:"mem_copy_util_max,omitempty"`
	MemCopyUtilStd *float32 `json:"mem_copy_util_std,omitempty"`

	PCIETxAvg *float32 `json:"pcie_tx_avg,omitempty"`
	PCIETxMax *float32 `json:"pcie_tx_max,omitempty"`

	PCIERxAvg *float32 `json:"pcie_rx_avg,omitempty"`
	PCIERxMax *float32 `json:"pcie_rx_max,omitempty"`

	GPUMemTotal *float32 `json:"gpu_mem_total,omitempty"`
	GPUMemMax   *float32 `json:"gpu_mem_max,omitempty"`
	GPUMemAvg   *float32 `json:"gpu_mem_avg,omitempty"`
	GPUMemStd   *float32 `json:"gpu_mem_std,omitempty"`

	TensorActiveAvg *float32 `json:"tensor_active_avg,omitempty"`
	TensorActiveMax *float32 `json:"tensor_active_max,omitempty"`
	TensorActiveStd *float32 `json:"tensor_active_std,omitempty"`

	Fp64ActiveAvg *float32 `json:"fp64_active_avg,omitempty"`
	Fp64ActiveMax *float32 `json:"fp64_active_max,omitempty"`
	Fp64ActiveStd *float32 `json:"fp64_active_std,omitempty"`

	Fp32ActiveAvg *float32 `json:"fp32_active_avg,omitempty"`
	Fp32ActiveMax *float32 `json:"fp32_active_max,omitempty"`
	Fp32ActiveStd *float32 `json:"fp32_active_std,omitempty"`

	DramActiveAvg *float32 `json:"dram_active_avg,omitempty"`
	DramActiveMax *float32 `json:"dram_active_max,omitempty"`
	DramActiveStd *float32 `json:"dram_active_std,omitempty"`

	Fp16ActiveAvg *float32 `json:"fp16_active_avg,omitempty"`
	Fp16ActiveMax *float32 `json:"fp16_active_max,omitempty"`
	Fp16ActiveStd *float32 `json:"fp16_active_std,omitempty"`
}

type PodUtil struct {
	GPUUtilAvg     float32 `json:"gpu_util_avg"`
	GPUUtilMax     float32 `json:"gpu_util_max"`
	GPUUtilStd     float32 `json:"gpu_util_std"`
	SMActiveAvg    float32 `json:"sm_active_avg"`
	SMActiveMax    float32 `json:"sm_active_max"`
	SMActiveStd    float32 `json:"sm_active_std"`
	SMOccupancyAvg float32 `json:"sm_occupancy_avg"`
	SMOccupancyMax float32 `json:"sm_occupancy_max"`
	SMOccupancyStd float32 `json:"sm_occupancy_std"`
	DramUtilAvg    float32 `json:"dram_util_avg"`
	DramUtilMax    float32 `json:"dram_util_max"`
	DramUtilStd    float32 `json:"dram_util_std"`
	MemCopyUtilAvg float32 `json:"mem_copy_util_avg"`
	MemCopyUtilMax float32 `json:"mem_copy_util_max"`
	MemCopyUtilStd float32 `json:"mem_copy_util_std"`
	PCIETxAvg      float32 `json:"pcie_tx_avg"`
	PCIETxMax      float32 `json:"pcie_tx_max"`
	PCIERxAvg      float32 `json:"pcie_rx_avg"`
	PCIERxMax      float32 `json:"pcie_rx_max"`
	CPUUsageAvg    float32 `json:"cpu_usage_avg"`
	GPUMemMax      float32 `json:"gpu_mem_max"`
	CPUMemMax      float32 `json:"cpu_mem_max"`
}
