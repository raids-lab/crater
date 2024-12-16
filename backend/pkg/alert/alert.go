package alert

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raids-lab/crater/dao/query"
)

type alertMgr struct {
	handler alertHandlerInterface
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
		panic(err)
	}
	return &alertMgr{
		handler: smtpHandler,
	}
}

func (a *alertMgr) sendJobMessage(ctx context.Context, jobName, subject, messageTemplate string) error {
	j := query.Job
	job, err := j.WithContext(ctx).Where(j.JobName.Eq(jobName)).Preload(j.User).First()
	if err != nil {
		return err
	}
	receiver := job.User.Attributes.Data()
	body := fmt.Sprintf(messageTemplate, receiver.Nickname, job.Name, job.JobName)

	err = a.handler.SendMessageTo(ctx, &receiver, subject, body)
	if err != nil {
		return err
	}

	// TODO: 审计，留下所有发送邮件记录
	return nil
}

func (a *alertMgr) DeleteJob(ctx context.Context, jobName string, _ map[string]any) error {
	subject := "作业删除通知"
	messageTemplate := `用户 %s 您好：您的作业 %s (%s) 申请了 GPU 资源，但资源利用率过低，平台已经删除该作业。`
	return a.sendJobMessage(ctx, jobName, subject, messageTemplate)
}

func (a *alertMgr) RemindLowUsageJob(ctx context.Context, jobName string, deleteTime time.Time, _ map[string]any) error {
	subject := "作业即将删除告警"
	deleteTimeStr := deleteTime.Format("2006-01-02 15:04:05")
	messageTemplate := `用户 %s 您好：您的作业 %s (%s) 申请了 GPU 资源，但资源利用率过低，平台将于 %s 删除该作业。`
	message := fmt.Sprintf(messageTemplate, "%s", "%s", "%s", deleteTimeStr)
	return a.sendJobMessage(ctx, jobName, subject, message)
}
