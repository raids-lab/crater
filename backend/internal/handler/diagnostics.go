package handler

import (
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/monitor"
)

//nolint:gochecknoinits // Handler managers are registered during package initialization.
func init() {
	Registers = append(Registers, NewDiagnosticsMgr)
}

type DiagnosticsMgr struct {
	name       string
	client     client.Client
	kubeClient kubernetes.Interface
}

func NewDiagnosticsMgr(conf *RegisterConfig) Manager {
	return &DiagnosticsMgr{
		name:       "diagnostics",
		client:     conf.Client,
		kubeClient: conf.KubeClient,
	}
}

func (mgr *DiagnosticsMgr) GetName() string { return mgr.name }

func (mgr *DiagnosticsMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *DiagnosticsMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("failure-types/top", mgr.GetTopFailureTypes)
	g.GET("context/:name", mgr.GetDiagnosticContext)
}

func (mgr *DiagnosticsMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("failure-types/top", mgr.GetTopFailureTypesAdmin)
}

type FailureStat struct {
	Type    string   `json:"type"`
	Count   int      `json:"count"`
	Samples []string `json:"samples"`
}

// GetTopFailureTypes godoc
// @Summary Get top failure types
// @Description Get aggregated statistics of the most common failure types for current user's failed jobs.
// @Tags diagnostics
// @Produce json
// @Param days query int false "Number of days to look back (-1 means all time)"
// @Param limit query int false "Maximum number of failure types to return"
// @Success 200 {object} resputil.Response[[]FailureStat]
// @Router /api/v1/diagnostics/failure-types/top [get]
func (mgr *DiagnosticsMgr) GetTopFailureTypes(c *gin.Context) {
	type QueryParams struct {
		Days  int `form:"days"`
		Limit int `form:"limit"`
	}
	var req QueryParams
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	days := -1
	if req.Days > 0 || req.Days == -1 {
		days = req.Days
	}
	limit := 10
	if req.Limit > 0 {
		limit = req.Limit
	}

	token := util.GetToken(c)
	j := query.Job
	q := j.WithContext(c).
		Preload(j.User).
		Preload(j.Account).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		Where(j.Status.Eq(string(batch.Failed)))

	if days != -1 {
		now := time.Now()
		lookback := now.AddDate(0, 0, -days)
		q = q.Where(j.CompletedTimestamp.Gte(lookback))
	}

	jobs, err := q.Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	type agg struct {
		count   int
		samples []string
	}
	result := map[string]*agg{}

	for i := range jobs {
		job := jobs[i]
		t := categorizeFailure(job)
		if _, ok := result[t.typeName]; !ok {
			result[t.typeName] = &agg{count: 0, samples: []string{}}
		}
		result[t.typeName].count++
		if t.sample != "" && len(result[t.typeName].samples) < 5 {
			result[t.typeName].samples = append(result[t.typeName].samples, t.sample)
		}
	}

	stats := make([]FailureStat, 0, len(result))
	for k, v := range result {
		stats = append(stats, FailureStat{Type: k, Count: v.count, Samples: v.samples})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Count > stats[j].Count })
	if len(stats) > limit {
		stats = stats[:limit]
	}
	resputil.Success(c, stats)
}

