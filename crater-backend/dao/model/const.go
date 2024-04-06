package model

// User role in platform and project
type Role uint8

const (
	RoleGuest Role = iota
	RoleUser
	RoleAdmin
)

// Project and user status
type Status uint8

const (
	StatusPending  Status = iota // Pending status, not yet activated
	StatusActive                 // Active status
	StatusInactive               // Inactive status
)

// Space access mode (read-write, read-only)
type AccessMode uint8

const (
	AccessModeRO AccessMode = iota // Read-only mode
	AccessModeAO                   // Append-only mode
	AccessModeRW                   // Read-write mode
)

// Job status
type JobStatus uint8

const (
	JobInitial   JobStatus = iota // 初始状态，未进行或未通过配额检查
	JobCreated                    // 作业已通过配额检查，提交到集群中，等待调度
	JobRunning                    // 作业正在运行
	JobSucceeded                  // 作业的所有 Pod 均成功完成
	JobFailed                     // 作业中的一个或多个 Pod 失败
	JobPreempted                  // 作业中的一个或多个 Pod 被抢占
)
