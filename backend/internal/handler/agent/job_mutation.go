package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
)

type agentJobNameArgs struct {
	JobName string `json:"job_name"`
}

func isAgentOwnedJobMutationTool(toolName string) bool {
	switch toolName {
	case agentToolDeleteJob, agentToolStopJob, agentToolResubmitJob:
		return true
	default:
		return false
	}
}

func agentOwnedJobMutationActionName(toolName string) string {
	switch toolName {
	case agentToolDeleteJob:
		return "删除"
	case agentToolStopJob:
		return "停止"
	case agentToolResubmitJob:
		return "重提"
	default:
		return "操作"
	}
}

func (mgr *AgentMgr) validateOwnedJobMutationBeforeConfirmation(
	c *gin.Context,
	token util.JWTMessage,
	toolName string,
	rawArgs json.RawMessage,
) error {
	if !isAgentOwnedJobMutationTool(toolName) {
		return nil
	}

	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, "invalid args")
	}
	args.JobName = strings.TrimSpace(args.JobName)
	if args.JobName == "" {
		return bizerr.BadRequest.MissingParameter.New("job_name is required")
	}

	j := query.Job
	jobQuery := j.WithContext(c).Where(j.JobName.Eq(args.JobName))
	if token.RolePlatform != model.RoleAdmin {
		jobQuery = jobQuery.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	if _, err := jobQuery.First(); err != nil {
		return bizerr.Forbidden.PermissionDenied.New(
			fmt.Sprintf("该作业不存在或你没有访问权限，不能发起%s确认", agentOwnedJobMutationActionName(toolName)),
		)
	}
	return nil
}

func (mgr *AgentMgr) getOwnedJobForMutation(
	c *gin.Context,
	token util.JWTMessage,
	rawArgs json.RawMessage,
) (*model.Job, *batch.Job, error) {
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid args")
	}
	if args.JobName == "" {
		return nil, nil, bizerr.BadRequest.MissingParameter.New("job_name is required")
	}

	j := query.Job
	jobRecord, err := j.WithContext(c).
		Where(j.JobName.Eq(args.JobName)).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		First()
	if err != nil {
		return nil, nil, bizerr.NotFound.DataBaseNotFound.New("job not found")
	}

	clusterJob := &batch.Job{}
	if err := mgr.client.Get(
		c,
		client.ObjectKey{
			Name:      args.JobName,
			Namespace: pkgconfig.GetConfig().Namespaces.Job,
		},
		clusterJob,
	); err != nil {
		if k8serrors.IsNotFound(err) {
			return jobRecord, nil, nil
		}
		return nil, nil, bizerr.Internal.K8sServiceError.Wrap(err, "failed to load cluster job")
	}

	return jobRecord, clusterJob, nil
}

func (mgr *AgentMgr) deleteOwnedJob(
	c *gin.Context,
	jobRecord *model.Job,
	clusterJob *batch.Job,
	deleteTerminalRecord bool,
) (any, error) {
	j := query.Job
	shouldDeleteRecord := clusterJob == nil
	shouldDeleteJob := clusterJob != nil

	if clusterJob != nil {
		phase := clusterJob.Status.State.Phase
		if deleteTerminalRecord && (phase == batch.Failed ||
			phase == batch.Completed ||
			phase == batch.Aborted ||
			phase == batch.Terminated) {
			shouldDeleteRecord = true
		}
	}

	if shouldDeleteRecord {
		if _, err := j.WithContext(c).
			Where(j.JobName.Eq(jobRecord.JobName)).
			Where(j.UserID.Eq(jobRecord.UserID), j.AccountID.Eq(jobRecord.AccountID)).
			Delete(); err != nil {
			return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to delete job record")
		}
	} else {
		if _, err := j.WithContext(c).
			Where(j.JobName.Eq(jobRecord.JobName)).
			Where(j.UserID.Eq(jobRecord.UserID), j.AccountID.Eq(jobRecord.AccountID)).
			Updates(model.Job{
				Status:             model.Deleted,
				CompletedTimestamp: time.Now(),
			}); err != nil {
			return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to update job status")
		}
	}

	if shouldDeleteJob {
		if err := mgr.client.Delete(c, clusterJob); err != nil && !k8serrors.IsNotFound(err) {
			return nil, bizerr.Internal.K8sServiceError.Wrap(err, "failed to delete cluster job")
		}
	}

	return map[string]any{
		"jobName":       jobRecord.JobName,
		"status":        "deleted",
		"recordDeleted": shouldDeleteRecord,
	}, nil
}

