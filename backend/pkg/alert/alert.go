//nolint:lll // Package alert 提供了一个统一的接口来发送不同类型的告警通知
package alert

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
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
		klog.Error("Init alert mgr error")
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

	subject := "邮箱验证码"
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px; color: #333;">
			<h2 style="color: #2c3e50;">Crater 邮箱验证</h2>
			<p>您好，</p>
			<p>您的邮箱验证码为：</p>
			<div style="background-color: #f8f9fa; padding: 10px; border-radius: 5px; font-size: 18px; font-weight: bold; text-align: center; letter-spacing: 2px;">
				%s
			</div>
			<p style="font-size: 12px; color: #7f8c8d; margin-top: 20px;">该验证码有效期为10分钟，请勿将验证码泄露给他人。</p>
		</div>
	`, code)

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
		Name:              job.Name,
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
	alertType model.AlertType,
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

	// 检查是否已发送过
	alertDB := query.Alert
	record, alertErr := alertDB.WithContext(ctx).Where(alertDB.JobName.Eq(jobName), alertDB.AlertType.Eq(alertType.String())).First()

	if alertErr == nil && !record.AllowRepeat {
		// 该job该type的邮件已经发送过，且不允许再发送
		klog.Infof("job %s type %s already sent", jobName, alertType.String())
		return nil
	}

	if alertErr != nil && !errors.Is(alertErr, gorm.ErrRecordNotFound) {
		// 发生错误
		return alertErr
	}

	body := bodyFormatter(info)
	if err := a.handler.SendMessageTo(ctx, &info.Receiver, subject, body); err != nil {
		return err
	}

	// 审计，留下所有发送邮件记录
	if alertErr != nil && errors.Is(alertErr, gorm.ErrRecordNotFound) {
		// 1. 邮件没发送过，创建新纪录
		newRecord := &model.Alert{
			JobName:        jobName,
			AlertType:      alertType.String(),
			AllowRepeat:    false,
			AlertTimestamp: utils.GetLocalTime(),
			SendCount:      1,
		}
		if err := alertDB.WithContext(ctx).Create(newRecord); err != nil {
			return err
		}
	} else {
		// 2. 邮件已经发送过，更新记录
		record.SendCount++
		record.AlertTimestamp = utils.GetLocalTime()
		record.AllowRepeat = false
		if err := alertDB.WithContext(ctx).Save(record); err != nil {
			return err
		}
	}

	return nil
}

// 作业开始通知，只有当作业创建和运行间隔超过 10 分钟时才发送
func (a *alertMgr) JobRunningAlert(ctx context.Context, jobName string) error {
	return a.sendJobNotification(ctx, jobName, "作业已开始运行", model.JobRunningAlert,
		func(info *JobInformation) bool {
			timeRangeMinite := 10
			return info.RunningTimestamp.Sub(info.CreationTimestamp).Minutes() > float64(timeRangeMinite)
		},
		func(info *JobInformation) string {
			return generateHTMLEmail(
				info.Username,
				"作业已开始运行",
				fmt.Sprintf("您的作业 <strong>%s</strong> (ID: %s) 已开始运行。", info.Name, info.JobName),
				info.jobURL,
				"查看作业详情",
			)
		},
	)
}

// 作业失败通知
func (a *alertMgr) JobFailureAlert(ctx context.Context, jobName string) error {
	return a.sendJobNotification(ctx, jobName, "作业运行失败", model.JobFailedAlert,
		nil, // 无需额外判断
		func(info *JobInformation) string {
			return generateHTMLEmail(
				info.Username,
				"作业运行失败",
				fmt.Sprintf("您的作业 <strong>%s</strong> (ID: %s) 运行失败。请查看日志了解详细信息。", info.Name, info.JobName),
				info.jobURL,
				"查看失败详情",
			)
		},
	)
}

// 作业完成通知
func (a *alertMgr) JobCompleteAlert(ctx context.Context, jobName string) error {
	return a.sendJobNotification(ctx, jobName, "作业已成功完成", model.JobCompletedAlert,
		nil, // 无需额外判断
		func(info *JobInformation) string {
			return generateHTMLEmail(
				info.Username,
				"作业已成功完成",
				fmt.Sprintf("您的作业 <strong>%s</strong> (ID: %s) 已成功运行完成。", info.Name, info.JobName),
				info.jobURL,
				"查看作业结果",
			)
		},
	)
}

// 低GPU利用率作业删除通知
func (a *alertMgr) DeleteJob(ctx context.Context, jobName string, _ map[string]any) error {
	return a.sendJobNotification(ctx, jobName, "作业已被系统删除 - GPU利用率过低", model.LowGPUJobDeletedAlert,
		nil,
		func(info *JobInformation) string {
			return generateHTMLEmail(
				info.Username,
				"作业已被系统删除",
				fmt.Sprintf("您的作业 <strong>%s</strong> (ID: %s) 因GPU利用率持续过低，已被系统自动删除。请确保您的作业能够充分利用申请的GPU资源，或调整资源申请量以匹配实际需求。", info.Name, info.JobName),
				info.jobURL,
				"查看作业详情",
			)
		},
	)
}

// 长时间运行作业删除通知
func (a *alertMgr) CleanJob(ctx context.Context, jobName string, _ map[string]any) error {
	return a.sendJobNotification(ctx, jobName, "作业已被系统删除 - 运行时间超限", model.LongTimeJobDeletedAlert,
		nil,
		func(info *JobInformation) string {
			return generateHTMLEmail(
				info.Username,
				"作业已被系统删除",
				fmt.Sprintf("您的作业 <strong>%s</strong> (ID: %s) 因运行时间达到平台上限，已被系统自动删除。如需长时间运行作业，请联系管理员申请特殊权限。", info.Name, info.JobName),
				info.jobURL,
				"查看作业详情",
			)
		},
	)
}

// RemindLowUsageJob 发送低资源使用率告警
func (a *alertMgr) RemindLowUsageJob(ctx context.Context, jobName string, deleteTime time.Time, _ map[string]any) error {
	return a.sendJobNotification(ctx, jobName, "警告：作业即将被删除 - GPU利用率过低", model.LowGPUJobRemindedAlert,
		nil,
		func(info *JobInformation) string {
			deleteTimeStr := deleteTime.Format("2006-01-02 15:04:05")
			return generateHTMLEmail(
				info.Username,
				"警告：作业即将被删除",
				fmt.Sprintf("您的作业 <strong>%s</strong> (ID: %s) 申请了GPU资源，但资源利用率持续过低。<br><br><strong style='color: #e74c3c;'>系统将于 %s 自动删除该作业</strong>。<br><br>如有特殊需求，请及时联系管理员锁定作业或调整您的作业以提高资源利用率。",
					info.Name, info.JobName, deleteTimeStr),
				info.jobURL,
				"立即查看作业",
			)
		},
	)
}

// RemindLongTimeRunningJob 发送长时间运行告警
func (a *alertMgr) RemindLongTimeRunningJob(ctx context.Context, jobName string, deleteTime time.Time, _ map[string]any) error {
	return a.sendJobNotification(ctx, jobName, "警告：作业即将被删除 - 运行时间过长", model.LongTimeJobRemindedAlert,
		nil,
		func(info *JobInformation) string {
			deleteTimeStr := deleteTime.Format("2006-01-02 15:04:05")
			return generateHTMLEmail(
				info.Username,
				"警告：作业即将被删除",
				fmt.Sprintf("您的作业 <strong>%s</strong> (ID: %s) 已运行较长时间，达到了平台设定的运行时间上限。<br><br><strong style='color: #e74c3c;'>系统将于 %s 自动删除该作业</strong>。<br><br>如有特殊需求，请及时联系管理员锁定作业或考虑对作业进行优化以减少运行时间。",
					info.Name, info.JobName, deleteTimeStr),
				info.jobURL,
				"立即查看作业",
			)
		},
	)
}

// 生成HTML格式的邮件内容
func generateHTMLEmail(username, title, message, url, buttonText string) string {
	return fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #333; border: 1px solid #e0e0e0; border-radius: 5px;">
			<h2 style="color: #2c3e50; border-bottom: 1px solid #eee; padding-bottom: 10px;">%s</h2>
			<p>尊敬的 <strong>%s</strong>：</p>
			<p>%s</p>
			<div style="margin: 25px 0;">
				<a href="%s" style="background-color: #3498db; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px; display: inline-block; font-weight: bold;">%s</a>
			</div>
			<p style="margin-top: 30px; font-size: 12px; color: #7f8c8d;">此邮件由系统自动发送，请勿直接回复。如有疑问，请联系系统管理员。</p>
			<div style="margin-top: 20px; padding-top: 15px; border-top: 1px solid #eee; font-size: 12px; color: #95a5a6; text-align: center;">
				© Crater 计算平台
			</div>
		</div>
	`, title, username, message, url, buttonText)
}
