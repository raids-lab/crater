package alert

import (
	"context"
	"time"

	"github.com/raids-lab/crater/dao/model"
)

// AlertMgr 是封装好的通知组件，提供：
// 支持四种初步场景：
//  1. 作业开始通知（如果创建时间与开始时间间隔 > 10min）
//  2. 作业成功通知
//  3. 作业失败通知
//  3. 作业因低利用率即将被释放通知
//  4. 作业因低利用率已经被释放通知
//  5. 作业异常的资源使用警告
//  6. 发送邮箱验证码
type AlertInterface interface {
	JobRunningAlert(ctx context.Context, jobName string) error
	JobFailureAlert(ctx context.Context, jobName string) error
	DeleteJob(ctx context.Context, jobName string, extra map[string]any) error
	RemindLowUsageJob(ctx context.Context, jobName string, deleteTime time.Time, extra map[string]any) error
	SendVerificationCode(ctx context.Context, code string, receiver *model.UserAttribute) error
}

// alertHandlerInterface 是具体的通知组件对外部提供的接口，WPS Robot 或者 SMTP 邮件通知都应该实现这两个接口
type alertHandlerInterface interface {
	SendMessageTo(ctx context.Context, receiver *model.UserAttribute, subject, body string) error
}
