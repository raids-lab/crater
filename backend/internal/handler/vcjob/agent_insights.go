package vcjob

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/monitor"
)

//nolint:gochecknoinits // The agent package discovers this facade through handler registration.
func init() {
	handler.RegisterJobInsightReaderFactory(NewAgentJobInsightReader)
}

type agentJobInsightReader struct {
	mgr *VolcanojobMgr
}

func NewAgentJobInsightReader(conf *handler.RegisterConfig) handler.JobInsightReader {
	return &agentJobInsightReader{mgr: NewVolcanojobMgr(conf).(*VolcanojobMgr)}
}

func (r *agentJobInsightReader) FindScopedJob(
	ctx context.Context,
	token util.JWTMessage,
	jobName string,
) (*model.Job, error) {
	job, err := getJob(ctx, jobName, &token)
	if err == nil {
		return job, nil
	}

	if isRecordNotFound(err) {
		k := query.Kaniko
		buildQuery := k.WithContext(ctx).Where(k.ImagePackName.Eq(jobName))
		if token.RolePlatform != model.RoleAdmin {
			buildQuery = buildQuery.Where(k.UserID.Eq(token.UserID))
		}
		if _, buildErr := buildQuery.First(); buildErr == nil {
			return nil, bizerr.BadRequest.ParameterError.New(
				fmt.Sprintf(
					"%q is an image build, not a platform job; use get_image_build_detail first, "+
						"then inspect its pod with k8s_get_pod_logs / k8s_get_events",
					jobName,
				),
			)
		}
	}
	return nil, bizerr.NotFound.DataBaseNotFound.Wrap(err, "job not found")
}

func (r *agentJobInsightReader) BuildJobDetail(job *model.Job) any {
	var profileData *monitor.ProfileData
	if job.ProfileData != nil {
		profileData = job.ProfileData.Data()
	}

	var scheduleData *model.ScheduleData
	if job.ScheduleData != nil {
		scheduleData = job.ScheduleData.Data()
	}

	var terminatedStates []v1.ContainerStateTerminated
	if job.TerminatedStates != nil {
		terminatedStates = job.TerminatedStates.Data()
	}

	scheduleType := model.ScheduleTypeNormal
	if job.ScheduleType != nil {
		scheduleType = *job.ScheduleType
	}

	namespace := ""
	if vcjob := job.Attributes.Data(); vcjob != nil {
		namespace = vcjob.Namespace
	}

	return JobDetailResp{
		Name:      job.Name,
		Namespace: namespace,
		Username:  job.User.Name,
		Nickname:  job.User.Nickname,
		UserInfo: model.UserInfo{
			Username: job.User.Name,
			Nickname: job.User.Nickname,
		},
		JobName:                 job.JobName,
		JobType:                 job.JobType,
		ScheduleType:            scheduleType,
		WaitingToleranceSeconds: job.WaitingToleranceSeconds,
		Queue:                   job.Account.Nickname,
		Status:                  job.Status,
		Resources:               job.Resources.Data(),
		ProfileData:             profileData,
		ScheduleData:            scheduleData,
		Events:                  getStoredJobEvents(job),
		TerminatedStates:        terminatedStates,
		CreationTimestamp:       metav1.NewTime(job.CreationTimestamp),
		RunningTimestamp:        metav1.NewTime(job.RunningTimestamp),
		CompletedTimestamp:      metav1.NewTime(job.CompletedTimestamp),
	}
}

func (r *agentJobInsightReader) GetJobEvents(
	ctx context.Context,
	token util.JWTMessage,
	jobName string,
) (any, error) {
	job, err := r.FindScopedJob(ctx, token, jobName)
	if err != nil {
		return nil, err
	}
	vcjob := job.Attributes.Data()
	storedEvents := getStoredJobEvents(job)
	if vcjob == nil {
		if len(storedEvents) > 0 {
			return storedEvents, nil
		}
		return nil, bizerr.NotFound.DataBaseNotFound.New("job attributes not found")
	}

	events, err := r.listLiveJobEvents(ctx, vcjob)
	if err != nil {
		if len(storedEvents) > 0 {
			return storedEvents, nil
		}
		return nil, err
	}
	if len(events) == 0 && len(storedEvents) > 0 {
		return storedEvents, nil
	}
	return events, nil
}

func (r *agentJobInsightReader) GetJobLog(
	ctx context.Context,
	token util.JWTMessage,
	jobName string,
	tailLines int64,
	keyword string,
) (map[string]string, error) {
	job, err := r.FindScopedJob(ctx, token, jobName)
	if err != nil {
		return nil, err
	}
	return r.readJobLogPayload(ctx, job, tailLines, keyword)
}

