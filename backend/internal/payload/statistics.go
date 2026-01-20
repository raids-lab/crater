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
	Step      TimeStep        `json:"step" form:"step" binding:"oneof=day week"`               // 聚合粒度
	Scope     StatisticsScope `json:"scope" form:"scope" binding:"oneof=user account cluster"` // 统计范围
	TargetID  uint            `json:"targetID" form:"targetID"`                                // 用户ID或账户ID，Cluster模式下忽略
}

// ResourceUsage 资源用量 (单位：核时/卡时/GiB时)
type ResourceUsage map[string]float64

// ResourceDetail 资源详细统计信息
type ResourceDetail struct {
	Usage float64 `json:"usage"` // 用量 (核时/卡时/GiB时)
	Label string  `json:"label"` // 显示名称 (例如 "NVIDIA V100", "CPU", "内存")
	Type  string  `json:"type"`  // 资源类型 (gpu, vgpu, rdma, common)
}

// TimePointData 时间点数据 (趋势图保持轻量，只返回数值)
type TimePointData struct {
	Timestamp time.Time          `json:"timestamp"`
	Usage     map[string]float64 `json:"usage"` // Key: ResourceName, Value: Usage
}

// StatisticsResp 响应结构
type StatisticsResp struct {
	// TotalUsage Key: ResourceName (如 nvidia.com/v100)
	// Value: 包含 Label 和 Type 的详细对象
	TotalUsage map[string]ResourceDetail `json:"totalUsage"`

	Series []TimePointData `json:"series"`
}
