// 请将此文件保存为 dao/model/gpu_analysis.go

package model

import (
	"time"

	"gorm.io/gorm" // 必须引入 gorm
)

type GpuAnalysis struct {
	ID uint `gorm:"primarykey"`

	// 自动追踪的时间戳
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index;comment:软删除时间戳"`

	// --- 核心关联信息 ---
	// 我们存储ID用于程序逻辑（如跳转），并冗余存储名称用于UI显示。
	// 这样即使Job被删除，我们依然知道是哪个Job、哪个用户的。

	JobID    uint   `gorm:"index;comment:关联的作业ID，用于跳转，但不设外键约束"`
	JobName  string `gorm:"index;comment:关联的作业名称 (冗余字段，用于显示)"`
	UserID   uint   `gorm:"index;comment:关联的用户ID，用于统计"`
	UserName string `gorm:"comment:提交作业的用户名 (冗余字段，用于显示)"`

	// 原始 Kubernetes 信息
	PodName   string `gorm:"index;comment:被分析的Pod名称"`
	Namespace string `gorm:"comment:Pod所在的命名空间"`

	// LLM 分析结果
	Phase1Score     int    `gorm:"comment:基于监控数据的初步评分"`
	Phase2Score     int    `gorm:"comment:结合脚本内容的二次评分"`
	Phase1LLMReason string `gorm:"type:text;comment:LLM给出的初步分析理由"`
	Phase2LLMReason string `gorm:"type:text;comment:LLM给出的二次分析理由"`
	LLMVersion      string `gorm:"size:100;comment:使用的LLM模型版本"`

	// 采集到的原始数据
	Command           string `gorm:"type:text;comment:GPU进程的启动命令"`
	ScriptContent     string `gorm:"type:text;comment:获取到的脚本内容" json:"-"`
	HistoricalMetrics string `gorm:"type:text;comment:用于分析的历史指标摘要(JSON格式)"`

	// 管理状态
	ReviewStatus ReviewStatus `gorm:"default:1;comment:审核状态 (1: Pending, 2: Confirmed, 3: Ignored)"`
}