func (r *agentJobInsightReader) GetDiagnosticContext(
	ctx context.Context,
	token util.JWTMessage,
	jobName string,
	includeLog bool,
	tailLines int64,
) (handler.JobContextResp, error) {
	job, err := r.FindScopedJob(ctx, token, jobName)
	if err != nil {
		return handler.JobContextResp{}, err
	}

	resp := handler.JobContextResp{}
	resp.Meta.Name = job.Name
	resp.Meta.JobName = job.JobName
	resp.Meta.Namespace = getAgentJobNamespace(job)
	resp.Meta.User = job.User.Name
	resp.Meta.Queue = job.Account.Nickname
	resp.Meta.JobType = job.JobType
	resp.Meta.Status = job.Status
	resp.Meta.CreationTimestamp = job.CreationTimestamp
	resp.Meta.RunningTimestamp = job.RunningTimestamp
	resp.Meta.CompletedTimestamp = job.CompletedTimestamp
	if job.Nodes.Data() != nil {
		resp.Meta.Nodes = job.Nodes.Data()
	}
	resp.Meta.Resources = job.Resources.Data()

	if job.ProfileData != nil {
		resp.DB.ProfileData = job.ProfileData.Data()
	}
	if job.ScheduleData != nil {
		resp.DB.ScheduleData = job.ScheduleData.Data()
	}
	if events, eventsErr := r.GetJobEvents(ctx, token, jobName); eventsErr == nil {
		if typedEvents, ok := events.([]v1.Event); ok {
			resp.DB.Events = typedEvents
		}
	} else if job.Events != nil {
		resp.DB.Events = job.Events.Data()
	}
	if job.TerminatedStates != nil {
		resp.DB.TerminatedStates = job.TerminatedStates.Data()
	}

	if includeLog {
		logPayload, logErr := r.readJobLogPayload(ctx, job, tailLines, "")
		if logErr != nil {
			return handler.JobContextResp{}, logErr
		}
		resp.Log.Container = logPayload["container"]
		resp.Log.Tail = logPayload["log"]
	}
	return resp, nil
}

func (r *agentJobInsightReader) listLiveJobEvents(ctx context.Context, vcjob *v1alpha1.Job) ([]v1.Event, error) {
	jobEvents, err := r.mgr.kubeClient.CoreV1().Events(vcjob.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", vcjob.Name),
		TypeMeta:      metav1.TypeMeta{Kind: "Job", APIVersion: "batch.volcano.sh/v1alpha1"},
	})
	if err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "failed to list job events")
	}
	events := jobEvents.Items

	baseURL, ok := vcjob.Labels[crclient.LabelKeyBaseURL]
	if !ok || baseURL == "" {
		return events, nil
	}

	podList := &v1.PodList{}
	labels := client.MatchingLabels{crclient.LabelKeyBaseURL: baseURL}
	if err := r.mgr.client.List(ctx, podList, client.InNamespace(vcjob.Namespace), labels); err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "failed to list job pods")
	}

	containsPodEvents := false
	for i := range podList.Items {
		pod := &podList.Items[i]
		podEvents, err := r.mgr.kubeClient.CoreV1().Events(vcjob.Namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		})
		if err != nil {
			return nil, bizerr.Internal.K8sServiceError.Wrap(err, "failed to list pod events")
		}
		if len(podEvents.Items) > 0 && !containsPodEvents {
			containsPodEvents = true
			events = []v1.Event{}
		}
		events = append(events, podEvents.Items...)
	}
	return events, nil
}

func (r *agentJobInsightReader) readJobLogPayload(
	ctx context.Context,
	job *model.Job,
	tailLines int64,
	keyword string,
) (map[string]string, error) {
	if tailLines <= 0 {
		tailLines = 100
	}

	namespace := getAgentJobNamespace(job)
	labelSelector := fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, job.JobName)
	if vcjob := job.Attributes.Data(); vcjob != nil {
		if labelVal, ok := vcjob.Labels[crclient.LabelKeyBaseURL]; ok && labelVal != "" {
			labelSelector = fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, labelVal)
		}
	}

	podList, podErr := r.mgr.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if podErr != nil || len(podList.Items) == 0 {
		return map[string]string{"log": "Pod not found or no live logs available."}, nil
	}

	pod := podList.Items[0]
	containerName := ""
	if len(pod.Spec.Containers) > 0 {
		containerName = pod.Spec.Containers[0].Name
	}

	logBytes, logErr := r.mgr.kubeClient.CoreV1().Pods(namespace).GetLogs(pod.Name, &v1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
	}).DoRaw(ctx)
	if logErr != nil {
		return map[string]string{"log": fmt.Sprintf("Failed to retrieve logs: %v", logErr)}, nil
	}

	logContent, filterErr := filterAgentJobLogByKeyword(string(logBytes), keyword)
	if filterErr != nil {
		return nil, filterErr
	}

	payload := map[string]string{
		"podName":   pod.Name,
		"container": containerName,
		"log":       logContent,
	}
	if keyword != "" {
		payload["keyword"] = keyword
	}
	return payload, nil
}

func getAgentJobNamespace(job *model.Job) string {
	if job != nil {
		if vcjob := job.Attributes.Data(); vcjob != nil && vcjob.Namespace != "" {
			return vcjob.Namespace
		}
	}
	return config.GetConfig().Namespaces.Job
}

func filterAgentJobLogByKeyword(logContent, keyword string) (string, error) {
	if keyword == "" {
		return logContent, nil
	}
	re, err := regexp.Compile(keyword)
	if err != nil {
		return "", bizerr.BadRequest.ParameterError.Wrap(err, "invalid keyword regex")
	}
	lines := strings.Split(logContent, "\n")
	matched := make([]string, 0, len(lines))
	for _, line := range lines {
		if re.MatchString(line) {
			matched = append(matched, line)
		}
	}
	return strings.Join(matched, "\n"), nil
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(err.Error()), "record not found")
}
