package payload

import "time"

type StatisticsScope string

const (
	ScopeUser    StatisticsScope = "user"
	ScopeAccount StatisticsScope = "account"
	ScopeCluster StatisticsScope = "cluster"
)

type TimeStep string

const (
	StepDay  TimeStep = "day"
	StepWeek TimeStep = "week"
)

// StatisticsReq 统计请求参数
type StatisticsReq struct {
	StartTime time.Time       `json:"startTime" form:"startTime" binding:"required"`
	EndTime   time.Time       `json:"endTime" form:"endTime" binding:"required"`
	Step      TimeStep        `json:"step" form:"step,default=day" binding:"oneof=day week"`   // 聚合粒度
	Scope     StatisticsScope `json:"scope" form:"scope" binding:"oneof=user account cluster"` // 统计范围
	TargetID  uint            `json:"targetID" form:"targetID"`                                // 用户ID或账户ID，Cluster模式下忽略
}

// ResourceUsage 资源用量 (单位：核时/卡时/GiB时)
type ResourceUsage map[string]float64

// ResourceDetail 资源详细统计信息
type ResourceDetail struct {
	Usage float64 `json:"usage"` // 分配用量 (核时/卡时/GiB时)
	Label string  `json:"label"` // 显示名称 (例如 "NVIDIA V100", "CPU", "内存")
	Type  string  `json:"type"`  // 资源类型 (gpu, vgpu, rdma, common)
	*ResourceProfileDetail
}

// ResourceProfileDetail 资源利用率与有效用量统计。
type ResourceProfileDetail struct {
	EffectiveUsage float64  `json:"effectiveUsage"`                      // 估算有效用量 (核时/卡时/GiB时)
	ProfiledUsage  float64  `json:"profiledUsage"`                       // 有监控摘要的分配用量 (核时/卡时/GiB时)
	Utilization    *float64 `json:"utilization" extensions:"x-nullable"` // 加权平均利用率
	Coverage       *float64 `json:"coverage" extensions:"x-nullable"`    // 评估覆盖率
}

// TimePointData 时间点数据 (趋势图保持轻量，只返回数值)
type TimePointData struct {
	Timestamp time.Time          `json:"timestamp" format:"date-time"`            // 时间桶起始时间 (UTC)
	Status    string             `json:"status" enums:"complete,partial,pending"` // 统计完成状态 (complete/partial/pending)
	Usage     map[string]float64 `json:"usage"`                                   // 分配用量 (Key: ResourceName)
	*TimePointProfile
}

// TimePointProfile 时间点有效用量。
type TimePointProfile struct {
	EffectiveUsage map[string]float64 `json:"effectiveUsage"` // 估算有效用量 (Key: ResourceName)
}

// StatisticsProgress 每日统计任务聚合进度
type StatisticsProgress struct {
	Ready            bool       `json:"ready"`                                                   // 是否已完成至少一轮可用统计
	FirstDate        string     `json:"firstDate"`                                               // 最早统计日期 (YYYY-MM-DD)
	CompleteBefore   string     `json:"completeBefore"`                                          // 该日期之前均已完成 (YYYY-MM-DD)
	CoveredUntil     *time.Time `json:"coveredUntil" format:"date-time" extensions:"x-nullable"` // 统计数据覆盖截止时间
	UpdatedAt        *time.Time `json:"updatedAt" format:"date-time" extensions:"x-nullable"`    // 最近更新时间
	PendingDays      int64      `json:"pendingDays"`                                             // 待补算天数
	BackfillComplete bool       `json:"backfillComplete"`                                        // 历史回填是否全部完成
}

// StatisticsResp 响应结构
type StatisticsResp struct {
	// TotalUsage Key: ResourceName (如 nvidia.com/v100)
	// Value: 包含 Label 和 Type 的详细对象
	TotalUsage map[string]ResourceDetail `json:"totalUsage"` // 汇总资源用量
	Series     []TimePointData           `json:"series"`     // 时间序列数据
	Progress   StatisticsProgress        `json:"progress"`   // 统计聚合进度
}

// StatisticsRankingReq 资源使用排行请求
type StatisticsRankingReq struct {
	StatisticsReq
	Dimension    string `json:"dimension" form:"dimension" binding:"required,oneof=user account"`               // 排名维度
	Metric       string `json:"metric" form:"metric" binding:"required,oneof=usage effectiveUsage utilization"` // 排序指标
	Order        string `json:"order" form:"order" binding:"required,oneof=asc desc"`                           // 排序方向
	ResourceName string `json:"resourceName" form:"resourceName"`                                               // 资源名称
	ResourceType string `json:"resourceType" form:"resourceType" binding:"omitempty,oneof=gpu"`                 // 资源类型
	Page         int    `json:"page" form:"page,default=1" binding:"min=1"`                                     // 页码
	PageSize     int    `json:"pageSize" form:"pageSize,default=20" binding:"min=1,max=100"`                    // 每页条数
}

// StatisticsRankingRow 资源使用排行单项
type StatisticsRankingRow struct {
	Rank           int64    `json:"rank"`                                // 名次
	ID             uint     `json:"id"`                                  // 用户ID或账户ID
	Name           string   `json:"name"`                                // 用户名或账户名
	Nickname       string   `json:"nickname"`                            // 昵称
	Usage          float64  `json:"usage"`                               // 分配用量
	EffectiveUsage float64  `json:"effectiveUsage"`                      // 估算有效用量
	ProfiledUsage  float64  `json:"profiledUsage"`                       // 已评估分配用量
	Utilization    *float64 `json:"utilization" extensions:"x-nullable"` // 加权平均利用率
	Coverage       *float64 `json:"coverage" extensions:"x-nullable"`    // 评估覆盖率
}

// StatisticsRankingResp 资源使用排行响应
type StatisticsRankingResp struct {
	Rows     []StatisticsRankingRow `json:"rows"`     // 排行列表
	Total    int64                  `json:"total"`    // 总条数
	Progress StatisticsProgress     `json:"progress"` // 统计聚合进度
}
