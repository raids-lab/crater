package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/alert"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	pkgutils "github.com/raids-lab/crater/pkg/utils"
)

const (
	agentForwardTypeIngress  = 1
	agentForwardTypeNodePort = 2
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

func normalizeForwardTypeValue(value any) (int, error) {
	switch typed := value.(type) {
	case nil:
		return agentForwardTypeIngress, nil
	case float64:
		switch int(typed) {
		case agentForwardTypeIngress, agentForwardTypeNodePort:
			return int(typed), nil
		}
	case int:
		switch typed {
		case agentForwardTypeIngress, agentForwardTypeNodePort:
			return typed, nil
		}
	case string:
		switch strings.TrimSpace(strings.ToLower(typed)) {
		case "", "ingress":
			return agentForwardTypeIngress, nil
		case "nodeport", "node_port":
			return agentForwardTypeNodePort, nil
		case "1":
			return agentForwardTypeIngress, nil
		case "2":
			return agentForwardTypeNodePort, nil
		}
	}
	return 0, fmt.Errorf("forward type must be ingress or nodeport")
}

func parseForwardTextSpecs(raw string) ([]map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})
	result := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		parts := strings.Split(field, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return nil, fmt.Errorf("invalid forward spec %q, expected name:port[:ingress|nodeport]", field)
		}
		port, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || port <= 0 {
			return nil, fmt.Errorf("invalid forward port in %q", field)
		}
		forwardType := agentForwardTypeIngress
		if len(parts) == 3 {
			forwardType, err = normalizeForwardTypeValue(parts[2])
			if err != nil {
				return nil, err
			}
		}
		result = append(result, map[string]any{
			"type": forwardType,
			"name": strings.TrimSpace(parts[0]),
			"port": port,
		})
	}
	return result, nil
}

func parseForwardArgs(args map[string]any) ([]map[string]any, error) {
	raw, ok := args["forwards"]
	if ok && raw != nil {
		items, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("forwards must be a list")
		}
		result := make([]map[string]any, 0, len(items))
		for _, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("forwards entries must be objects")
			}
			name := getToolArgString(entry, "name", "")
			if name == "" {
				return nil, fmt.Errorf("forward name is required")
			}
			port := getToolArgInt(entry, "port", 0)
			if port <= 0 {
				return nil, fmt.Errorf("forward %q requires a positive port", name)
			}
			forwardType, err := normalizeForwardTypeValue(entry["type"])
			if err != nil {
				return nil, err
			}
			result = append(result, map[string]any{
				"type": forwardType,
				"name": name,
				"port": port,
			})
		}
		return result, nil
	}
	return parseForwardTextSpecs(getToolArgString(args, "forwards_text", ""))
}

func lookupToolArgValue(args map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		value, ok := args[key]
		if !ok || value == nil {
			continue
		}
		return value, true
	}
	return nil, false
}

func getToolArgStringAny(args map[string]any, fallback string, keys ...string) string {
	for _, key := range keys {
		if value := getToolArgString(args, key, ""); value != "" {
			return value
		}
	}
	return fallback
}

func getToolArgIntAny(args map[string]any, fallback int, keys ...string) int {
	for _, key := range keys {
		if value, ok := lookupToolArgValue(args, key); ok {
			switch typed := value.(type) {
			case float64:
				return int(typed)
			case int:
				return typed
			case int32:
				return int(typed)
			case int64:
				return int(typed)
			case string:
				parsed, err := strconv.Atoi(strings.TrimSpace(typed))
				if err == nil {
					return parsed
				}
			}
		}
	}
	return fallback
}