func (mgr *AgentMgr) stopOwnedJob(c *gin.Context, jobRecord *model.Job, clusterJob *batch.Job) (any, error) {
	if clusterJob == nil {
		return map[string]any{
			"jobName": jobRecord.JobName,
			"status":  "already_stopped",
		}, nil
	}

	phase := clusterJob.Status.State.Phase
	if phase == batch.Completed || phase == batch.Failed || phase == batch.Aborted || phase == batch.Terminated {
		return map[string]any{
			"jobName": jobRecord.JobName,
			"status":  "already_finished",
		}, nil
	}

	if err := mgr.client.Delete(c, clusterJob); err != nil && !k8serrors.IsNotFound(err) {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "failed to delete cluster job")
	}
	j := query.Job
	if _, err := j.WithContext(c).
		Where(j.JobName.Eq(jobRecord.JobName)).
		Where(j.UserID.Eq(jobRecord.UserID), j.AccountID.Eq(jobRecord.AccountID)).
		Updates(model.Job{
			Status:             batch.Aborted,
			CompletedTimestamp: time.Now(),
		}); err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to update job status")
	}
	return map[string]any{
		"jobName": jobRecord.JobName,
		"status":  "stopped",
	}, nil
}

func (mgr *AgentMgr) ensureAgentResubmitAccess(c *gin.Context, job *batch.Job) error {
	serviceManager := crclient.NewServiceManager(mgr.client, mgr.kubeClient)
	labels := copyStringMap(job.Labels)
	if len(job.Spec.Tasks) == 0 {
		return nil
	}

	taskType := labels[crclient.LabelKeyTaskType]
	baseURL := labels[crclient.LabelKeyBaseURL]
	ownerRefs := []metav1.OwnerReference{
		*metav1.NewControllerRef(job, batch.SchemeGroupVersion.WithKind("Job")),
	}
	switch taskType {
	case string(model.JobTypeJupyter):
		_, err := serviceManager.CreateIngressWithPrefix(
			c,
			ownerRefs,
			labels,
			&v1.ServicePort{
				Name:       "notebook",
				Port:       8888,
				TargetPort: intstrFromInt(8888),
				Protocol:   v1.ProtocolTCP,
			},
			pkgconfig.GetConfig().Host,
			baseURL,
		)
		return err
	case string(model.JobTypeWebIDE):
		username := labels[crclient.LabelKeyTaskUser]
		randomPrefix := uuid.New().String()[:5]
		_, err := serviceManager.CreateNamedIngress(
			c,
			ownerRefs,
			labels,
			&v1.ServicePort{
				Name:       "webide",
				Port:       8888,
				TargetPort: intstrFromInt(8888),
				Protocol:   v1.ProtocolTCP,
			},
			pkgconfig.GetConfig().Host,
			username,
			randomPrefix,
		)
		return err
	default:
		return nil
	}
}

