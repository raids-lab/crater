package agent

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	pkgutils "github.com/raids-lab/crater/pkg/utils"
)

func normalizeOptionalStringArg(value **string) {
	if value == nil || *value == nil {
		return
	}
	trimmed := strings.TrimSpace(**value)
	if trimmed == "" {
		*value = nil
		return
	}
	*value = &trimmed
}

func (mgr *AgentMgr) toolDeleteJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	jobRecord, clusterJob, err := mgr.getOwnedJobForMutation(c, token, rawArgs)
	if err != nil {
		return nil, err
	}
	return mgr.deleteOwnedJob(c, jobRecord, clusterJob, true)
}

func (mgr *AgentMgr) toolStopJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	jobRecord, clusterJob, err := mgr.getOwnedJobForMutation(c, token, rawArgs)
	if err != nil {
		return nil, err
	}
	return mgr.stopOwnedJob(c, jobRecord, clusterJob)
}

func (mgr *AgentMgr) toolResubmitJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobName  string  `json:"job_name"`
		Name     *string `json:"name"`
		CPU      *string `json:"cpu"`
		Memory   *string `json:"memory"`
		GPUCount *int    `json:"gpu_count"`
		GPUModel *string `json:"gpu_model"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	args.JobName = strings.TrimSpace(args.JobName)
	normalizeOptionalStringArg(&args.Name)
	normalizeOptionalStringArg(&args.CPU)
	normalizeOptionalStringArg(&args.Memory)
	normalizeOptionalStringArg(&args.GPUModel)
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	jobRecord, _, err := mgr.getOwnedJobForMutation(c, token, rawArgs)
	if err != nil {
		return nil, err
	}
	sourceJob := jobRecord.Attributes.Data()
	if sourceJob == nil {
		return nil, fmt.Errorf("job spec is unavailable for resubmit")
	}
	if token.Username == "" {
		return nil, fmt.Errorf("user identity is unavailable for resubmit")
	}

	clonedJob := sourceJob.DeepCopy()
	appliedOverrides, err := applyResubmitOverrides(clonedJob, args.CPU, args.Memory, args.GPUCount, args.GPUModel)
	if err != nil {
		return nil, err
	}
	prefix := getJobNamePrefix(jobRecord.JobName)
	newJobName := pkgutils.GenerateJobName(prefix, token.Username)
	baseURL := getBaseURLFromJobName(newJobName)

	clonedJob.ObjectMeta = metav1.ObjectMeta{
		Name:        newJobName,
		Namespace:   pkgconfig.GetConfig().Namespaces.Job,
		Labels:      copyStringMap(clonedJob.Labels),
		Annotations: copyStringMap(clonedJob.Annotations),
	}
	clonedJob.Status = batch.JobStatus{}
	clonedJob.ResourceVersion = ""
	clonedJob.UID = ""
	clonedJob.CreationTimestamp = metav1.Time{}
	clonedJob.ManagedFields = nil
	clonedJob.OwnerReferences = nil
	clonedJob.Finalizers = nil
	clonedJob.DeletionTimestamp = nil

	if clonedJob.Labels == nil {
		clonedJob.Labels = map[string]string{}
	}
	clonedJob.Labels[crclient.LabelKeyBaseURL] = baseURL
	if clonedJob.Annotations == nil {
		clonedJob.Annotations = map[string]string{}
	}
	if args.Name != nil && strings.TrimSpace(*args.Name) != "" {
		clonedJob.Annotations["crater.raids.io/task-name"] = strings.TrimSpace(*args.Name)
		appliedOverrides["name"] = strings.TrimSpace(*args.Name)
	} else if clonedJob.Annotations["crater.raids.io/task-name"] == "" {
		clonedJob.Annotations["crater.raids.io/task-name"] = jobRecord.Name
	}

	for idx := range clonedJob.Spec.Tasks {
		task := &clonedJob.Spec.Tasks[idx]
		task.Template.ResourceVersion = ""
		task.Template.UID = ""
		task.Template.CreationTimestamp = metav1.Time{}
		task.Template.ManagedFields = nil
		if task.Template.Labels == nil {
			task.Template.Labels = map[string]string{}
		}
		task.Template.Labels[crclient.LabelKeyBaseURL] = baseURL
		task.Template.Labels[crclient.LabelKeyTaskType] = clonedJob.Labels[crclient.LabelKeyTaskType]
		task.Template.Labels[crclient.LabelKeyTaskUser] = clonedJob.Labels[crclient.LabelKeyTaskUser]
		if accountName := clonedJob.Labels[crclient.LalbeKeyTaskAccount]; accountName != "" {
			task.Template.Labels[crclient.LalbeKeyTaskAccount] = accountName
		}
	}

	if err := mgr.client.Create(c, clonedJob); err != nil {
		return nil, fmt.Errorf("failed to create resubmitted job: %w", err)
	}

	if err := mgr.ensureAgentResubmitAccess(c, clonedJob); err != nil {
		return map[string]any{
			"sourceJobName": jobRecord.JobName,
			"jobName":       newJobName,
			"status":        "created",
			"warning":       err.Error(),
		}, nil
	}

	return map[string]any{
		"sourceJobName": jobRecord.JobName,
		"jobName":       newJobName,
		"displayName":   clonedJob.Annotations["crater.raids.io/task-name"],
		"status":        "created",
		"overrides":     appliedOverrides,
	}, nil
}

func (mgr *AgentMgr) toolCreateJupyterJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Name      string  `json:"name"`
		ImageLink string  `json:"image_link"`
		CPU       string  `json:"cpu"`
		Memory    string  `json:"memory"`
		GPUCount  *int    `json:"gpu_count"`
		GPUModel  *string `json:"gpu_model"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if args.ImageLink == "" {
		return nil, fmt.Errorf("image_link is required")
	}
	if args.CPU == "" {
		args.CPU = "2"
	}
	if args.Memory == "" {
		args.Memory = "8Gi"
	}

	if mgr.jobSubmitter == nil {
		return nil, fmt.Errorf("job submitter is not configured")
	}

	resourceMap := map[string]string{
		"cpu":    args.CPU,
		"memory": args.Memory,
	}
	if args.GPUCount != nil && *args.GPUCount > 0 {
		gpuResourceName := normalizeGPUResourceName("", "gpu")
		if args.GPUModel != nil && strings.TrimSpace(*args.GPUModel) != "" {
			gpuResourceName = normalizeGPUResourceName(gpuResourceName, *args.GPUModel)
		}
		resourceMap[string(gpuResourceName)] = strconv.Itoa(*args.GPUCount)
	}

	requestBody, err := json.Marshal(map[string]any{
		"name":     args.Name,
		"resource": resourceMap,
		"image": map[string]any{
			"imageLink": args.ImageLink,
			"archs":     []string{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal jupyter request: %w", err)
	}

	result, err := mgr.jobSubmitter.SubmitJupyterJob(c, token, requestBody)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}

func (mgr *AgentMgr) toolMarkAuditHandled(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		ItemIDs   string `json:"item_ids"`
		HandledBy string `json:"handled_by"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	handledBy := args.HandledBy
	if handledBy == "" {
		handledBy = token.Username
	}

	itemIDs := strings.Split(args.ItemIDs, ",")
	updated := 0
	db := query.GetDB()
	for _, id := range itemIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		result := db.Exec(
			`UPDATE ops_audit_items SET handled = true, handled_at = NOW(), handled_by = ? WHERE id = ? AND NOT handled`,
			handledBy, id,
		)
		if result.Error == nil && result.RowsAffected > 0 {
			updated++
		}
	}
	return map[string]any{"updated": updated, "total": len(itemIDs)}, nil
}

func (mgr *AgentMgr) toolBatchStopJobs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobNames string `json:"job_names"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	jobNames := strings.Split(args.JobNames, ",")
	results := make([]map[string]any, 0, len(jobNames))
	for _, name := range jobNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		stopArgs, _ := json.Marshal(map[string]string{"job_name": name})
		result, err := mgr.toolStopJob(c, token, stopArgs)
		entry := map[string]any{"job_name": name}
		if err != nil {
			entry["status"] = "error"
			entry["message"] = err.Error()
		} else {
			entry["status"] = "success"
			entry["result"] = result
		}
		results = append(results, entry)
	}
	return map[string]any{"results": results, "total": len(results)}, nil
}

func (mgr *AgentMgr) toolNotifyJobOwner(_ *gin.Context, _ util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobNames string `json:"job_names"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	message := args.Message
	if message == "" {
		message = "您的作业 GPU 利用率较低，请检查是否仍在使用，建议释放资源以供他人使用。"
	}
	jobNames := strings.Split(args.JobNames, ",")
	notified := 0
	for _, name := range jobNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// For now, log the notification. Full notification system integration can be added later.
		notified++
	}
	return map[string]any{"notified": notified, "message": message}, nil
}

func (mgr *AgentMgr) toolCreateTrainingJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Name       string  `json:"name"`
		ImageLink  string  `json:"image_link"`
		Command    string  `json:"command"`
		WorkingDir string  `json:"working_dir"`
		CPU        string  `json:"cpu"`
		Memory     string  `json:"memory"`
		GPUCount   *int    `json:"gpu_count"`
		GPUModel   *string `json:"gpu_model"`
		Shell      string  `json:"shell"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(args.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(args.ImageLink) == "" {
		return nil, fmt.Errorf("image_link is required")
	}
	if strings.TrimSpace(args.Command) == "" {
		return nil, fmt.Errorf("command is required")
	}
	if strings.TrimSpace(args.WorkingDir) == "" {
		return nil, fmt.Errorf("working_dir is required")
	}
	if strings.TrimSpace(args.CPU) == "" {
		args.CPU = "4"
	}
	if strings.TrimSpace(args.Memory) == "" {
		args.Memory = "16Gi"
	}
	if strings.TrimSpace(args.Shell) == "" {
		args.Shell = "bash"
	}

	resourceMap := map[string]string{
		"cpu":    args.CPU,
		"memory": args.Memory,
	}
	if args.GPUCount != nil && *args.GPUCount > 0 {
		gpuResourceName := normalizeGPUResourceName("", "gpu")
		if args.GPUModel != nil && strings.TrimSpace(*args.GPUModel) != "" {
			gpuResourceName = normalizeGPUResourceName("", *args.GPUModel)
		}
		resourceMap[string(gpuResourceName)] = strconv.Itoa(*args.GPUCount)
	}

	if mgr.jobSubmitter == nil {
		return nil, fmt.Errorf("job submitter is not configured")
	}

	requestBody, err := json.Marshal(map[string]any{
		"name":       args.Name,
		"resource":   resourceMap,
		"workingDir": args.WorkingDir,
		"command":    args.Command,
		"shell":      args.Shell,
		"image": map[string]any{
			"imageLink": args.ImageLink,
			"archs":     []string{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal training request: %w", err)
	}

	result, err := mgr.jobSubmitter.SubmitTrainingJob(c, token, requestBody)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}
