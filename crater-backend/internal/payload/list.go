package payload

// 排序顺序常量
type Order string

const (
	Asc  Order = "asc"
	Desc Order = "desc"
)

// 分页请求统一接口
type (
	// ListReqQuery 分页请求参数（从 query 中获取）
	// 如果需要包含其他参数，不能通过组合的方式，需要直接定义在结构体中（否则无法通过 Gin 校验）
	// 示例 API 见 get /admin/projects
	ListReqQuery struct {
		PageIndex *int `form:"page_index" binding:"required"`
		PageSize  *int `form:"page_size" binding:"required"`
	}
	ListResp[T any] struct {
		Rows  []T   `json:"rows"`
		Count int64 `json:"count"`
	}
)
