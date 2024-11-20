package alert

import (
	"context"
	"fmt"
	"sync"

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

func (a *alertMgr) JobFreed(ctx context.Context, jobName string, _ map[string]any) error {
	j := query.Job
	job, err := j.WithContext(ctx).Where(j.JobName.Eq(jobName)).Preload(j.User).First()
	if err != nil {
		return err
	}
	receiver := job.User.Attributes.Data()
	subject := "作业删除告警"
	body := fmt.Sprintf(`用户 %s 您好：您的作业 %s (%s) 申请了 GPU 资源，但资源利用率过低，平台即将删除该作业。`,
		receiver.Nickname, job.Name, job.JobName)
	err = a.handler.SendMessageTo(ctx, &receiver, subject, body)
	if err != nil {
		return err
	}

	// TODO: 审计，留下所有发送邮件记录
	return nil
}