// GetTopFailureTypesAdmin godoc
// @Summary Get top failure types (admin)
// @Description Get aggregated statistics of the most common failure types for all users' failed jobs.
// @Tags diagnostics
// @Produce json
// @Param days query int false "Number of days to look back (-1 means all time)"
// @Param limit query int false "Maximum number of failure types to return"
// @Success 200 {object} resputil.Response[[]FailureStat]
// @Router /api/v1/admin/diagnostics/failure-types/top [get]
func (mgr *DiagnosticsMgr) GetTopFailureTypesAdmin(c *gin.Context) {
	type QueryParams struct {
		Days  int `form:"days"`
		Limit int `form:"limit"`
	}
	var req QueryParams
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	days := -1
	if req.Days > 0 || req.Days == -1 {
		days = req.Days
	}
	limit := 10
	if req.Limit > 0 {
		limit = req.Limit
	}

	j := query.Job
	q := j.WithContext(c).
		Preload(j.User).
		Preload(j.Account).
		Where(j.Status.Eq(string(batch.Failed)))

	if days != -1 {
		now := time.Now()
		lookback := now.AddDate(0, 0, -days)
		q = q.Where(j.CompletedTimestamp.Gte(lookback))
	}

	jobs, err := q.Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	type agg struct {
		count   int
		samples []string
	}
	result := map[string]*agg{}

	for i := range jobs {
		job := jobs[i]
		t := categorizeFailure(job)
		if _, ok := result[t.typeName]; !ok {
			result[t.typeName] = &agg{count: 0, samples: []string{}}
		}
		result[t.typeName].count++
		if t.sample != "" && len(result[t.typeName].samples) < 5 {
			result[t.typeName].samples = append(result[t.typeName].samples, t.sample)
		}
	}

	stats := make([]FailureStat, 0, len(result))
	for k, v := range result {
		stats = append(stats, FailureStat{Type: k, Count: v.count, Samples: v.samples})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Count > stats[j].Count })
	if len(stats) > limit {
		stats = stats[:limit]
	}
	resputil.Success(c, stats)
}

type classifyResult struct {
	typeName string
	sample   string
}

const (
	exitCodeSegmentationFault = 139
	exitCodeCommandNotFound   = 127
	exitCodeGracefulTerm      = 143
)

//nolint:gocyclo // Rule matching intentionally keeps failure classification in one place.
func categorizeFailure(job *model.Job) classifyResult {
	if job.TerminatedStates != nil {
		terminated := job.TerminatedStates.Data()
		for i := range terminated {
			ts := &terminated[i]
			if strings.EqualFold(ts.Reason, "OOMKilled") || ts.ExitCode == 137 {
				return classifyResult{typeName: "OOMKilled", sample: sampleTerminated(ts)}
			}
			if ts.ExitCode == exitCodeSegmentationFault {
				return classifyResult{typeName: "SegmentationFault", sample: sampleTerminated(ts)}
			}
			if ts.ExitCode == exitCodeCommandNotFound {
				return classifyResult{typeName: "CommandNotFound", sample: sampleTerminated(ts)}
			}
			if ts.ExitCode == exitCodeGracefulTerm {
				return classifyResult{typeName: "GracefulTermination", sample: sampleTerminated(ts)}
			}
			if strings.EqualFold(ts.Reason, "Error") && ts.ExitCode != 0 {
				return classifyResult{typeName: "ContainerError", sample: sampleTerminated(ts)}
			}
		}
	}
	if job.Events != nil {
		events := job.Events.Data()
		for i := range events {
			ev := &events[i]
			if ev.Reason == "ErrImagePull" || ev.Reason == "ImagePullBackOff" {
				return classifyResult{typeName: "ImagePullError", sample: sampleEvent(ev)}
			}
			if ev.Reason == "FailedScheduling" {
				msg := strings.ToLower(ev.Message)
				switch {
				case strings.Contains(msg, "insufficient"):
					return classifyResult{typeName: "SchedulingInsufficientResources", sample: sampleEvent(ev)}
				case strings.Contains(msg, "didn't match node selector") || strings.Contains(msg, "node(s) didn't match"):
					return classifyResult{typeName: "SchedulingNodeSelectorMismatch", sample: sampleEvent(ev)}
				case strings.Contains(msg, "taint"):
					return classifyResult{typeName: "SchedulingTaintMismatch", sample: sampleEvent(ev)}
				default:
					return classifyResult{typeName: "SchedulingFailed", sample: sampleEvent(ev)}
				}
			}
			if ev.Reason == "BackOff" && strings.Contains(strings.ToLower(ev.Message), "back-off restarting failed container") {
				return classifyResult{typeName: "CrashLoopBackOff", sample: sampleEvent(ev)}
			}
			if ev.Reason == "Evicted" {
				return classifyResult{typeName: "Evicted", sample: sampleEvent(ev)}
			}
			if ev.Reason == "FailedMount" || strings.Contains(strings.ToLower(ev.Message), "mountvolume") {
				return classifyResult{typeName: "VolumeMountFailed", sample: sampleEvent(ev)}
			}
			if ev.Reason == "DeadlineExceeded" {
				return classifyResult{typeName: "JobDeadlineExceeded", sample: sampleEvent(ev)}
			}
		}
	}
	switch job.Status {
	case batch.Aborted, batch.Terminated:
		return classifyResult{typeName: "JobAbortedOrTerminated", sample: ""}
	}
	return classifyResult{typeName: "UnknownFailure", sample: ""}
}

