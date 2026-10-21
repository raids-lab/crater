package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/crclient"
)

type PodSnapshotHandler struct {
	kubeClient kubernetes.Interface
}

func NewPodSnapshotHandler(conf *RegisterConfig) *PodSnapshotHandler {
	return &PodSnapshotHandler{kubeClient: conf.KubeClient}
}

func (h *PodSnapshotHandler) Register(g *gin.RouterGroup) {
	// Keep the wildcard name aligned with the existing /vcjobs/:name routes.
	// Gin treats wildcard names at the same path position as conflicting even
	// though the generated URL is otherwise identical.
	g.GET("/:name/pods/snapshot", h.ListJobPods)
}

type PodSnapshotQuery struct {
	Limit           int64  `form:"limit"`
	Cursor          string `form:"cursor"`
	ResourceVersion string `form:"resource_version"`
}

type PodSnapshotPage struct {
	Items           []crclient.Pod `json:"items"`
	NextCursor      *string        `json:"next_cursor"`
	HasMore         bool           `json:"has_more"`
	SnapshotID      string         `json:"snapshot_id"`
	ResourceVersion string         `json:"resource_version"`
	ObservedAt      string         `json:"observed_at"`
}

// ListJobPods godoc
//
//	@Summary		获取作业 Pod 快照
//	@Description	使用 Kubernetes limit/continue 返回作业范围的只读 Pod 快照
//	@Tags		PodSnapshot
//	@Produce	json
//	@Security	Bearer
//	@Param		name	path	string	true	"作业名称"
//	@Param		limit	query	int	false	"每页数量，1-200"
//	@Param		cursor	query	string	false	"不透明 continue 游标"
//	@Param		resource_version	query	string	false	"Kubernetes resource version"
//	@Success	200	{object}	resputil.Response[PodSnapshotPage]
//	@Failure	409	{object}	resputil.Response[any]	"游标过期"
//	@Router		/v1/vcjobs/{name}/pods/snapshot [get]
//
// The handler returns a read-only, cursor-based Pod snapshot for a VCJob.
// The job lookup intentionally reuses the existing ownership check so this
// endpoint cannot be used to enumerate another user's Pods.
//
//nolint:gocyclo // This boundary keeps authorization, snapshot validation, and Kubernetes error mapping together.
func (h *PodSnapshotHandler) ListJobPods(c *gin.Context) {
	jobName := c.Param("name")
	if jobName == "" {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("job is required"))
		return
	}

	var req PodSnapshotQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid pod snapshot query"))
		return
	}
	if req.Limit == 0 {
		req.Limit = 50
	}
	if req.Limit < 1 || req.Limit > 200 {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("limit must be between 1 and 200"))
		return
	}

	token := util.GetToken(c)
	jobQuery := query.Job
	job, err := jobQuery.WithContext(c).
		Preload(jobQuery.Account).
		Preload(jobQuery.User).
		Where(jobQuery.JobName.Eq(jobName)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.Wrap(err, "job not found"))
		} else {
			resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "load job snapshot metadata failed"))
		}
		return
	}
	if token.RolePlatform != model.RoleAdmin && (job.UserID != token.UserID || job.AccountID != token.AccountID) {
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("job is not visible to this user"))
		return
	}

	attrs := job.Attributes.Data()
	if attrs == nil || attrs.Namespace == "" {
		resputil.HandleError(c, bizerr.Internal.ServiceError.New("job snapshot metadata is missing"))
		return
	}
	selector := attrs.Labels[crclient.LabelKeyBaseURL]
	if selector == "" {
		resputil.HandleError(c, bizerr.Internal.ServiceError.New("job snapshot selector is missing"))
		return
	}

	pods, err := h.kubeClient.CoreV1().Pods(attrs.Namespace).List(c, metav1.ListOptions{
		Limit:           req.Limit,
		Continue:        req.Cursor,
		LabelSelector:   fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, selector),
		ResourceVersion: req.ResourceVersion,
	})
	if err != nil {
		if apierrors.IsResourceExpired(err) || apierrors.IsGone(err) {
			resputil.HandleError(c, bizerr.Conflict.CursorExpired.New("snapshot cursor expired; refresh and retry"))
			return
		}
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "list pod snapshot failed"))
		return
	}

	items := make([]crclient.Pod, 0, len(pods.Items))
	for i := range pods.Items {
		items = append(items, crclient.PodInfoFromKubePod(&pods.Items[i]))
	}
	var nextCursor *string
	if pods.Continue != "" {
		nextCursor = &pods.Continue
	}
	hash := sha256.Sum256([]byte(attrs.Namespace + "\x00" + selector + "\x00" + pods.ResourceVersion))
	resputil.Success(c, PodSnapshotPage{
		Items:           items,
		NextCursor:      nextCursor,
		HasMore:         pods.Continue != "",
		SnapshotID:      hex.EncodeToString(hash[:8]),
		ResourceVersion: pods.ResourceVersion,
		ObservedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	})
}
