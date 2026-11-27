package vcjob

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gen/field"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/klog/v2"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/extender"
)

// QueueJob contains only the information visible on the queue board.
type QueueJob struct {
	Name      string                              `json:"name"`
	JobName   string                              `json:"jobName"`
	JobType   string                              `json:"jobType"`
	AccountID uint                                `json:"accountId"`
	Account   string                              `json:"account"`
	UserInfo  model.UserInfo                      `json:"userInfo" gorm:"embedded"`
	Resources datatypes.JSONType[v1.ResourceList] `json:"resources" swaggertype:"object,string"`
	CreatedAt time.Time                           `json:"createdAt"`
	Phase     string                              `json:"phase"`
	Reason    string                              `json:"reason,omitempty"`
	JobUID    string                              `json:"-" gorm:"column:job_uid"`
}

type QueueSnapshot struct {
	ObservedAt       time.Time  `json:"observedAt"`
	Jobs             []QueueJob `json:"jobs"`
	ReasonsAvailable bool       `json:"reasonsAvailable"`
}

// GetAccountQueue godoc
//
//	@Summary	List current unscheduled jobs in the active account
//	@Tags	VolcanoJob
//	@Security	Bearer
//	@Param	mine	query	bool	false	"Limit to the signed-in user"
//	@Success	200	{object}	resputil.Response[QueueSnapshot]
//	@Router	/v1/vcjobs/queue [get]
func (mgr *VolcanojobMgr) GetAccountQueue(c *gin.Context) {
	var req struct {
		Mine bool `form:"mine"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid queue filter"))
		return
	}
	token := util.GetToken(c)
	scope := jobListScope{AccountID: &token.AccountID}
	if req.Mine {
		scope.UserID = &token.UserID
	}
	mgr.serveQueue(c, scope)
}

// GetAdminQueue godoc
//
//	@Summary	List current unscheduled jobs across accounts
//	@Tags	VolcanoJob
//	@Security	Bearer
//	@Success	200	{object}	resputil.Response[QueueSnapshot]
//	@Router	/v1/admin/vcjobs/queue [get]
func (mgr *VolcanojobMgr) GetAdminQueue(c *gin.Context) {
	mgr.serveQueue(c, jobListScope{})
}

func (mgr *VolcanojobMgr) serveQueue(c *gin.Context, scope jobListScope) {
	rows, err := findQueueJobs(c, scope)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list queued jobs failed"))
		return
	}
	err = mgr.addQueueReasons(c, rows)
	if err != nil {
		klog.Warningf("Unable to read queue admission events: %v", err)
	}
	resputil.Success(c, QueueSnapshot{Jobs: rows, ReasonsAvailable: err == nil, ObservedAt: time.Now()})
}

func findQueueJobs(ctx context.Context, scope jobListScope) ([]QueueJob, error) {
	j, u := query.Job, query.User
	request := jobListQuery{
		Statuses: []string{"", string(batch.Pending), string(model.Prequeue)},
		Phases:   []string{filterPhaseInit, string(batch.Pending), string(scheduling.PodGroupInqueue)},
	}
	rows := make([]QueueJob, 0)
	err := applyJobFilters(ctx, scope, &request, -1).
		Select(j.Name, j.JobName, j.JobType, j.AccountID, j.Resources,
			j.CreationTimestamp.As("created_at"), jobFilterPhaseExpr().As("phase"),
			u.Name.As("username"), u.Nickname.As("nickname"),
			field.NewUnsafeFieldRaw("COALESCE(NULLIF(accounts.nickname, ''), accounts.name, '')").As("account"),
			field.NewUnsafeFieldRaw("jobs.attributes -> 'metadata' ->> 'uid'").As("job_uid")).
		Order(j.CreationTimestamp.Asc(), j.ID.Asc()).Scan(&rows)
	return rows, err
}

func (mgr *VolcanojobMgr) addQueueReasons(ctx context.Context, jobs []QueueJob) error {
	waiting := make(map[string]*QueueJob)
	for i := range jobs {
		j := &jobs[i]
		if j.Phase != string(scheduling.PodGroupInqueue) && j.JobUID != "" {
			waiting[j.JobUID] = j
		}
	}
	if len(waiting) == 0 {
		return nil
	}
	selector := fields.Set{
		"involvedObject.kind": "Job", "involvedObject.apiVersion": batch.SchemeGroupVersion.String(),
		"type": v1.EventTypeWarning, "source": "crater-volcano-extender",
	}
	events, err := mgr.kubeClient.CoreV1().Events(config.GetConfig().Namespaces.Job).
		List(ctx, metav1.ListOptions{FieldSelector: selector.AsSelector().String()})
	if err != nil {
		return err
	}
	latest := make(map[string]time.Time)
	// The extender refreshes an active hold every five minutes. Ignore old
	// warnings after two refresh periods instead of attributing a past hold.
	cutoff := time.Now().Add(-10 * time.Minute)
	for i := range events.Items {
		event := &events.Items[i]
		uid := string(event.InvolvedObject.UID)
		row := waiting[uid]
		if row == nil || row.JobName != event.InvolvedObject.Name {
			continue
		}
		at := queueWarningTime(event)
		if at.Before(cutoff) || !at.After(latest[uid]) {
			continue
		}
		latest[uid] = at
		switch event.Reason {
		case extender.ReasonQueueQuotaExceeded, extender.ReasonBlockedByTimedOutJob, extender.ReasonExceedsQueueCapability:
			row.Reason = event.Reason
		default:
			row.Reason = ""
		}
	}
	return nil
}

func queueWarningTime(event *v1.Event) time.Time {
	at := event.LastTimestamp.Time
	if event.EventTime.After(at) {
		at = event.EventTime.Time
	}
	if event.Series != nil && event.Series.LastObservedTime.After(at) {
		at = event.Series.LastObservedTime.Time
	}
	if at.IsZero() {
		return event.CreationTimestamp.Time
	}
	return at
}
