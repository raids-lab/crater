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

// ListPageQuery 分页请求基类（弱校验，可直接 form 绑定）。
// 与 ListReqQuery 的区别：所有字段都有默认值，缺省时不会报 binding:"required" 错误，
// 适合作为列表接口的可选嵌入字段。
type ListPageQuery struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	SortBy   string `form:"sort_by"`
	Order    Order  `form:"order"`
}

// MaxListPageSize 列表接口允许的单页最大条数，超过会被截断到此值。
const MaxListPageSize = 200

// Normalize 返回经过校验/截断后的 (offset, limit) 与 order 默认值。
//
// 规则：
//   - Page < 1 视为 1
//   - PageSize < 1 视为默认 20，> MaxListPageSize 截断到 MaxListPageSize
//   - Order 非 asc/desc 时返回 Desc
func (q ListPageQuery) Normalize() (offset, limit int, order Order) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size < 1 {
		size = 20
	}
	if size > MaxListPageSize {
		size = MaxListPageSize
	}
	ord := q.Order
	if ord != Asc && ord != Desc {
		ord = Desc
	}
	return (page - 1) * size, size, ord
}

// IsPagingRequested reports whether the caller explicitly opted into pagination
// by setting page or page_size. When both are zero, the handler should return
// the full result set (preserving backwards-compatible "list all" behavior).
func (q ListPageQuery) IsPagingRequested() bool {
	return q.Page != 0 || q.PageSize != 0
}
