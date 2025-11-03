package cleaner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/alert"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
)

type CleanLowGPUUsageRequest struct {
	TimeRange int `form:"timeRange" binding:"required"`
	WaitTime  int `form:"waitTime"`
	Util      int `form:"util"`
}

func CleanLowGPUUsageJobs(c context.Context, clients *Clients, req *CleanLowGPUUsageRequest) (map[string][]string, error) {
	if req == nil {
		err := errors.New("invalid request")
		return nil, err
	}
	if req.TimeRange <= 0 || req.WaitTime <= 0 {
		err := errors.New("timeRange and waitTime must be greater than 0")
		return nil, err
	}

	remindJobList, deletionJobList := cleanLowGPUUsageJobs(c, clients, req.TimeRange, req.WaitTime, req.Util)

	ret := map[string][]string{
		"reminded": remindJobList,
		"deleted":  deletionJobList,
	}
	return ret, nil
}

func cleanLowGPUUsageJobs(
	c context.Context, clients *Clients, timeRange, waitTime, gpuUtil int) (remindJobList, deletionJobList []string) {
	remindJobList = []string{}
	deletionJobList = []string{}

	deletionJobs, reamindJobs, normalJobs := classifyLowGPUUsageJobs(c, clients, timeRange, waitTime, gpuUtil)

	// 删除作业
	for _, job := range deletionJobs {
		err := freeLowGPUUsageVCjob(c, clients, job)
		if err != nil {
			klog.Errorf("Failed to delete job %s: %v", job.JobName, err)
			continue
		}
		deletionJobList = append(deletionJobList, job.JobName)
	}

	// 提醒作业
	deleteTime := utils.GetLocalTime().Add(time.Duration(waitTime) * time.Minute)
	for _, job := range reamindJobs {
		err := remindLowGPUUsageVCjob(c, job, deleteTime)
		if err != nil {
			klog.Errorf("Failed to remind job %s: %v", job.JobName, err)
			continue
		}
		remindJobList = append(remindJobList, job.JobName)
	}

	// 对于GPU利用率恢复正常的作业，允许再次被提醒
	for _, job := range normalJobs {
		err := allowRepeatAlert(c, job, model.LowGPUJobRemindedAlert)
		if err != nil {
			klog.Errorf("Failed to allow repeat alert for job %s: %v", job.JobName, err)
			continue
		}
	}

	return remindJobList, deletionJobList
}

func allowRepeatAlert(c context.Context, job *model.Job, alertType model.AlertType) error {
	alertDB := query.Alert
	record, err := alertDB.WithContext(c).Where(
		alertDB.JobName.Eq(job.JobName),
		alertDB.AlertType.Eq(alertType.String()),
	).First()

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return nil
	}

	// 允许再次被提醒
	record.AllowRepeat = true
	if err := alertDB.WithContext(c).Save(record); err != nil {
		return err
	}

	return nil
}

func freeLowGPUUsageVCjob(c context.Context, clients *Clients, job *model.Job) error {
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
	if err := alertMgr.DeleteJob(c, job.JobName, nil); err != nil {
		klog.Errorf("Send Alarm Email failed for job %s", job.JobName)
	}

	return nil
}

func remindLowGPUUsageVCjob(c context.Context, job *model.Job, deleteTime time.Time) error {
	if !job.AlertEnabled {
		// 不需要发送邮件
		klog.Infof("Job %s is not alert enabled", job.JobName)
		return nil
	}

	// 发送邮件
	alertMgr := alert.GetAlertMgr()
	if err := alertMgr.RemindLowUsageJob(c, job.JobName, deleteTime, nil); err != nil {
		klog.Errorf("Send Alarm Email failed for job %s", job.JobName)
		return err
	}

	return nil
}

func deleteVCjobInCluster(c context.Context, clients *Clients, job *model.Job) error {
	vcjob := &batch.Job{}
	namespace := config.GetConfig().Namespaces.Job
	if err := clients.Client.Get(c, client.ObjectKey{Name: job.JobName, Namespace: namespace}, vcjob); err != nil {
		return err
	}

	if err := clients.Client.Delete(c, vcjob); err != nil {
		return err
	}
	return nil
}

