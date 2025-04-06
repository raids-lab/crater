package alert

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
)

type alertMgr struct {
	handler alertHandlerInterface
	err     error
}

var (
	once    sync.Once
	alerter *alertMgr
)

func GetAlertMgr() AlertInterface {
	once.Do(func() {
		alerter = initAlertMgr()
	})
	return alerter
}

func initAlertMgr() *alertMgr {
	// 初始化选择具体要使用的 alert handler
	// 目前只支持 SMTP，下一步支持 WPS Robot
	// 后续可以考虑从 Config 中进行配置
	smtpHandler, err := newSMTPAlerter()
	if err != nil {
		logutils.Log.Error("Init alert mgr error")
	}
	return &alertMgr{
		handler: smtpHandler,
		err:     err,
	}
}

func (a *alertMgr) SendVerificationCode(ctx context.Context, code string, receiver *model.UserAttribute) error {
	if a.err != nil {
		return a.err
	}

	subject := "crater邮箱验证码"
	body := fmt.Sprintf("邮箱验证码为：%s", code)
	err := a.handler.SendMessageTo(ctx, receiver, subject, body)
	if err != nil {
		return err
	}

	// TODO: 审计，留下所有发送邮件记录
	return nil
}

// Email中可能用到的Job信息
type JobInformation struct {
	Name              string
	JobName           string
	Username          string
	jobURL            string
	Receiver          model.UserAttribute
	CreationTimestamp time.Time
	RunningTimestamp  time.Time
}

func (a *alertMgr) getJobAlertInfo(ctx context.Context, jobName string) (*JobInformation, error) {
	jobDB := query.Job
	job, err := jobDB.WithContext(ctx).Where(jobDB.JobName.Eq(jobName)).Preload(jobDB.User).First()
	if err != nil {
		return nil, err
	}

	host := config.GetConfig().Host
	jobURL := fmt.Sprintf("https://%s/portal/job/batch/%s", host, job.JobName)
	if job.JobType == "jupyter" {
		jobURL = fmt.Sprintf("https://%s/portal/job/inter/%s", host, job.JobName)
	}

	receiver := job.User.Attributes.Data()

	return &JobInformation{
		Name:              jobName,
		JobName:           job.JobName,
		Username:          job.User.Attributes.Data().Nickname,
		jobURL:            jobURL,
		Receiver:          receiver,
		CreationTimestamp: job.CreationTimestamp,
		RunningTimestamp:  job.RunningTimestamp,
	}, nil
}

// Job 相关邮件
// condition 为条件函数，返回 true 则发送通知
// bodyFormatter 为邮件内容格式化函数，返回格式化后的邮件内容
func (a *alertMgr) sendJobNotification(
	ctx context.Context,
	jobName, subject string,
	condition func(info *JobInformation) bool,
	bodyFormatter func(info *JobInformation) string,
) error {
	if a.err != nil {
		return a.err
	}

	info, err := a.getJobAlertInfo(ctx, jobName)
	if err != nil {
		return err
	}

	// 如果条件不满足，则不发送通知
	if condition != nil && !condition(info) {
		return nil
	}

	body := bodyFormatter(info)
	if err := a.handler.SendMessageTo(ctx, &info.Receiver, subject, body); err != nil {
		return err
	}

	// TODO: 审计，留下所有发送邮件记录
	return nil
}

// 作业开始通知，只有当作业创建和运行间隔超过 10 分钟时才发送
func (a *alertMgr) JobRunningAlert(ctx context.Context, jobName string) error {
	return a.sendJobNotification(ctx, jobName, "作业开始通知",
		func(info *JobInformation) bool {
			timeRangeMinite := 10
			return info.RunningTimestamp.Sub(info.CreationTimestamp).Minutes() > float64(timeRangeMinite)
		},
		func(info *JobInformation) string {
			return fmt.Sprintf("用户 %s 您好：您的作业 %s (%s) 已经开始运行。\n作业链接 %s",
				info.Username, info.Name, info.JobName, info.jobURL)
		},
	)
}

// 作业失败通知
func (a *alertMgr) JobFailureAlert(ctx context.Context, jobName string) error {
	return a.sendJobNotification(ctx, jobName, "作业失败通知",
		nil, // 无需额外判断
		func(info *JobInformation) string {
			return fmt.Sprintf("用户 %s 您好：您的作业 %s (%s) 运行失败。\n作业链接 %s",
				info.Username, info.Name, info.JobName, info.jobURL)
		},
	)
}

// 作业完成通知
func (a *alertMgr) JobCompleteAlert(ctx context.Context, jobName string) error {
	return a.sendJobNotification(ctx, jobName, "作业完成通知",
		nil, // 无需额外判断
		func(info *JobInformation) string {
			return fmt.Sprintf("用户 %s 您好：您的作业 %s (%s) 运行完成。\n作业链接 %s",
				info.Username, info.Name, info.JobName, info.jobURL)
		},
	)
}

// 低利用率作业删除通知
func (a *alertMgr) DeleteJob(ctx context.Context, jobName string, _ map[string]any) error {
	return a.sendJobNotification(ctx, jobName, "作业删除通知",
		nil,
		func(info *JobInformation) string {
			return fmt.Sprintf("用户 %s 您好：您的作业 %s (%s) 利用率过低，平台已经删除该作业。\n作业链接 %s",
				info.Username, info.Name, info.JobName, info.jobURL)
		},
	)
}

// 长时间运行作业删除通知
func (a *alertMgr) CleanJob(ctx context.Context, jobName string, _ map[string]any) error {
	return a.sendJobNotification(ctx, jobName, "作业删除通知",
		nil,
		func(info *JobInformation) string {
			return fmt.Sprintf("用户 %s 您好：您的作业 %s (%s) 运行时间达到上限，平台已经删除该作业。\n作业链接 %s",
				info.Username, info.Name, info.JobName, info.jobURL)
		},
	)
}

// RemindLowUsageJob 发送低资源使用率告警
func (a *alertMgr) RemindLowUsageJob(ctx context.Context, jobName string, deleteTime time.Time, _ map[string]any) error {
	return a.sendJobNotification(ctx, jobName, "作业即将删除告警",
		nil,
		func(info *JobInformation) string {
			deleteTimeStr := deleteTime.Format("2006-01-02 15:04:05")
			return fmt.Sprintf("用户 %s 您好：您的作业 %s (%s) 申请了 GPU 资源，但资源利用率过低，平台将于 %s 删除该作业。如果有特殊需求，请联系管理员锁定作业。\n作业链接 %s",
				info.Username, info.Name, info.JobName, deleteTimeStr, info.jobURL)
		},
	)
}

// RemindLongTimeRunningJob 发送长时间运行告警
func (a *alertMgr) RemindLongTimeRunningJob(ctx context.Context, jobName string, deleteTime time.Time, _ map[string]any) error {
	return a.sendJobNotification(ctx, jobName, "作业即将删除告警",
		nil,
		func(info *JobInformation) string {
			deleteTimeStr := deleteTime.Format("2006-01-02 15:04:05")
			return fmt.Sprintf("用户 %s 您好：您的作业 %s (%s) 运行时间较长，平台将于 %s 删除该作业。如果有特殊需求，请联系管理员锁定作业。\n作业链接 %s",
				info.Username, info.Name, info.JobName, deleteTimeStr, info.jobURL)
		},
	)
}