func parseDistributedPorts(raw any) ([]map[string]any, error) {
	if raw == nil {
		return nil, nil
	}

	if text, ok := raw.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, nil
		}
		if strings.HasPrefix(text, "[") {
			var decoded []any
			if err := json.Unmarshal([]byte(text), &decoded); err != nil {
				return nil, fmt.Errorf("ports_json must be a JSON array: %w", err)
			}
			raw = decoded
		} else {
			specs, err := parseForwardTextSpecs(text)
			if err != nil {
				return nil, err
			}
			ports := make([]map[string]any, 0, len(specs))
			for _, spec := range specs {
				ports = append(ports, map[string]any{
					"name": spec["name"],
					"port": spec["port"],
				})
			}
			return ports, nil
		}
	}

	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("ports must be a list or text specs")
	}

	ports := make([]map[string]any, 0, len(items))
	for idx, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("port #%d must be an object", idx+1)
		}
		name := getToolArgStringAny(entry, "", "name")
		if name == "" {
			return nil, fmt.Errorf("port #%d requires name", idx+1)
		}
		port := getToolArgIntAny(entry, 0, "port")
		if port <= 0 {
			return nil, fmt.Errorf("port %q requires a positive port number", name)
		}
		ports = append(ports, map[string]any{
			"name": name,
			"port": port,
		})
	}
	return ports, nil
}

func normalizeDistributedTask(entry map[string]any) (map[string]any, error) {
	name := getToolArgStringAny(entry, "", "name")
	if name == "" {
		return nil, fmt.Errorf("task name is required")
	}
	imageLink := getToolArgStringAny(entry, "", "image_link", "imageLink")
	if imageLink == "" {
		return nil, fmt.Errorf("task %q requires image_link", name)
	}

	replicas := getToolArgIntAny(entry, 1, "replicas")
	if replicas <= 0 {
		return nil, fmt.Errorf("task %q requires replicas > 0", name)
	}

	resourceMap := map[string]string{
		"cpu":    getToolArgStringAny(entry, "4", "cpu"),
		"memory": getToolArgStringAny(entry, "16Gi", "memory"),
	}
	if gpuCount := getToolArgIntAny(entry, 0, "gpu_count", "gpuCount"); gpuCount > 0 {
		gpuResourceName := normalizeGPUResourceName("", getToolArgStringAny(entry, "gpu", "gpu_model", "gpuModel"))
		resourceMap[string(gpuResourceName)] = strconv.Itoa(gpuCount)
	}

	ports, err := parseDistributedPorts(func() any {
		if value, ok := lookupToolArgValue(entry, "ports"); ok {
			return value
		}
		if value, ok := lookupToolArgValue(entry, "ports_text", "ports_json"); ok {
			return value
		}
		return nil
	}())
	if err != nil {
		return nil, err
	}

	task := map[string]any{
		"name":     name,
		"replicas": replicas,
		"resource": resourceMap,
		"image": map[string]any{
			"imageLink": imageLink,
			"archs":     []string{},
		},
		"ports": ports,
	}

	if command := getToolArgStringAny(entry, "", "command"); command != "" {
		task["command"] = command
		task["shell"] = getToolArgStringAny(entry, "bash", "shell")
	}
	if workingDir := getToolArgStringAny(entry, "", "working_dir", "workingDir"); workingDir != "" {
		task["workingDir"] = workingDir
	}

	return task, nil
}

