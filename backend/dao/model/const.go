// 定义与数据库表字段对应的常量
// 由于 Gin 框架在进行参数校验时，如果给了 required 标签，则不能传入零值
// 这个时候，我们可以通过定义对应类型的指针解决该问题，但这可能导致出错
// 所以在定义常量时，最好将零值排除在外
package model

// User role in platform and project
type Role uint8

const (
	_ Role = iota
	RoleGuest
	RoleUser
	RoleAdmin
)

// Project and user status
type Status uint8

const (
	_              Status = iota
	StatusPending         // Pending status, not yet activated
	StatusActive          // Active status
	StatusInactive        // Inactive status
)

// Space access mode (read-only, append-only, read-write)
type AccessMode uint8

const (
	_            AccessMode = iota
	AccessModeNA            // Not-allowed mode
	AccessModeRO            // Read-only mode
	AccessModeRW            // Read-write mode
	AccessModeAO            // Append-only mode
)

// Job status
type JobStatus uint8

const (
	_            JobStatus = iota
	JobInitial             // 初始状态，未进行或未通过配额检查
	JobCreated             // 作业已通过配额检查，提交到集群中，等待调度
	JobRunning             // 作业正在运行
	JobSucceeded           // 作业的所有 Pod 均成功完成
	JobFailed              // 作业中的一个或多个 Pod 失败
	JobPreempted           // 作业中的一个或多个 Pod 被抢占
)

type ImageTaskType uint8

const (
	_              ImageTaskType = iota
	JupyterTask                  // Jupyter交互式任务
	WebIDETask                   // Web IDE任务
	TensorflowTask               // Tensorflow任务
	PytorchTask                  // Pytorch任务
	RayTask                      // Ray任务
	DeepSpeedTask                // DeepSpeed任务
	OpenMPITask                  // OpenMPI任务
	UserDefineTask               // 用户自定义任务
)

type WorkerType uint8

const (
	_       WorkerType = iota
	Nvidia             // Nvidia GPU worker
	Enflame            // Enflame AI worker
	Unknown            // Unknown worker
)

type ImageSourceType uint8

const (
	_               ImageSourceType = iota
	ImageCreateType                 // 镜像制造
	ImageUploadType                 // 镜像上传
)

type AlertType uint8

const (
	_                        AlertType = iota
	JobRunningAlert                    // 作业开始通知
	JobFailedAlert                     // 作业失败通知
	JobCompletedAlert                  // 作业完成通知
	LowGPUJobRemindedAlert             // 低GPU利用率作业提醒通知
	LowGPUJobDeletedAlert              // 低GPU利用率作业删除通知
	LongTimeJobRemindedAlert           // 长时间作业提醒通知
	LongTimeJobDeletedAlert            // 长时间作业删除通知
)

//go:generate stringer -type=Role,Status,AccessMode,JobStatus,ImageTaskType,WorkerType,ImageSourceType,AlertType -output=const_string.go