func classifyLowGPUUsageJobs(
	c context.Context, clients *Clients, timeRange, waitTime, gpuUtil int) (deletionJobs, reamindJobs, normalJobs []*model.Job) {
	// 返回待删除作业、待提醒作业、正常作业
	// 只考虑vcjob
	jobDB := query.Job
	deletionJobs = getLowGPUUsageVCjobs(c, clients, timeRange+waitTime, gpuUtil)
	toRemindJobs := getLowGPUUsageVCjobs(c, clients, timeRange, gpuUtil)
	runningJobs, _ := jobDB.WithContext(c).Where(jobDB.Status.Eq(string(batch.Running))).Find()

	deletionMap := make(map[string]bool)
	for _, job := range deletionJobs {
		deletionMap[job.JobName] = true
	}

	reamindJobs = []*model.Job{}
	reamindMap := make(map[string]bool)
	for _, job := range toRemindJobs {
		if _, ok := deletionMap[job.JobName]; ok {
			continue
		}
		reamindMap[job.JobName] = true
		reamindJobs = append(reamindJobs, job)
	}

	normalJobs = []*model.Job{}
	for _, job := range runningJobs {
		if _, ok := deletionMap[job.JobName]; ok {
			continue
		}
		if _, ok := reamindMap[job.JobName]; ok {
			continue
		}
		normalJobs = append(normalJobs, job)
	}
	return deletionJobs, reamindJobs, normalJobs
}

func getLowGPUUsageVCjobs(c context.Context, clients *Clients, duration, gpuUtil int) []*model.Job {
	jobDB := query.Job
	podList := getLowGPUUsagePods(c, clients, duration, gpuUtil)

	whiteList, err := getJobWhiteList(c)
	if err != nil {
		// 拿不到白名单，就不进行清理，以免"误伤"
		klog.Errorf("Failed to get job white list: %v", err)
		return nil
	}

	jobList := []*model.Job{}
	for _, pod := range podList {
		if len(pod.OwnerReferences) == 0 {
			continue
		}

		owner := pod.OwnerReferences[0]

		if owner.Kind != VCJOBKIND || owner.APIVersion != VCJOBAPIVERSION {
			continue
		}

		// 过滤掉白名单中的作业
		if lo.Contains(whiteList, owner.Name) {
			continue
		}

		job, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(owner.Name)).First()
		if err != nil {
			klog.Infof("Fail to get vcjob %s\n", owner.Name)
			continue
		}

		jobList = append(jobList, job)
	}
	return jobList
}

func getJobWhiteList(c context.Context) ([]string, error) {
	var cleanList []string
	jobDB := query.Job
	curTime := utils.GetLocalTime()

	data, err := jobDB.WithContext(c).Where(jobDB.LockedTimestamp.Gt(curTime)).Find()

	if err != nil {
		return nil, err
	}
	for _, item := range data {
		cleanList = append(cleanList, item.JobName)
	}
	return cleanList, nil
}

func getLowGPUUsagePods(c context.Context, clients *Clients, duration, gpuUtil int) []*v1.Pod {
	namespace := config.GetConfig().Namespaces.Job
	querys := clients.PromClient.QueryNodeGPUUtilInNS(namespace)
	podList := []*v1.Pod{}

	for i := range querys {
		q := &querys[i]
		pod, err := clients.KubeClient.CoreV1().
			Pods(namespace).
			Get(c, q.Pod, metav1.GetOptions{})
		if err != nil {
			continue
		}

		if clients.PromClient.
			GetLeastUsedGPUJobList(q.Pod, fmt.Sprintf("%d", duration), fmt.Sprintf("%d", gpuUtil)) <= 0 {
			// 该Pod GPU利用率低于util
			// 或：该Pod生命周期小于duration
			continue
		}

		podList = append(podList, pod)
	}

	return podList
}