func parseDistributedTasks(args map[string]any) ([]map[string]any, error) {
	if raw, ok := lookupToolArgValue(args, "tasks"); ok {
		items, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("tasks must be a list")
		}
		result := make([]map[string]any, 0, len(items))
		for idx, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("task #%d must be an object", idx+1)
			}
			task, err := normalizeDistributedTask(entry)
			if err != nil {
				return nil, err
			}
			result = append(result, task)
		}
		return result, nil
	}

	rawJSON := getToolArgString(args, "tasks_json", "")
	if rawJSON == "" {
		return nil, nil
	}
	var items []any
	if err := json.Unmarshal([]byte(rawJSON), &items); err != nil {
		return nil, fmt.Errorf("tasks_json must be a JSON array: %w", err)
	}
	result := make([]map[string]any, 0, len(items))
	for idx, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("task #%d must be an object", idx+1)
		}
		task, err := normalizeDistributedTask(entry)
		if err != nil {
			return nil, err
		}
		result = append(result, task)
	}
	return result, nil
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
	argsMap := parseToolArgsMap(rawArgs)
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
	forwards, err := parseForwardArgs(argsMap)
	if err != nil {
		return nil, err
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
		"forwards": forwards,
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

func (mgr *AgentMgr) toolCreateWebIDEJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	argsMap := parseToolArgsMap(rawArgs)
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
	if strings.TrimSpace(args.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(args.ImageLink) == "" {
		return nil, fmt.Errorf("image_link is required")
	}
	if strings.TrimSpace(args.CPU) == "" {
		args.CPU = "2"
	}
	if strings.TrimSpace(args.Memory) == "" {
		args.Memory = "8Gi"
	}
	if mgr.jobSubmitter == nil {
		return nil, fmt.Errorf("job submitter is not configured")
	}
	forwards, err := parseForwardArgs(argsMap)
	if err != nil {
		return nil, err
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
		"forwards": forwards,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal webide request: %w", err)
	}

	result, err := mgr.jobSubmitter.SubmitWebIDEJob(c, token, requestBody)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}

func (mgr *AgentMgr) toolMarkAuditHandled(_ *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
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

func (mgr *AgentMgr) toolNotifyJobOwner(c *gin.Context, _ util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobNames string `json:"job_names"`
		Subject  string `json:"subject"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(args.JobNames) == "" {
		return nil, fmt.Errorf("job_names is required")
	}
	subject := strings.TrimSpace(args.Subject)
	if subject == "" {
		subject = "作业通知"
	}
	message := args.Message
	if message == "" {
		message = "您的作业 GPU 利用率较低，请检查是否仍在使用，建议释放资源以供他人使用。"
	}
	jobNames := strings.Split(args.JobNames, ",")
	results := make([]map[string]any, 0, len(jobNames))
	seen := make(map[string]struct{}, len(jobNames))
	notifier := alert.GetAlertMgr()
	sent := 0
	skipped := 0
	failed := 0
	for _, name := range jobNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		entry := map[string]any{"job_name": name}
		err := notifier.NotifyJobOwner(c.Request.Context(), name, subject, message)
		switch {
		case err == nil:
			sent++
			entry["status"] = "sent"
		case errors.Is(err, alert.ErrReceiverEmailMissing):
			skipped++
			entry["status"] = "skipped"
			entry["reason"] = "owner_email_missing"
		default:
			failed++
			entry["status"] = "error"
			entry["message"] = err.Error()
		}
		results = append(results, entry)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("job_names contains no valid job names")
	}
	return map[string]any{
		"sent":     sent,
		"notified": sent,
		"skipped":  skipped,
		"failed":   failed,
		"total":    len(results),
		"subject":  subject,
		"message":  message,
		"results":  results,
	}, nil
}

func (mgr *AgentMgr) toolCreateTrainingJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	argsMap := parseToolArgsMap(rawArgs)
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
	forwards, err := parseForwardArgs(argsMap)
	if err != nil {
		return nil, err
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
		"forwards": forwards,
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

func (mgr *AgentMgr) toolCreateCustomJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	return mgr.toolCreateTrainingJob(c, token, rawArgs)
}

func (mgr *AgentMgr) toolCreatePytorchJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	argsMap := parseToolArgsMap(rawArgs)
	name := getToolArgString(argsMap, "name", "")
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	tasks, err := parseDistributedTasks(argsMap)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("tasks or tasks_json is required")
	}
	forwards, err := parseForwardArgs(argsMap)
	if err != nil {
		return nil, err
	}
	if mgr.jobSubmitter == nil {
		return nil, fmt.Errorf("job submitter is not configured")
	}

	requestBody, err := json.Marshal(map[string]any{
		"name":     name,
		"tasks":    tasks,
		"forwards": forwards,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pytorch request: %w", err)
	}

	result, err := mgr.jobSubmitter.SubmitPytorchJob(c, token, requestBody)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}

func (mgr *AgentMgr) toolCreateTensorflowJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	argsMap := parseToolArgsMap(rawArgs)
	name := getToolArgString(argsMap, "name", "")
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	tasks, err := parseDistributedTasks(argsMap)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("tasks or tasks_json is required")
	}
	forwards, err := parseForwardArgs(argsMap)
	if err != nil {
		return nil, err
	}
	if mgr.jobSubmitter == nil {
		return nil, fmt.Errorf("job submitter is not configured")
	}

	requestBody, err := json.Marshal(map[string]any{
		"name":     name,
		"tasks":    tasks,
		"forwards": forwards,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tensorflow request: %w", err)
	}

	result, err := mgr.jobSubmitter.SubmitTensorflowJob(c, token, requestBody)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}