func getJobNamePrefix(jobName string) string {
	parts := strings.SplitN(jobName, "-", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "job"
}

func getBaseURLFromJobName(jobName string) string {
	parts := strings.SplitN(jobName, "-", 2)
	if len(parts) == 2 && parts[1] != "" {
		return parts[1]
	}
	return jobName
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func intstrFromInt(v int) intstr.IntOrString {
	return intstr.FromInt(v)
}

func mergeToolArgsWithPayload(baseArgs json.RawMessage, payload json.RawMessage) (json.RawMessage, error) {
	if len(payload) == 0 || string(payload) == "null" {
		return baseArgs, nil
	}

	base := make(map[string]any)
	if len(baseArgs) > 0 {
		if err := json.Unmarshal(baseArgs, &base); err != nil {
			return nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid stored tool args")
		}
	}

	incoming := make(map[string]any)
	if err := json.Unmarshal(payload, &incoming); err != nil {
		return nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid confirmation payload")
	}

	for key, value := range incoming {
		base[key] = value
	}

	merged, err := json.Marshal(base)
	if err != nil {
		return nil, bizerr.Internal.ServiceError.Wrap(err, "failed to merge confirmation payload")
	}
	return merged, nil
}

func applyResubmitOverrides(job *batch.Job, cpu *string, memory *string, gpuCount *int, gpuModel *string) (map[string]any, error) {
	if job == nil {
		return nil, bizerr.BadRequest.MissingParameter.New("job spec is unavailable for override")
	}

	applied := make(map[string]any)
	for taskIdx := range job.Spec.Tasks {
		task := &job.Spec.Tasks[taskIdx]
		for containerIdx := range task.Template.Spec.Containers {
			container := &task.Template.Spec.Containers[containerIdx]
			if cpu != nil {
				quantity, err := resource.ParseQuantity(strings.TrimSpace(*cpu))
				if err != nil {
					return nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid cpu override")
				}
				if container.Resources.Requests == nil {
					container.Resources.Requests = v1.ResourceList{}
				}
				container.Resources.Requests[v1.ResourceCPU] = quantity
				if container.Resources.Limits == nil {
					container.Resources.Limits = v1.ResourceList{}
				}
				container.Resources.Limits[v1.ResourceCPU] = quantity
				applied["cpu"] = quantity.String()
			}
			if memory != nil {
				quantity, err := resource.ParseQuantity(strings.TrimSpace(*memory))
				if err != nil {
					return nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid memory override")
				}
				if container.Resources.Requests == nil {
					container.Resources.Requests = v1.ResourceList{}
				}
				container.Resources.Requests[v1.ResourceMemory] = quantity
				if container.Resources.Limits == nil {
					container.Resources.Limits = v1.ResourceList{}
				}
				container.Resources.Limits[v1.ResourceMemory] = quantity
				applied["memory"] = quantity.String()
			}

			gpuResourceName, changed, err := overrideGPUResourceRequirements(
				&container.Resources,
				gpuCount,
				gpuModel,
			)
			if err != nil {
				return nil, err
			}
			if changed {
				if gpuCount != nil {
					applied["gpu_count"] = *gpuCount
				}
				if gpuResourceName != "" {
					applied["gpu_resource_name"] = gpuResourceName
					if gpuModel != nil && strings.TrimSpace(*gpuModel) != "" {
						applied["gpu_model"] = normalizeGPUModelName(*gpuModel)
					}
				}
			}
		}
	}

	if len(applied) == 0 {
		applied["inherit"] = "original_spec"
	}
	return applied, nil
}

func overrideGPUResourceRequirements(
	requirements *v1.ResourceRequirements,
	gpuCount *int,
	gpuModel *string,
) (string, bool, error) {
	if requirements == nil {
		return "", false, nil
	}
	currentGPUKey := detectGPUResourceName(requirements.Requests)
	if currentGPUKey == "" {
		currentGPUKey = detectGPUResourceName(requirements.Limits)
	}
	if currentGPUKey == "" && gpuCount == nil && gpuModel == nil {
		return "", false, nil
	}

	targetGPUKey := currentGPUKey
	if gpuModel != nil && strings.TrimSpace(*gpuModel) != "" {
		targetGPUKey = normalizeGPUResourceName(currentGPUKey, *gpuModel)
	}
	if targetGPUKey == "" && gpuCount != nil && *gpuCount > 0 {
		targetGPUKey = normalizeGPUResourceName(currentGPUKey, "gpu")
	}
	if targetGPUKey == "" {
		return "", false, nil
	}

	changed := false
	if requirements.Requests == nil {
		requirements.Requests = v1.ResourceList{}
	}
	if requirements.Limits == nil {
		requirements.Limits = v1.ResourceList{}
	}

	if currentGPUKey != "" && currentGPUKey != targetGPUKey {
		moveResourceQuantity(requirements.Requests, currentGPUKey, targetGPUKey)
		moveResourceQuantity(requirements.Limits, currentGPUKey, targetGPUKey)
		changed = true
	}
	if removeGPUResourcesExcept(requirements.Requests, targetGPUKey) {
		changed = true
	}
	if removeGPUResourcesExcept(requirements.Limits, targetGPUKey) {
		changed = true
	}

	if gpuCount != nil {
		if *gpuCount < 0 {
			return "", false, bizerr.BadRequest.ParameterError.New("gpu_count must be non-negative")
		}
		if *gpuCount == 0 {
			if removeGPUResourcesExcept(requirements.Requests, "") {
				changed = true
			}
			if removeGPUResourcesExcept(requirements.Limits, "") {
				changed = true
			}
			return "", changed, nil
		}
		quantity := *resource.NewQuantity(int64(*gpuCount), resource.DecimalSI)
		requirements.Requests[targetGPUKey] = quantity
		requirements.Limits[targetGPUKey] = quantity
		changed = true
	}

	return string(targetGPUKey), changed, nil
}