func sampleTerminated(ts *v1.ContainerStateTerminated) string {
	if ts == nil {
		return ""
	}
	return strings.TrimSpace(ts.Reason + " " + ts.Message)
}

func sampleEvent(ev *v1.Event) string {
	if ev == nil {
		return ""
	}
	return strings.TrimSpace(ev.Message)
}

type JobContextResp struct {
	Meta struct {
		Name               string          `json:"name"`
		JobName            string          `json:"jobName"`
		Namespace          string          `json:"namespace"`
		User               string          `json:"user"`
		Queue              string          `json:"queue"`
		JobType            model.JobType   `json:"jobType"`
		Status             batch.JobPhase  `json:"status"`
		CreationTimestamp  time.Time       `json:"createdAt"`
		RunningTimestamp   time.Time       `json:"startedAt"`
		CompletedTimestamp time.Time       `json:"completedAt"`
		Nodes              []string        `json:"nodes"`
		Resources          v1.ResourceList `json:"resources"`
	} `json:"meta"`
	DB struct {
		ProfileData      *monitor.ProfileData          `json:"profileData"`
		ScheduleData     *model.ScheduleData           `json:"scheduleData"`
		Events           []v1.Event                    `json:"events"`
		TerminatedStates []v1.ContainerStateTerminated `json:"terminatedStates"`
	} `json:"db"`
	Log struct {
		Container string `json:"container"`
		Tail      string `json:"tail"`
	} `json:"log"`
}

//nolint:gocyclo // Context assembly combines auth scope, DB query, and optional log collection.
func (mgr *DiagnosticsMgr) GetDiagnosticContext(c *gin.Context) {
	type URI struct {
		Name string `uri:"name" binding:"required"`
	}
	type QueryParams struct {
		TailLines  int  `form:"tailLines"`
		IncludeLog bool `form:"includeLog"`
	}
	var uri URI
	if err := c.ShouldBindUri(&uri); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	var qp QueryParams
	if err := c.ShouldBindQuery(&qp); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	if qp.TailLines <= 0 {
		qp.TailLines = 200
	}

	token := util.GetToken(c)
	j := query.Job
	q := j.WithContext(c).Preload(j.Account).Preload(j.User).Where(j.JobName.Eq(uri.Name))
	var job *model.Job
	var err error
	if token.RolePlatform == model.RoleAdmin {
		job, err = q.First()
	} else {
		job, err = q.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).First()
	}
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resp := JobContextResp{}
	resp.Meta.Name = job.Name
	resp.Meta.JobName = job.JobName
	resp.Meta.Namespace = job.Attributes.Data().Namespace
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
	if job.Events != nil {
		resp.DB.Events = job.Events.Data()
	}
	if job.TerminatedStates != nil {
		resp.DB.TerminatedStates = job.TerminatedStates.Data()
	}

	if qp.IncludeLog {
		namespace := job.Attributes.Data().Namespace
		podName := ""
		for i := range job.Attributes.Data().Spec.Tasks {
			task := &job.Attributes.Data().Spec.Tasks[i]
			podName = job.JobName + "-" + task.Name + "-0"
			break
		}
		if podName != "" {
			var pod v1.Pod
			if err := mgr.client.Get(c, client.ObjectKey{Namespace: namespace, Name: podName}, &pod); err == nil {
				container := pod.Spec.Containers[0].Name
				tail := int64(qp.TailLines)
				data, logErr := mgr.kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
					Container: container,
					TailLines: &tail,
				}).DoRaw(c)
				if logErr == nil {
					resp.Log.Container = container
					resp.Log.Tail = string(data)
				}
			}
		}
	}

	resputil.Success(c, resp)
}
