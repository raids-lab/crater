package cleaner

import (
	"context"
	"errors"
	"time"

	"github.com/samber/lo"
	"k8s.io/klog/v2"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/alert"
	"github.com/raids-lab/crater/pkg/utils"
)

type CleanLongTimeRunningJobsRequest struct {
	BatchDays       *int `form:"batchDays"`
	InteractiveDays *int `form:"interactiveDays"`
}

func CleanLongTimeRunningJobs(c context.Context, clients *Clients, req *CleanLongTimeRunningJobsRequest) (map[string][]string, error) {
	if req == nil {
		err := errors.New("invalid request")
		return nil, err
	}

	batchJobTimeout := 4 * 24 * time.Hour
	interactiveJobTimeout := 24 * time.Hour
	if req.BatchDays != nil {
		batchJobTimeout = time.Duration(*req.BatchDays) * 24 * time.Hour
	}
	if req.InteractiveDays != nil {
		interactiveJobTimeout = time.Duration(*req.InteractiveDays) * 24 * time.Hour
	}

	defaultRemindTime := 24 * time.Hour

	remindJobList, deletionJobList := cleanLongTimeRunningJobs(c, clients, batchJobTimeout, interactiveJobTimeout, defaultRemindTime)
	ret := map[string][]string{
		"reminded": remindJobList,
		"deleted":  deletionJobList,
	}
	return ret, nil
}

func cleanLongTimeRunningJobs(
	c context.Context,
	clients *Clients,
	batchJobTimeout,
	interactiveJobTimeout,
	defaultRemindTime time.Duration,
) (remindJobList, deletionJobList []string) {
	// 返回待删除作业、待提醒作业
	// 只考虑vcjob
	deletionJobs, reamindJobs := classifyLongTimeJobs(c, batchJobTimeout, interactiveJobTimeout, defaultRemindTime)
	deletionJobList = []string{}
	remindJobList = []string{}

	// 删除作业
	for _, job := range deletionJobs {
		err := freeLongTimeVCjob(c, clients, job)
		if err != nil {
			klog.Errorf("Failed to delete job %s: %v", job.JobName, err)
			continue
		}
		deletionJobList = append(deletionJobList, job.JobName)
	}

	// 提醒作业
	deleteTime := utils.GetLocalTime().Add(defaultRemindTime)
	for _, job := range reamindJobs {
		err := remindLongTimeVCjob(c, job, deleteTime)
		if err != nil {
			klog.Errorf("Failed to remind job %s: %v", job.JobName, err)
			continue
		}
		remindJobList = append(remindJobList, job.JobName)
	}

	return remindJobList, deletionJobList
}

func freeLongTimeVCjob(c context.Context, clients *Clients, job *model.Job) error {
	err := deleteVCjobInCluster(c, clients, job)
	if err != nil {
		return err
	}

	if !job.AlertEnabled {
		// 不需要发送邮件
		return nil
	}

	// 发送邮件
	alertMgr := alert.GetAlertMgr()
	if err := alertMgr.CleanJob(c, job.JobName, nil); err != nil {
		klog.Errorf("Send Alarm Email failed for job %s", job.JobName)
		return err
	}

	return nil
}

func remindLongTimeVCjob(c context.Context, job *model.Job, deleteTime time.Time) error {
	if !job.AlertEnabled {
		// 不需要发送邮件
		klog.Infof("Job %s is not alert enabled", job.JobName)
		return nil
	}

	// 发送邮件
	alertMgr := alert.GetAlertMgr()
	if err := alertMgr.RemindLongTimeRunningJob(c, job.JobName, deleteTime, nil); err != nil {
		klog.Errorf("Send Alarm Email failed for job %s", job.JobName)
		return err
	}

	return nil
}

func classifyLongTimeJobs(
	c context.Context, batchJobTimeout, interactiveJobTimeout, defaultRemindTime time.Duration) (deletionJobs, reamindJobs []*model.Job) {
	deletionJobs = getLongTimeVCjobs(c, batchJobTimeout, interactiveJobTimeout)
	toRemindjobs := getLongTimeVCjobs(c, batchJobTimeout-defaultRemindTime, interactiveJobTimeout-defaultRemindTime)

	deletionMap := make(map[string]bool)
	for _, job := range deletionJobs {
		deletionMap[job.JobName] = true
	}
	reamindJobs = []*model.Job{}
	for _, job := range toRemindjobs {
		if _, ok := deletionMap[job.JobName]; ok {
			continue
		}
		reamindJobs = append(reamindJobs, job)
	}

	return
}

func getLongTimeVCjobs(c context.Context, batchTimeout, interactiveTimeout time.Duration) []*model.Job {
	jobDB := query.Job
	runningJobs, err := jobDB.WithContext(c).Where(jobDB.Status.Eq(string(batch.Running))).Find()

	if err != nil {
		klog.Errorf("Failed to get running jobs: %v", err)
		return nil
	}

	whiteList, err := getJobWhiteList(c)
	if err != nil {
		// 拿不到白名单，就不进行清理，以免"误伤"
		klog.Errorf("Failed to get job white list: %v", err)
		return nil
	}

	jobList := []*model.Job{}

	now := time.Now()
	for _, job := range runningJobs {
		// 过滤掉白名单中的作业
		if lo.Contains(whiteList, job.JobName) {
			continue
		}

		jobAge := now.Sub(job.RunningTimestamp)
		shouldCheck := false
		if job.JobType == model.JobTypeJupyter || job.JobType == model.JobTypeWebIDE {
			shouldCheck = jobAge > interactiveTimeout
		} else {
			shouldCheck = jobAge > batchTimeout
		}
		if !shouldCheck {
			continue
		}

		jobList = append(jobList, job)
	}
	return jobList
}
