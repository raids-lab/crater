package payload

// 定义返回值时，优先在使用到该返回值的 /internal/handler/xxx.go 中直接定义
// 当某个返回值的结构体通用时，从 /internal/handler/xxx.go 中提升至此文件中

type (
	ResourceBase struct {
		Amount int64  `json:"amount"`
		Format string `json:"format"`
	}

	ResourceResp struct {
		Label      string        `json:"label"`
		Allocated  *ResourceBase `json:"allocated"`
		Guarantee  *ResourceBase `json:"guarantee"`
		Deserved   *ResourceBase `json:"deserved"`
		Capability *ResourceBase `json:"capability"`
	}

	QuotaResp struct {
		CPU    ResourceResp   `json:"cpu"`
		Memory ResourceResp   `json:"memory"`
		GPUs   []ResourceResp `json:"gpus"`
	}
)
