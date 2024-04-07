// 定义与数据库表字段对应的常量
// 由于 Gin 框架在进行参数校验时，如果给了 required 标签，则不能传入零值
// 这个时候，我们可以通过定义对应类型的指针解决该问题，但这可能导致出错
// 所以在定义常量时，最好将零值排除在外，请使用 iota + 1 定义第一个常量
package model

// User role in platform and project
type Role uint8

const (
	RoleGuest Role = iota + 1
	RoleUser
	RoleAdmin
)

// Project and user status
type Status uint8

const (
	StatusPending  Status = iota + 1 // Pending status, not yet activated
	StatusActive                     // Active status
	StatusInactive                   // Inactive status
)

// Space access mode (read-only, append-only, read-write)
type AccessMode uint8

const (
	AccessModeRO AccessMode = iota + 1 // Read-only mode
	AccessModeAO                       // Append-only mode
	AccessModeRW                       // Read-write mode
)

// Job status
type JobStatus uint8

const (
	JobInitial   JobStatus = iota + 1 // 初始状态，未进行或未通过配额检查
	JobCreated                        // 作业已通过配额检查，提交到集群中，等待调度
	JobRunning                        // 作业正在运行
	JobSucceeded                      // 作业的所有 Pod 均成功完成
	JobFailed                         // 作业中的一个或多个 Pod 失败
	JobPreempted                      // 作业中的一个或多个 Pod 被抢占
)
