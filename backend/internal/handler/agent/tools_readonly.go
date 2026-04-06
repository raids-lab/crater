package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/monitor"
)

// ─── Individual tool implementations ─────────────────────────────────────────

func (mgr *AgentMgr) findScopedJob(ctx context.Context, token util.JWTMessage, jobName string) (*model.Job, error) {
	j := query.Job
	q := j.WithContext(ctx).
		Preload(j.User).
		Preload(j.Account).
		Where(j.JobName.Eq(jobName))
	if token.RolePlatform != model.RoleAdmin {
		q = q.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	job, err := q.First()
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	return job, nil
}

func buildJobDetailResponse(job *model.Job) map[string]any {
	resp := map[string]any{
		"jobName":            job.JobName,
		"name":               job.Name,
		"status":             job.Status,
		"jobType":            job.JobType,
		"creationTimestamp":  job.CreationTimestamp,
		"runningTimestamp":   job.RunningTimestamp,
		"completedTimestamp": job.CompletedTimestamp,
		"resources":          job.Resources.Data(),
	}
	if job.Nodes.Data() != nil {
		resp["nodes"] = job.Nodes.Data()
	}
	if job.ScheduleData != nil {
		resp["scheduleData"] = job.ScheduleData.Data()
	}
	if job.ProfileData != nil {
		resp["profileData"] = job.ProfileData.Data()
	}
	if job.TerminatedStates != nil {
		resp["terminatedStates"] = job.TerminatedStates.Data()
	}
	return resp
}

func getPrimaryContainerImage(job *model.Job) string {
	if job == nil || job.Attributes.Data() == nil {
		return ""
	}
	for i := range job.Attributes.Data().Spec.Tasks {
		task := &job.Attributes.Data().Spec.Tasks[i]
		for j := range task.Template.Spec.Containers {
			if image := task.Template.Spec.Containers[j].Image; image != "" {
				return image
			}
		}
	}
	return ""
}

func getFirstTerminatedState(job *model.Job) *v1.ContainerStateTerminated {
	if job == nil || job.TerminatedStates == nil {
		return nil
	}
	states := job.TerminatedStates.Data()
	if len(states) == 0 {
		return nil
	}
	return &states[0]
}

func filterLogByKeyword(logContent, keyword string) (string, error) {
	if keyword == "" {
		return logContent, nil
	}
	re, err := regexp.Compile(keyword)
	if err != nil {
		return "", fmt.Errorf("invalid keyword regex: %w", err)
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

func getMetricAlias(metric string) string {
	switch strings.TrimSpace(strings.ToLower(metric)) {
	case "gpu_utilization":
		return "gpu_util"
	case "gpu_memory":
		return "gpu_mem"
	case "gpu_mem_used":
		return "gpu_mem"
	case "cpu":
		return "cpu_usage"
	case "memory":
		return "mem_usage"
	case "cpu_mem_used":
		return "mem_usage"
	default:
		return strings.TrimSpace(strings.ToLower(metric))
	}
}

func normalizeMetricSelection(metrics []string) []string {
	if len(metrics) == 0 {
		return []string{"gpu_util", "gpu_mem", "cpu_usage", "mem_usage"}
	}
	seen := make(map[string]struct{}, len(metrics))
	normalized := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		alias := getMetricAlias(metric)
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		normalized = append(normalized, alias)
	}
	if len(normalized) == 0 {
		return []string{"gpu_util", "gpu_mem", "cpu_usage", "mem_usage"}
	}
	return normalized
}

func parseToolTimeRange(input string) time.Duration {
	switch strings.TrimSpace(strings.ToLower(input)) {
	case "", "last_2h":
		return 2 * time.Hour
	case "last_1h":
		return time.Hour
	case "last_6h":
		return 6 * time.Hour
	case "last_12h":
		return 12 * time.Hour
	case "last_24h":
		return 24 * time.Hour
	default:
		return 2 * time.Hour
	}
}

func buildMetricValueMap(profile *monitor.ProfileData, selected []string) map[string]any {
	if profile == nil {
		return map[string]any{}
	}
	get := func(v *float32) any {
		if v == nil {
			return nil
		}
		return *v
	}
	result := make(map[string]any, len(selected))
	for _, metric := range selected {
		switch metric {
		case "gpu_util":
			result[metric] = map[string]any{
				"avg": get(profile.GPUUtilAvg),
				"max": get(profile.GPUUtilMax),
				"std": get(profile.GPUUtilStd),
			}
		case "gpu_mem":
			result[metric] = map[string]any{
				"avg":   get(profile.GPUMemAvg),
				"max":   get(profile.GPUMemMax),
				"std":   get(profile.GPUMemStd),
				"total": get(profile.GPUMemTotal),
			}
		case "cpu_usage":
			result[metric] = map[string]any{
				"avg":     get(profile.CPUUsageAvg),
				"max":     get(profile.CPUUsageMax),
				"std":     get(profile.CPUUsageStd),
				"request": get(profile.CPURequest),
				"limit":   get(profile.CPULimit),
			}
		case "mem_usage":
			result[metric] = map[string]any{
				"avg":     get(profile.CPUMemAvg),
				"max":     get(profile.CPUMemMax),
				"std":     get(profile.CPUMemStd),
				"request": get(profile.MemRequest),
				"limit":   get(profile.MemLimit),
			}
		}
	}
	return result
}

func getJobNamespace(job *model.Job) string {
	if job != nil && job.Attributes.Data() != nil && job.Attributes.Data().Namespace != "" {
		return job.Attributes.Data().Namespace
	}
	return pkgconfig.GetConfig().Namespaces.Job
}

func getPodNameFromJob(job *model.Job) string {
	if job == nil || job.Attributes.Data() == nil {
		return ""
	}
	for i := range job.Attributes.Data().Spec.Tasks {
		task := &job.Attributes.Data().Spec.Tasks[i]
		if task.Name != "" {
			return fmt.Sprintf("%s-%s-0", job.JobName, task.Name)
		}
	}
	return ""
}

func (mgr *AgentMgr) readJobLogPayload(ctx context.Context, job *model.Job, tailLines int64, keyword string) (map[string]string, error) {
	if tailLines <= 0 {
		tailLines = 100
	}
	namespace := getJobNamespace(job)
	labelSelector := fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, job.JobName)
	if job.Attributes.Data() != nil {
		if labelVal, ok := job.Attributes.Data().Labels[crclient.LabelKeyBaseURL]; ok && labelVal != "" {
			labelSelector = fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, labelVal)
		}
	}
	podList, podErr := mgr.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
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

	logBytes, logErr := mgr.kubeClient.CoreV1().Pods(namespace).GetLogs(pod.Name, &v1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
	}).DoRaw(ctx)
	if logErr != nil {
		return map[string]string{"log": fmt.Sprintf("Failed to retrieve logs: %v", logErr)}, nil
	}

	logContent, filterErr := filterLogByKeyword(string(logBytes), keyword)
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

func getFailureCategory(job *model.Job) string {
	return handler.CategorizeFailure(job).TypeName
}

func getFailureSimilarityScore(target, candidate *model.Job) int {
	score := 0
	if getFailureCategory(candidate) == getFailureCategory(target) {
		score += 5
	}
	targetTerminated := getFirstTerminatedState(target)
	candidateTerminated := getFirstTerminatedState(candidate)
	if targetTerminated != nil && candidateTerminated != nil {
		if targetTerminated.ExitCode != 0 && targetTerminated.ExitCode == candidateTerminated.ExitCode {
			score += 3
		}
		if targetTerminated.Reason != "" && strings.EqualFold(targetTerminated.Reason, candidateTerminated.Reason) {
			score += 2
		}
	}
	if target.JobType == candidate.JobType {
		score += 2
	}
	targetImage := getPrimaryContainerImage(target)
	candidateImage := getPrimaryContainerImage(candidate)
	if targetImage != "" && targetImage == candidateImage {
		score += 1
	}
	return score
}

func buildSimilarFailureEntry(job *model.Job, score int) map[string]any {
	entry := map[string]any{
		"jobName":            job.JobName,
		"name":               job.Name,
		"status":             job.Status,
		"jobType":            job.JobType,
		"category":           getFailureCategory(job),
		"similarityScore":    score,
		"completedTimestamp": job.CompletedTimestamp,
	}
	if ts := getFirstTerminatedState(job); ts != nil {
		entry["exitCode"] = ts.ExitCode
		entry["exitReason"] = ts.Reason
	}
	if image := getPrimaryContainerImage(job); image != "" {
		entry["image"] = image
	}
	return entry
}

// toolGetJobDetail returns job detail from the database.
func (mgr *AgentMgr) toolGetJobDetail(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}
	return buildJobDetailResponse(job), nil
}

// toolGetJobEvents returns events for a job stored in the database cache.
func (mgr *AgentMgr) toolGetJobEvents(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}
	if job.Events == nil {
		return []v1.Event{}, nil
	}
	return job.Events.Data(), nil
}

// toolGetJobLogs retrieves recent log lines for a job's first pod via the Kubernetes API.
func (mgr *AgentMgr) toolGetJobLogs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobName   string `json:"job_name"`
		Tail      int64  `json:"tail"`
		TailLines int64  `json:"tail_lines"`
		Keyword   string `json:"keyword"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}
	if args.TailLines <= 0 {
		args.TailLines = args.Tail
	}
	if args.TailLines <= 0 {
		args.TailLines = 100
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}
	return mgr.readJobLogPayload(c.Request.Context(), job, args.TailLines, args.Keyword)
}

// toolDiagnoseJob runs the existing rule-based diagnosis on a job.
func (mgr *AgentMgr) toolDiagnoseJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}
	return handler.PerformDiagnosis(job), nil
}

func (mgr *AgentMgr) toolGetDiagnosticContext(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobName    string `json:"job_name"`
		IncludeLog *bool  `json:"include_log"`
		TailLines  int64  `json:"tail_lines"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}
	if args.TailLines <= 0 {
		args.TailLines = 200
	}
	includeLog := true
	if args.IncludeLog != nil {
		includeLog = *args.IncludeLog
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}

	resp := handler.JobContextResp{}
	resp.Meta.Name = job.Name
	resp.Meta.JobName = job.JobName
	resp.Meta.Namespace = getJobNamespace(job)
	if job.User.ID != 0 {
		resp.Meta.User = job.User.Name
	}
	if job.Account.ID != 0 {
		resp.Meta.Queue = job.Account.Nickname
	}
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

	if includeLog {
		logPayload, logErr := mgr.readJobLogPayload(c.Request.Context(), job, args.TailLines, "")
		if logErr != nil {
			return nil, logErr
		}
		resp.Log.Container = logPayload["container"]
		resp.Log.Tail = logPayload["log"]
	}

	return resp, nil
}

func (mgr *AgentMgr) toolQueryJobMetrics(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobName   string   `json:"job_name"`
		Metrics   []string `json:"metrics"`
		TimeRange string   `json:"time_range"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}

	selectedMetrics := normalizeMetricSelection(args.Metrics)
	timeRange := strings.TrimSpace(args.TimeRange)
	if timeRange == "" {
		timeRange = "last_2h"
	}

	var profile *monitor.ProfileData
	source := "persisted_profile"
	if mgr.promClient != nil {
		namespace := getJobNamespace(job)
		podName := getPodNameFromJob(job)
		if namespace != "" && podName != "" {
			profile = mgr.promClient.QueryProfileData(types.NamespacedName{
				Namespace: namespace,
				Name:      podName,
			}, time.Now().Add(-parseToolTimeRange(timeRange)))
			if profile != nil {
				source = "prometheus_live"
			}
		}
	}
	if profile == nil && job.ProfileData != nil {
		profile = job.ProfileData.Data()
		source = "persisted_profile"
	}

	return map[string]any{
		"jobName":   job.JobName,
		"timeRange": timeRange,
		"source":    source,
		"metrics":   buildMetricValueMap(profile, selectedMetrics),
	}, nil
}

func (mgr *AgentMgr) toolSearchSimilarFailures(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobName string `json:"job_name"`
		Days    int    `json:"days"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}
	if args.Days <= 0 {
		args.Days = 30
	}
	if args.Limit <= 0 || args.Limit > 20 {
		args.Limit = 5
	}

	targetJob, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}

	j := query.Job
	q := j.WithContext(c).
		Preload(j.User).
		Preload(j.Account).
		Where(j.JobName.Neq(targetJob.JobName)).
		Where(j.Status.Eq(string(batch.Failed))).
		Where(j.CompletedTimestamp.Gte(time.Now().AddDate(0, 0, -args.Days)))
	if token.RolePlatform != model.RoleAdmin {
		q = q.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	candidates, err := q.Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query similar failures: %w", err)
	}

	type scoredFailure struct {
		job   *model.Job
		score int
	}
	scored := make([]scoredFailure, 0, len(candidates))
	for _, candidate := range candidates {
		score := getFailureSimilarityScore(targetJob, candidate)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredFailure{job: candidate, score: score})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].job.CompletedTimestamp.After(scored[j].job.CompletedTimestamp)
		}
		return scored[i].score > scored[j].score
	})
	if len(scored) > args.Limit {
		scored = scored[:args.Limit]
	}

	items := make([]map[string]any, 0, len(scored))
	for _, entry := range scored {
		items = append(items, buildSimilarFailureEntry(entry.job, entry.score))
	}

	return map[string]any{
		"jobName":         targetJob.JobName,
		"targetCategory":  getFailureCategory(targetJob),
		"lookbackDays":    args.Days,
		"matches":         items,
		"targetJobType":   targetJob.JobType,
		"targetExitState": getFirstTerminatedState(targetJob),
	}, nil
}

func (mgr *AgentMgr) toolGetRealtimeCapacity(c *gin.Context, _ util.JWTMessage, _ json.RawMessage) (any, error) {
	nodes, err := mgr.nodeClient.ListNodes(c.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	statusCount := make(map[string]int)
	items := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		statusCount[string(node.Status)]++
		items = append(items, map[string]any{
			"name":        node.Name,
			"status":      node.Status,
			"role":        node.Role,
			"vendor":      node.Vendor,
			"workloads":   node.Workloads,
			"capacity":    node.Capacity,
			"allocatable": node.Allocatable,
			"used":        node.Used,
		})
	}
	return map[string]any{
		"scope":       "cluster",
		"statusCount": statusCount,
		"nodeCount":   len(nodes),
		"nodes":       items,
	}, nil
}

func (mgr *AgentMgr) toolListAvailableImages(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobType string `json:"job_type"`
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
	}
	if len(rawArgs) > 0 {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return nil, fmt.Errorf("invalid args: %w", err)
		}
	}
	if args.Limit <= 0 || args.Limit > 100 {
		args.Limit = 20
	}

	images, err := mgr.listAccessibleImages(c, token)
	if err != nil {
		return nil, err
	}

	filtered := make([]agentAccessibleImage, 0, len(images))
	for _, item := range images {
		if !matchesImageJobType(item.Image.TaskType, args.JobType) {
			continue
		}
		if !matchesImageKeyword(item.Image, args.Keyword) {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Image.CreatedAt.After(filtered[j].Image.CreatedAt)
	})
	if len(filtered) > args.Limit {
		filtered = filtered[:args.Limit]
	}

	items := make([]map[string]any, 0, len(filtered))
	for _, item := range filtered {
		items = append(items, buildAgentImageSummary(item))
	}

	return map[string]any{
		"images":            items,
		"count":             len(items),
		"requestedJobType":  strings.TrimSpace(args.JobType),
		"requestedKeyword":  strings.TrimSpace(args.Keyword),
		"supportsRealImage": true,
	}, nil
}

func (mgr *AgentMgr) toolListCudaBaseImages(c *gin.Context, _ util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Limit int `json:"limit"`
	}
	if len(rawArgs) > 0 {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return nil, fmt.Errorf("invalid args: %w", err)
		}
	}
	if args.Limit <= 0 || args.Limit > 100 {
		args.Limit = 20
	}

	cudaQuery := query.CudaBaseImage
	images, err := cudaQuery.WithContext(c).
		Order(cudaQuery.CreatedAt.Desc()).
		Limit(args.Limit).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list cuda base images: %w", err)
	}

	items := make([]map[string]any, 0, len(images))
	for _, item := range images {
		items = append(items, map[string]any{
			"id":         item.ID,
			"label":      item.Label,
			"imageLabel": item.ImageLabel,
			"value":      item.Value,
			"createdAt":  item.CreatedAt,
		})
	}

	return map[string]any{
		"images": items,
		"count":  len(items),
	}, nil
}

func (mgr *AgentMgr) toolListAvailableGPUModels(c *gin.Context, _ util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Limit int `json:"limit"`
	}
	if len(rawArgs) > 0 {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return nil, fmt.Errorf("invalid args: %w", err)
		}
	}
	if args.Limit <= 0 || args.Limit > 100 {
		args.Limit = 20
	}

	nodes, err := mgr.nodeClient.ListNodes(c.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	type gpuSummary struct {
		ResourceName string
		GPUModel     string
		Vendor       string
		NodeCount    int
		Total        int64
		Used         int64
		Free         int64
	}

	summaryByName := make(map[string]*gpuSummary)
	for _, node := range nodes {
		for resourceName, quantity := range node.Allocatable {
			name := string(resourceName)
			if !isGPUResourceName(name) {
				continue
			}
			total := quantity.Value()
			used := int64(0)
			if usedQuantity, ok := node.Used[resourceName]; ok {
				used = usedQuantity.Value()
			}
			entry := summaryByName[name]
			if entry == nil {
				vendor := ""
				modelName := extractGPUModelFromResourceName(name)
				if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
					vendor = parts[0]
				}
				entry = &gpuSummary{
					ResourceName: name,
					GPUModel:     modelName,
					Vendor:       vendor,
				}
				summaryByName[name] = entry
			}
			entry.NodeCount++
			entry.Total += total
			entry.Used += used
			entry.Free += total - used
		}
	}

	items := make([]map[string]any, 0, len(summaryByName))
	for _, item := range summaryByName {
		items = append(items, map[string]any{
			"resourceName": item.ResourceName,
			"gpuModel":     item.GPUModel,
			"vendor":       item.Vendor,
			"nodeCount":    item.NodeCount,
			"total":        item.Total,
			"used":         item.Used,
			"free":         item.Free,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		leftFree, _ := items[i]["free"].(int64)
		rightFree, _ := items[j]["free"].(int64)
		if leftFree == rightFree {
			leftTotal, _ := items[i]["total"].(int64)
			rightTotal, _ := items[j]["total"].(int64)
			return leftTotal > rightTotal
		}
		return leftFree > rightFree
	})
	if len(items) > args.Limit {
		items = items[:args.Limit]
	}

	return map[string]any{
		"gpuModels": items,
		"count":     len(items),
	}, nil
}

func (mgr *AgentMgr) toolRecommendTrainingImages(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		TaskDescription string `json:"task_description"`
		Framework       string `json:"framework"`
		Limit           int    `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(args.TaskDescription) == "" {
		return nil, fmt.Errorf("task_description is required")
	}
	if args.Limit <= 0 || args.Limit > 20 {
		args.Limit = 5
	}

	images, err := mgr.listAccessibleImages(c, token)
	if err != nil {
		return nil, err
	}

	keywords := buildTrainingImageKeywords(args.TaskDescription, args.Framework)
	type scoredImage struct {
		Item    agentAccessibleImage
		Score   int
		Reasons []string
	}
	scored := make([]scoredImage, 0, len(images))
	for _, item := range images {
		score, reasons := scoreTrainingImage(item, keywords)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredImage{
			Item:    item,
			Score:   score,
			Reasons: reasons,
		})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].Item.Image.CreatedAt.After(scored[j].Item.Image.CreatedAt)
		}
		return scored[i].Score > scored[j].Score
	})
	if len(scored) > args.Limit {
		scored = scored[:args.Limit]
	}

	items := make([]map[string]any, 0, len(scored))
	for _, entry := range scored {
		summary := buildAgentImageSummary(entry.Item)
		summary["score"] = entry.Score
		summary["reasons"] = entry.Reasons
		summary["confidence"] = recommendationConfidence(entry.Score)
		items = append(items, summary)
	}

	return map[string]any{
		"taskDescription": args.TaskDescription,
		"framework":       strings.TrimSpace(args.Framework),
		"recommendations": items,
		"count":           len(items),
		"grounded":        "Recommendations are based only on currently visible Crater images",
	}, nil
}

func (mgr *AgentMgr) toolAnalyzeQueueStatus(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}

	pendingReasons := make([]string, 0)
	if job.Events != nil {
		for _, event := range job.Events.Data() {
			if event.Reason == "FailedScheduling" && event.Message != "" {
				pendingReasons = append(pendingReasons, event.Message)
			}
		}
	}

	diagnosis := handler.PerformDiagnosis(job)
	suggestions := make([]string, 0, 3)
	if diagnosis.Solution != "" {
		suggestions = append(suggestions, diagnosis.Solution)
	}
	if len(pendingReasons) == 0 && job.Status == batch.Pending {
		suggestions = append(suggestions, "暂无明确调度事件，建议先查看配额与节点实时容量。")
	}

	capacity, capacityErr := mgr.toolGetRealtimeCapacity(c, token, nil)
	quota, quotaErr := mgr.toolCheckQuota(c, token, nil)
	resp := map[string]any{
		"jobName":        job.JobName,
		"status":         job.Status,
		"category":       diagnosis.Category,
		"diagnosis":      diagnosis.Diagnosis,
		"pendingReasons": pendingReasons,
		"suggestions":    suggestions,
		"jobDetail":      buildJobDetailResponse(job),
	}
	if capacityErr == nil {
		resp["capacity"] = capacity
	}
	if quotaErr == nil {
		resp["quota"] = quota
	}
	return resp, nil
}

// toolCheckQuota returns the current resource quota for the user.
func (mgr *AgentMgr) toolCheckQuota(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	accountID := token.AccountID
	var args struct {
		AccountID *uint `json:"account_id"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.AccountID != nil {
		if token.RolePlatform != model.RoleAdmin && *args.AccountID != token.AccountID {
			return nil, fmt.Errorf("account_id is not accessible")
		}
		accountID = *args.AccountID
	}

	a := query.Account
	ua := query.UserAccount
	userAccount, err := ua.WithContext(c).
		Where(ua.AccountID.Eq(accountID), ua.UserID.Eq(token.UserID)).
		First()
	if err == nil {
		quota := userAccount.Quota.Data()
		return map[string]any{
			"accountId":  accountID,
			"source":     "user_account",
			"capability": quota.Capability,
		}, nil
	}

	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("user account not found: %w", err)
	}

	account, accountErr := a.WithContext(c).Where(a.ID.Eq(accountID)).First()
	if accountErr != nil {
		return nil, fmt.Errorf("account not found: %w", accountErr)
	}

	return map[string]any{
		"accountId":  accountID,
		"source":     "account",
		"capability": account.Quota.Data().Capability,
	}, nil
}

// toolGetHealthOverview returns a simplified health summary for the current user.
func (mgr *AgentMgr) toolGetHealthOverview(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Days int `json:"days"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 7
	}

	j := query.Job
	lookback := time.Now().AddDate(0, 0, -args.Days)
	jobs, err := j.WithContext(c).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		Where(j.CreationTimestamp.Gte(lookback)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}

	statusCount := make(map[string]int)
	for _, job := range jobs {
		statusCount[string(job.Status)]++
	}

	return map[string]any{
		"totalJobs":    len(jobs),
		"statusCount":  statusCount,
		"lookbackDays": args.Days,
	}, nil
}

func (mgr *AgentMgr) toolGetClusterHealthOverview(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("cluster overview requires admin privileges")
	}
	var args struct {
		Days int `json:"days"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 7
	}

	j := query.Job
	q := j.WithContext(c)
	if args.Days > 0 {
		q = q.Where(j.CreationTimestamp.Gte(time.Now().AddDate(0, 0, -args.Days)))
	}
	jobs, err := q.Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query cluster jobs: %w", err)
	}

	statusCount := make(map[string]int)
	accountCount := make(map[uint]int)
	userCount := make(map[uint]int)
	for _, job := range jobs {
		statusCount[string(job.Status)]++
		accountCount[job.AccountID]++
		userCount[job.UserID]++
	}

	return map[string]any{
		"scope":        "cluster",
		"totalJobs":    len(jobs),
		"statusCount":  statusCount,
		"lookbackDays": args.Days,
		"accountCount": len(accountCount),
		"userCount":    len(userCount),
	}, nil
}

func (mgr *AgentMgr) toolListUserJobs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Statuses []string `json:"statuses"`
		Days     int      `json:"days"`
		Limit    int      `json:"limit"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 30
	}
	if args.Limit <= 0 || args.Limit > 50 {
		args.Limit = 20
	}

	j := query.Job
	q := j.WithContext(c).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		Where(j.CreationTimestamp.Gte(time.Now().AddDate(0, 0, -args.Days))).
		Order(j.CreationTimestamp.Desc())

	if len(args.Statuses) > 0 {
		statuses := normalizeJobStatuses(args.Statuses)
		if len(statuses) > 0 {
			q = q.Where(j.Status.In(statuses...))
		}
	}

	jobs, err := q.Limit(args.Limit).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	items := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, map[string]any{
			"name":               job.Name,
			"jobName":            job.JobName,
			"jobType":            job.JobType,
			"status":             job.Status,
			"creationTimestamp":  job.CreationTimestamp,
			"runningTimestamp":   job.RunningTimestamp,
			"completedTimestamp": job.CompletedTimestamp,
		})
	}

	return map[string]any{
		"jobs":      items,
		"count":     len(items),
		"days":      args.Days,
		"requested": args.Statuses,
	}, nil
}

func (mgr *AgentMgr) toolListClusterJobs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("cluster job listing requires admin privileges")
	}
	var args struct {
		Statuses []string `json:"statuses"`
		Days     int      `json:"days"`
		Limit    int      `json:"limit"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 7
	}
	if args.Limit <= 0 || args.Limit > 100 {
		args.Limit = 30
	}

	j := query.Job
	q := j.WithContext(c).
		Where(j.CreationTimestamp.Gte(time.Now().AddDate(0, 0, -args.Days))).
		Order(j.CreationTimestamp.Desc())

	if len(args.Statuses) > 0 {
		statuses := normalizeJobStatuses(args.Statuses)
		if len(statuses) > 0 {
			q = q.Where(j.Status.In(statuses...))
		}
	}

	jobs, err := q.Limit(args.Limit).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster jobs: %w", err)
	}

	items := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, map[string]any{
			"name":               job.Name,
			"jobName":            job.JobName,
			"jobType":            job.JobType,
			"status":             job.Status,
			"userID":             job.UserID,
			"accountID":          job.AccountID,
			"creationTimestamp":  job.CreationTimestamp,
			"runningTimestamp":   job.RunningTimestamp,
			"completedTimestamp": job.CompletedTimestamp,
		})
	}

	return map[string]any{
		"scope":     "cluster",
		"jobs":      items,
		"count":     len(items),
		"days":      args.Days,
		"requested": args.Statuses,
	}, nil
}

func (mgr *AgentMgr) toolListClusterNodes(c *gin.Context, token util.JWTMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("cluster node listing requires admin privileges")
	}
	nodes, err := mgr.nodeClient.ListNodes(c.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster nodes: %w", err)
	}

	statusCount := make(map[string]int)
	items := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		statusCount[string(node.Status)]++
		items = append(items, map[string]any{
			"name":      node.Name,
			"role":      node.Role,
			"status":    node.Status,
			"workloads": node.Workloads,
			"vendor":    node.Vendor,
			"address":   node.Address,
		})
	}

	return map[string]any{
		"scope":       "cluster",
		"count":       len(items),
		"statusCount": statusCount,
		"nodes":       items,
	}, nil
}

// ─── New tools ──────────────────────────────────────────────────────────────

func jobDisplayUser(job *model.Job) string {
	if job == nil {
		return ""
	}
	if strings.TrimSpace(job.User.Nickname) != "" {
		return job.User.Nickname
	}
	return strings.TrimSpace(job.User.Name)
}

func extractRequestedGPUCount(resources v1.ResourceList) int {
	total := 0
	for name, quantity := range resources {
		if isGPUResourceName(string(name)) {
			total += int(quantity.Value())
		}
	}
	return total
}

func estimateActualGPUUsage(gpuUtilization float64, requested int) int {
	if requested <= 0 || gpuUtilization <= 0 {
		return 0
	}
	if requested == 1 {
		return 1
	}
	estimated := int((gpuUtilization / 100.0) * float64(requested))
	if estimated <= 0 {
		estimated = 1
	}
	if estimated > requested {
		estimated = requested
	}
	return estimated
}

type idleEntry struct {
	JobName            string   `json:"job_name"`
	Name               string   `json:"name"`
	JobType            string   `json:"job_type"`
	UserID             uint     `json:"user_id,omitempty"`
	AccountID          uint     `json:"account_id,omitempty"`
	Username           string   `json:"username,omitempty"`
	GPUUtilAvg         *float32 `json:"gpu_util_avg"`
	GPUMemAvg          *float32 `json:"gpu_mem_avg"`
	GPUUtilization     float64  `json:"gpu_utilization"`
	GPUCount           int      `json:"gpu_count"`
	ActualGPUUsed      int      `json:"gpu_actual_used"`
	IdleHours          float64  `json:"idle_hours"`
	RunningHours       float64  `json:"running_duration_hours"`
	IsIdle             bool     `json:"is_idle"`
	IdleReason         string   `json:"idle_reason,omitempty"`
	Resources          any      `json:"resources"`
	RequestedResources any      `json:"requested_resources,omitempty"`
}

// toolDetectIdleJobs finds running jobs with low GPU utilization.
func (mgr *AgentMgr) toolDetectIdleJobs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		GPUThreshold int `json:"gpu_threshold"`
		Hours        int `json:"hours"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.GPUThreshold <= 0 {
		args.GPUThreshold = 5
	}
	if args.Hours <= 0 {
		args.Hours = 24
	}

	j := query.Job
	q := j.WithContext(c).
		Preload(j.User).
		Where(j.Status.Eq(string(batch.Running)))
	if token.RolePlatform != model.RoleAdmin {
		q = q.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	runningJobs, err := q.Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query running jobs: %w", err)
	}

	idleJobs := make([]idleEntry, 0)
	normalJobs := make([]idleEntry, 0)
	wastedGPU := int64(0)

	for _, job := range runningJobs {
		entry := idleEntry{
			JobName:            job.JobName,
			Name:               job.Name,
			JobType:            string(job.JobType),
			UserID:             job.UserID,
			AccountID:          job.AccountID,
			Username:           jobDisplayUser(job),
			Resources:          job.Resources.Data(),
			RequestedResources: job.Resources.Data(),
			GPUCount:           extractRequestedGPUCount(job.Resources.Data()),
		}
		if !job.RunningTimestamp.IsZero() {
			entry.RunningHours = time.Since(job.RunningTimestamp).Hours()
			entry.IdleHours = entry.RunningHours
		}

		var profile *monitor.ProfileData
		if mgr.promClient != nil {
			namespace := getJobNamespace(job)
			podName := getPodNameFromJob(job)
			if namespace != "" && podName != "" {
				profile = mgr.promClient.QueryProfileData(types.NamespacedName{
					Namespace: namespace,
					Name:      podName,
				}, time.Now().Add(-time.Duration(args.Hours)*time.Hour))
			}
		}
		if profile == nil && job.ProfileData != nil {
			profile = job.ProfileData.Data()
		}
		if profile != nil {
			entry.GPUUtilAvg = profile.GPUUtilAvg
			entry.GPUMemAvg = profile.GPUMemAvg
			if profile.GPUUtilAvg != nil {
				entry.GPUUtilization = float64(*profile.GPUUtilAvg)
			}
			entry.ActualGPUUsed = estimateActualGPUUsage(entry.GPUUtilization, entry.GPUCount)
		}

		isIdle := false
		if profile != nil && profile.GPUUtilAvg != nil && *profile.GPUUtilAvg < float32(args.GPUThreshold) {
			isIdle = true
			entry.IdleReason = fmt.Sprintf("GPU utilization %.1f%% < %d%% threshold", *profile.GPUUtilAvg, args.GPUThreshold)
		}
		entry.IsIdle = isIdle

		if isIdle {
			idleJobs = append(idleJobs, entry)
			// Count wasted GPU from resources
			for resName, qty := range job.Resources.Data() {
				if isGPUResourceName(string(resName)) {
					wastedGPU += qty.Value()
				}
			}
		} else {
			normalJobs = append(normalJobs, entry)
		}
	}

	scope := "user"
	if token.RolePlatform == model.RoleAdmin {
		scope = "cluster_wide"
	}

	return map[string]any{
		"scope":       scope,
		"idle_jobs":   idleJobs,
		"normal_jobs": normalJobs,
		"summary": map[string]any{
			"total_running_jobs":        len(runningJobs),
			"idle_job_count":            len(idleJobs),
			"total_wasted_gpu":          wastedGPU,
			"estimated_gpu_waste_hours": float64(wastedGPU) * float64(args.Hours),
			"gpu_threshold_pct":         args.GPUThreshold,
			"lookback_hours":            args.Hours,
		},
	}, nil
}

// toolGetJobTemplates lists available job templates.
func (mgr *AgentMgr) toolGetJobTemplates(c *gin.Context, _ util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Limit int `json:"limit"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Limit <= 0 || args.Limit > 50 {
		args.Limit = 20
	}

	jt := query.Jobtemplate
	templates, err := jt.WithContext(c).
		Preload(jt.User).
		Order(jt.CreatedAt.Desc()).
		Limit(args.Limit).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list job templates: %w", err)
	}

	items := make([]map[string]any, 0, len(templates))
	for _, t := range templates {
		items = append(items, map[string]any{
			"id":        t.ID,
			"name":      t.Name,
			"describe":  t.Describe,
			"document":  t.Document,
			"template":  t.Template,
			"createdAt": t.CreatedAt,
			"owner": map[string]any{
				"userID":   t.User.ID,
				"username": t.User.Name,
				"nickname": t.User.Nickname,
			},
		})
	}

	return map[string]any{
		"templates": items,
		"count":     len(items),
	}, nil
}

// toolGetFailureStatistics aggregates failure categories over recent jobs.
func (mgr *AgentMgr) toolGetFailureStatistics(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Days  int `json:"days"`
		Limit int `json:"limit"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 7
	}
	if args.Limit <= 0 || args.Limit > 20 {
		args.Limit = 10
	}

	j := query.Job
	q := j.WithContext(c).
		Where(j.Status.Eq(string(batch.Failed))).
		Where(j.CompletedTimestamp.Gte(time.Now().AddDate(0, 0, -args.Days)))
	if token.RolePlatform != model.RoleAdmin {
		q = q.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	failedJobs, err := q.Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query failed jobs: %w", err)
	}

	type categoryAgg struct {
		count   int
		samples []string
	}
	categories := make(map[string]*categoryAgg)
	for _, job := range failedJobs {
		cat := getFailureCategory(job)
		if categories[cat] == nil {
			categories[cat] = &categoryAgg{}
		}
		categories[cat].count++
		if len(categories[cat].samples) < 3 {
			categories[cat].samples = append(categories[cat].samples, job.JobName)
		}
	}

	type statEntry struct {
		Category string   `json:"category"`
		Count    int      `json:"count"`
		Samples  []string `json:"samples"`
	}
	stats := make([]statEntry, 0, len(categories))
	for cat, agg := range categories {
		stats = append(stats, statEntry{Category: cat, Count: agg.count, Samples: agg.samples})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Count > stats[j].Count })
	if len(stats) > args.Limit {
		stats = stats[:args.Limit]
	}

	return map[string]any{
		"totalFailed":  len(failedJobs),
		"lookbackDays": args.Days,
		"categories":   stats,
	}, nil
}

// toolGetClusterHealthReport aggregates cluster health into a single report (admin only).
func (mgr *AgentMgr) toolGetClusterHealthReport(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("cluster health report requires admin privileges")
	}
	var args struct {
		Days int `json:"days"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 7
	}

	// Job overview
	jobOverview, _ := mgr.toolGetClusterHealthOverview(c, token, rawArgs)

	// Node & capacity
	capacity, _ := mgr.toolGetRealtimeCapacity(c, token, nil)

	// GPU models
	gpuModels, _ := mgr.toolListAvailableGPUModels(c, token, nil)

	// Failure stats
	failureStats, _ := mgr.toolGetFailureStatistics(c, token, rawArgs)

	return map[string]any{
		"scope":        "cluster",
		"lookbackDays": args.Days,
		"jobOverview":  jobOverview,
		"capacity":     capacity,
		"gpuModels":    gpuModels,
		"failureStats": failureStats,
	}, nil
}

// toolGetResourceRecommendation recommends resource configuration based on task requirements.
func (mgr *AgentMgr) toolGetResourceRecommendation(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		TaskDescription string `json:"task_description"`
		Framework       string `json:"framework"`
		GPURequired     *bool  `json:"gpu_required"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(args.TaskDescription) == "" {
		return nil, fmt.Errorf("task_description is required")
	}

	gpuRequired := true
	if args.GPURequired != nil {
		gpuRequired = *args.GPURequired
	}

	// Get available GPU models
	nodes, err := mgr.nodeClient.ListNodes(c.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	type gpuAvail struct {
		ResourceName string `json:"resource_name"`
		Model        string `json:"model"`
		Free         int64  `json:"free"`
	}
	availableGPUs := make([]gpuAvail, 0)
	for _, node := range nodes {
		for resName, allocatable := range node.Allocatable {
			name := string(resName)
			if !isGPUResourceName(name) {
				continue
			}
			used := int64(0)
			if u, ok := node.Used[resName]; ok {
				used = u.Value()
			}
			free := allocatable.Value() - used
			if free > 0 {
				availableGPUs = append(availableGPUs, gpuAvail{
					ResourceName: name,
					Model:        extractGPUModelFromResourceName(name),
					Free:         free,
				})
			}
		}
	}

	// Build recommendation
	rec := map[string]any{
		"task_description": args.TaskDescription,
	}

	desc := strings.ToLower(args.TaskDescription)
	framework := strings.ToLower(strings.TrimSpace(args.Framework))

	// Default resources
	cpuRec := "4"
	memRec := "16Gi"
	gpuCountRec := 0
	gpuModelRec := ""

	if gpuRequired && len(availableGPUs) > 0 {
		gpuCountRec = 1
		// Prefer first available
		sort.Slice(availableGPUs, func(i, j int) bool { return availableGPUs[i].Free > availableGPUs[j].Free })
		gpuModelRec = availableGPUs[0].Model

		// Adjust for large-scale tasks
		if strings.Contains(desc, "分布式") || strings.Contains(desc, "多卡") || strings.Contains(desc, "multi-gpu") ||
			strings.Contains(desc, "多gpu") || strings.Contains(desc, "distributed") {
			gpuCountRec = 4
			cpuRec = "16"
			memRec = "64Gi"
		} else if strings.Contains(desc, "llm") || strings.Contains(desc, "大模型") || strings.Contains(desc, "gpt") {
			gpuCountRec = 2
			cpuRec = "8"
			memRec = "32Gi"
		}
	}

	if framework == "pytorch" || strings.Contains(desc, "pytorch") || strings.Contains(desc, "torch") {
		rec["framework"] = "pytorch"
	} else if framework == "tensorflow" || strings.Contains(desc, "tensorflow") {
		rec["framework"] = "tensorflow"
	}

	rec["recommended"] = map[string]any{
		"cpu":       cpuRec,
		"memory":    memRec,
		"gpu_count": gpuCountRec,
		"gpu_model": gpuModelRec,
	}
	rec["available_gpus"] = availableGPUs
	rec["gpu_required"] = gpuRequired

	return rec, nil
}

func primaryNodeName(job *model.Job) string {
	if job == nil || job.Nodes.Data() == nil || len(job.Nodes.Data()) == 0 {
		return ""
	}
	return strings.TrimSpace(job.Nodes.Data()[0])
}

func minutesBetween(start, end time.Time) int {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return int(end.Sub(start).Minutes())
}

func formatDurationMinutes(totalMinutes int) string {
	if totalMinutes <= 0 {
		return ""
	}
	if totalMinutes >= 60 {
		return fmt.Sprintf("%dh%dm", totalMinutes/60, totalMinutes%60)
	}
	return fmt.Sprintf("%dm", totalMinutes)
}

func getIntValueFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// toolGetNodeDetail returns detailed info for a single cluster node (admin only).
func (mgr *AgentMgr) toolGetNodeDetail(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("node detail requires admin privileges")
	}
	var args struct {
		NodeName string `json:"node_name"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.NodeName == "" {
		return nil, fmt.Errorf("node_name is required")
	}

	nodeDetail, err := mgr.nodeClient.GetNode(c.Request.Context(), args.NodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	pods, podErr := mgr.nodeClient.AdminGetPodsForNode(c.Request.Context(), args.NodeName)
	if podErr != nil {
		return nil, fmt.Errorf("failed to get node workloads: %w", podErr)
	}

	buildResourcePressureAlert := func(resourceName string, used, allocatable int64) map[string]any {
		if allocatable <= 0 {
			return nil
		}
		usageRatio := float64(used) / float64(allocatable)
		if usageRatio < 0.9 {
			return nil
		}
		severity := "warning"
		if usageRatio >= 0.98 {
			severity = "critical"
		}
		return map[string]any{
			"severity": severity,
			"message":  fmt.Sprintf("%s 使用率 %.0f%%，节点资源接近上限", resourceName, usageRatio*100),
		}
	}

	runningJobs := make([]map[string]any, 0, len(pods))
	for _, pod := range pods {
		if len(pod.OwnerReference) == 0 || pod.Namespace != pkgconfig.GetConfig().Namespaces.Job {
			continue
		}
		owner := pod.OwnerReference[0]
		if owner.Kind != "Job" {
			continue
		}
		runningJobs = append(runningJobs, map[string]any{
			"job_name":          owner.Name,
			"pod_name":          pod.Name,
			"namespace":         pod.Namespace,
			"user":              pod.UserName,
			"account":           pod.AccountName,
			"status":            pod.Status,
			"created_at":        pod.CreateTime,
			"request_resources": pod.RequestResources,
			"limit_resources":   pod.Resources,
			"locked":            pod.Locked,
			"permanent_locked":  pod.PermanentLocked,
			"locked_timestamp":  pod.LockedTimestamp,
		})
	}

	alerts := make([]map[string]any, 0, 4)
	if nodeDetail.Status != v1.NodeReady {
		alerts = append(alerts, map[string]any{
			"severity": "warning",
			"message":  fmt.Sprintf("节点状态为 %s，需要关注调度与资源可用性", nodeDetail.Status),
		})
	}
	if strings.TrimSpace(nodeDetail.Taint) != "" {
		alerts = append(alerts, map[string]any{
			"severity": "info",
			"message":  fmt.Sprintf("节点存在 taint: %s", nodeDetail.Taint),
		})
	}
	for resourceName, allocatable := range nodeDetail.Allocatable {
		name := string(resourceName)
		if !isGPUResourceName(name) && name != string(v1.ResourceCPU) && name != string(v1.ResourceMemory) {
			continue
		}
		usedQuantity, ok := nodeDetail.Used[resourceName]
		if !ok {
			continue
		}
		if alert := buildResourcePressureAlert(name, usedQuantity.Value(), allocatable.Value()); alert != nil {
			alerts = append(alerts, alert)
		}
	}

	return map[string]any{
		"name":           nodeDetail.Name,
		"role":           nodeDetail.Role,
		"status":         nodeDetail.Status,
		"address":        nodeDetail.Address,
		"os":             nodeDetail.Os,
		"osVersion":      nodeDetail.OsVersion,
		"arch":           nodeDetail.Arch,
		"kernel":         nodeDetail.KernelVersion,
		"kubelet":        nodeDetail.KubeletVersion,
		"runtime":        nodeDetail.ContainerRuntimeVersion,
		"gpuDriver":      nodeDetail.GPUDriver,
		"gpuCount":       nodeDetail.GPUCount,
		"gpuArch":        nodeDetail.GPUArch,
		"gpuMemory":      nodeDetail.GPUMemory,
		"capacity":       nodeDetail.Capacity,
		"allocatable":    nodeDetail.Allocatable,
		"used":           nodeDetail.Used,
		"alerts":         alerts,
		"running_jobs":   runningJobs,
		"workload_count": len(runningJobs),
	}, nil
}

// toolGetAdminOpsReport aggregates a cluster-level AIOps report for admins.
func (mgr *AgentMgr) toolGetAdminOpsReport(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("admin ops report requires admin privileges")
	}

	var args struct {
		Days          int `json:"days"`
		SuccessLimit  int `json:"success_limit"`
		FailureLimit  int `json:"failure_limit"`
		GPUThreshold  int `json:"gpu_threshold"`
		IdleHours     int `json:"idle_hours"`
		LookbackHours int `json:"lookback_hours"`
		RunningLimit  int `json:"running_limit"`
		NodeLimit     int `json:"node_limit"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 7
	}
	if args.SuccessLimit <= 0 || args.SuccessLimit > 10 {
		args.SuccessLimit = 5
	}
	if args.FailureLimit <= 0 || args.FailureLimit > 10 {
		args.FailureLimit = 5
	}
	if args.GPUThreshold <= 0 {
		args.GPUThreshold = 5
	}
	if args.IdleHours <= 0 {
		args.IdleHours = 24
	}
	if args.LookbackHours <= 0 {
		args.LookbackHours = 1
	}
	if args.RunningLimit <= 0 || args.RunningLimit > 50 {
		args.RunningLimit = 20
	}
	if args.NodeLimit <= 0 || args.NodeLimit > 50 {
		args.NodeLimit = 10
	}

	j := query.Job
	lookback := time.Now().AddDate(0, 0, -args.Days)
	windowStart := time.Now().Add(-time.Duration(args.LookbackHours) * time.Hour)

	completedJobs, err := j.WithContext(c).
		Preload(j.User).
		Where(j.Status.Eq(string(batch.Completed))).
		Where(j.CompletedTimestamp.Gte(lookback)).
		Order(j.CompletedTimestamp.Desc()).
		Limit(args.SuccessLimit).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query completed jobs: %w", err)
	}

	failedJobs, err := j.WithContext(c).
		Preload(j.User).
		Where(j.Status.Eq(string(batch.Failed))).
		Where(j.CompletedTimestamp.Gte(lookback)).
		Order(j.CompletedTimestamp.Desc()).
		Limit(args.FailureLimit).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query failed jobs: %w", err)
	}

	overviewRaw, overviewErr := mgr.toolGetClusterHealthOverview(c, token, json.RawMessage(fmt.Sprintf(`{"days":%d}`, args.Days)))
	if overviewErr != nil {
		return nil, overviewErr
	}
	failureStatsRaw, failureStatsErr := mgr.toolGetFailureStatistics(c, token, json.RawMessage(fmt.Sprintf(`{"days":%d,"limit":10}`, args.Days)))
	if failureStatsErr != nil {
		return nil, failureStatsErr
	}
	idleRaw, idleErr := mgr.toolDetectIdleJobs(c, token, json.RawMessage(fmt.Sprintf(`{"gpu_threshold":%d,"hours":%d}`, args.GPUThreshold, args.IdleHours)))
	if idleErr != nil {
		return nil, idleErr
	}

	overview, _ := overviewRaw.(map[string]any)
	failureStats, _ := failureStatsRaw.(map[string]any)
	idleSummary, _ := idleRaw.(map[string]any)
	nodeOverviewRaw, nodeOverviewErr := mgr.toolListClusterNodes(c, token)
	if nodeOverviewErr != nil {
		return nil, nodeOverviewErr
	}
	nodeOverview, _ := nodeOverviewRaw.(map[string]any)

	successSamples := make([]map[string]any, 0, len(completedJobs))
	for _, job := range completedJobs {
		profile := job.ProfileData
		gpuUtil := 0.0
		cpuUsage := 0.0
		memUsage := 0.0
		if profile != nil && profile.Data() != nil {
			if profile.Data().GPUUtilAvg != nil {
				gpuUtil = float64(*profile.Data().GPUUtilAvg)
			}
			if profile.Data().CPUUsageAvg != nil {
				cpuUsage = float64(*profile.Data().CPUUsageAvg)
			}
			if profile.Data().CPUMemAvg != nil {
				memUsage = float64(*profile.Data().CPUMemAvg)
			}
		}
		gpuRequested := extractRequestedGPUCount(job.Resources.Data())
		runningMinutes := minutesBetween(job.RunningTimestamp, job.CompletedTimestamp)
		queueWaitMinutes := minutesBetween(job.CreationTimestamp, job.RunningTimestamp)
		successSamples = append(successSamples, map[string]any{
			"job_name":            job.JobName,
			"name":                job.Name,
			"user":                jobDisplayUser(job),
			"scheduled_node":      primaryNodeName(job),
			"requested_resources": job.Resources.Data(),
			"actual_usage": map[string]any{
				"gpu_util_avg":  gpuUtil,
				"cpu_usage_avg": cpuUsage,
				"mem_usage_avg": memUsage,
			},
			"queue_wait_minutes": queueWaitMinutes,
			"running_minutes":    runningMinutes,
			"gpu_requested":      gpuRequested,
			"gpu_actual_used":    estimateActualGPUUsage(gpuUtil, gpuRequested),
		})
	}

	failureSamples := make([]map[string]any, 0, len(failedJobs))
	for _, job := range failedJobs {
		exitCode := int32(0)
		exitReason := ""
		if terminated := getFirstTerminatedState(job); terminated != nil {
			exitCode = terminated.ExitCode
			exitReason = terminated.Reason
		}
		failureSamples = append(failureSamples, map[string]any{
			"job_name":            job.JobName,
			"name":                job.Name,
			"user":                jobDisplayUser(job),
			"failure_category":    getFailureCategory(job),
			"scheduled_node":      primaryNodeName(job),
			"requested_resources": job.Resources.Data(),
			"queue_wait_minutes":  minutesBetween(job.CreationTimestamp, job.RunningTimestamp),
			"running_minutes":     minutesBetween(job.RunningTimestamp, job.CompletedTimestamp),
			"exit_code":           exitCode,
			"exit_reason":         exitReason,
			"gpu_requested":       extractRequestedGPUCount(job.Resources.Data()),
		})
	}

	idleJobsAny, _ := idleSummary["idle_jobs"].([]idleEntry)
	idleActions := make([]map[string]any, 0, len(idleJobsAny))
	estimatedWasteHours := 0.0
	for _, item := range idleJobsAny {
		estimatedWasteHours += float64(item.GPUCount) * item.IdleHours
		idleActions = append(idleActions, map[string]any{
			"job_name":      item.JobName,
			"user":          item.Username,
			"gpu_util":      fmt.Sprintf("%.1f%%", item.GPUUtilization),
			"duration":      formatDurationMinutes(int(item.IdleHours * 60)),
			"gpu_requested": item.GPUCount,
			"gpu_actual":    item.ActualGPUUsed,
		})
	}

	failureActionItems := make([]map[string]any, 0, len(failureSamples))
	for _, item := range failureSamples {
		failureActionItems = append(failureActionItems, map[string]any{
			"job_name":      item["job_name"],
			"user":          item["user"],
			"gpu_util":      "failed",
			"duration":      formatDurationMinutes(item["running_minutes"].(int)),
			"gpu_requested": item["gpu_requested"],
			"gpu_actual":    0,
		})
	}

	successActionItems := make([]map[string]any, 0, len(successSamples))
	for _, item := range successSamples {
		actualUsage, _ := item["actual_usage"].(map[string]any)
		gpuUtil, _ := actualUsage["gpu_util_avg"].(float64)
		successActionItems = append(successActionItems, map[string]any{
			"job_name":      item["job_name"],
			"user":          item["user"],
			"gpu_util":      fmt.Sprintf("%.1f%%", gpuUtil),
			"duration":      formatDurationMinutes(item["running_minutes"].(int)),
			"gpu_requested": item["gpu_requested"],
			"gpu_actual":    item["gpu_actual_used"],
		})
	}

	statusCount, _ := overview["statusCount"].(map[string]int)
	if statusCount == nil {
		statusCount = map[string]int{}
	}
	totalJobs, _ := overview["totalJobs"].(int)
	successRate := 0.0
	failureRate := 0.0
	if totalJobs > 0 {
		successRate = float64(statusCount[string(batch.Completed)]) / float64(totalJobs)
		failureRate = float64(statusCount[string(batch.Failed)]) / float64(totalJobs)
	}

	activeRunningJobs, err := j.WithContext(c).
		Preload(j.User).
		Where(j.Status.Eq(string(batch.Running))).
		Order(j.RunningTimestamp.Desc()).
		Limit(args.RunningLimit).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query active running jobs: %w", err)
	}
	recentFinishedJobs, err := j.WithContext(c).
		Preload(j.User).
		Where(j.Status.In(string(batch.Completed), string(batch.Failed))).
		Where(j.CompletedTimestamp.Gte(windowStart)).
		Order(j.CompletedTimestamp.Desc()).
		Limit(args.RunningLimit).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query recent running window jobs: %w", err)
	}

	recentRunningJobs := make([]map[string]any, 0, args.RunningLimit)
	seenRecentJobs := make(map[string]struct{}, args.RunningLimit)
	appendRecentRunningJob := func(job *model.Job, observedState string) {
		if job == nil || strings.TrimSpace(job.JobName) == "" {
			return
		}
		if _, exists := seenRecentJobs[job.JobName]; exists {
			return
		}
		if job.RunningTimestamp.IsZero() {
			return
		}
		effectiveEnd := job.CompletedTimestamp
		if effectiveEnd.IsZero() || effectiveEnd.After(time.Now()) {
			effectiveEnd = time.Now()
		}
		if effectiveEnd.Before(windowStart) {
			return
		}
		overlapStart := job.RunningTimestamp
		if overlapStart.Before(windowStart) {
			overlapStart = windowStart
		}
		overlapMinutes := minutesBetween(overlapStart, effectiveEnd)
		runningMinutes := minutesBetween(job.RunningTimestamp, effectiveEnd)
		if overlapMinutes <= 0 && job.Status != batch.Running {
			return
		}
		seenRecentJobs[job.JobName] = struct{}{}
		recentRunningJobs = append(recentRunningJobs, map[string]any{
			"job_name":               job.JobName,
			"name":                   job.Name,
			"user":                   jobDisplayUser(job),
			"status":                 job.Status,
			"scheduled_node":         primaryNodeName(job),
			"requested_resources":    job.Resources.Data(),
			"gpu_requested":          extractRequestedGPUCount(job.Resources.Data()),
			"queue_wait_minutes":     minutesBetween(job.CreationTimestamp, job.RunningTimestamp),
			"running_minutes":        runningMinutes,
			"window_overlap_minutes": overlapMinutes,
			"observed_state":         observedState,
			"running_since":          job.RunningTimestamp,
			"completed_at":           job.CompletedTimestamp,
		})
	}

	for _, job := range activeRunningJobs {
		appendRecentRunningJob(job, "active")
		if len(recentRunningJobs) >= args.RunningLimit {
			break
		}
	}
	if len(recentRunningJobs) < args.RunningLimit {
		for _, job := range recentFinishedJobs {
			appendRecentRunningJob(job, "recently_finished")
			if len(recentRunningJobs) >= args.RunningLimit {
				break
			}
		}
	}

	nodeItems, _ := nodeOverview["nodes"].([]map[string]any)
	sort.Slice(nodeItems, func(i, j int) bool {
		leftStatus := fmt.Sprint(nodeItems[i]["status"])
		rightStatus := fmt.Sprint(nodeItems[j]["status"])
		if leftStatus != rightStatus {
			if leftStatus != string(v1.NodeReady) {
				return true
			}
			if rightStatus != string(v1.NodeReady) {
				return false
			}
		}
		return getIntValueFromAny(nodeItems[i]["workloads"]) > getIntValueFromAny(nodeItems[j]["workloads"])
	})
	sampledNodes := make([]map[string]any, 0, min(args.NodeLimit, len(nodeItems)))
	unhealthyNodes := make([]map[string]any, 0, args.NodeLimit)
	for _, node := range nodeItems {
		nodeSummary := map[string]any{
			"name":      node["name"],
			"role":      node["role"],
			"status":    node["status"],
			"workloads": node["workloads"],
			"vendor":    node["vendor"],
			"address":   node["address"],
		}
		if len(sampledNodes) < args.NodeLimit {
			sampledNodes = append(sampledNodes, nodeSummary)
		}
		if fmt.Sprint(node["status"]) != string(v1.NodeReady) && len(unhealthyNodes) < args.NodeLimit {
			unhealthyNodes = append(unhealthyNodes, nodeSummary)
		}
	}

	return map[string]any{
		"report_type":    "admin_ops_report",
		"generated_at":   time.Now(),
		"lookback_days":  args.Days,
		"lookback_hours": args.LookbackHours,
		"overview": map[string]any{
			"total_jobs":   totalJobs,
			"status_count": statusCount,
			"success_jobs": statusCount[string(batch.Completed)],
			"failed_jobs":  statusCount[string(batch.Failed)],
			"running_jobs": statusCount[string(batch.Running)],
			"pending_jobs": statusCount[string(batch.Pending)],
			"success_rate": successRate,
			"failure_rate": failureRate,
		},
		"failure_categories": failureStats["categories"],
		"idle_summary": map[string]any{
			"idle_job_count":            len(idleJobsAny),
			"estimated_gpu_waste_hours": estimatedWasteHours,
			"top_jobs":                  idleActions,
		},
		"recent_running_summary": map[string]any{
			"window_hours": args.LookbackHours,
			"job_count":    len(recentRunningJobs),
		},
		"recent_running_jobs": recentRunningJobs,
		"node_summary": map[string]any{
			"total_nodes":      nodeOverview["count"],
			"status_count":     nodeOverview["statusCount"],
			"sampled_nodes":    sampledNodes,
			"unhealthy_nodes":  unhealthyNodes,
			"requested_sample": args.NodeLimit,
		},
		"successful_jobs": successSamples,
		"failed_jobs":     failureSamples,
		"recommended_actions": []map[string]any{
			{
				"action":   "关注失败作业热点",
				"severity": "warning",
				"count":    len(failureActionItems),
				"items":    failureActionItems,
			},
			{
				"action":   "复盘成功作业资源差异",
				"severity": "info",
				"count":    len(successActionItems),
				"items":    successActionItems,
			},
			{
				"action":   "处理低利用率作业",
				"severity": "warning",
				"count":    len(idleActions),
				"items":    idleActions,
			},
		},
	}, nil
}

// ─── AIOps audit read-only tools ────────────────────────────────────────────

// toolGetLatestAuditReport returns the most recent audit report of a given type.
func (mgr *AgentMgr) toolGetLatestAuditReport(_ *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("audit reports require admin privileges")
	}
	var args struct {
		ReportType string `json:"report_type"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if strings.TrimSpace(args.ReportType) == "" {
		args.ReportType = "gpu_audit"
	}

	db := query.GetDB()
	var report struct {
		ID            string          `json:"id" gorm:"column:id"`
		ReportType    string          `json:"report_type" gorm:"column:report_type"`
		Status        string          `json:"status" gorm:"column:status"`
		TriggerSource *string         `json:"trigger_source" gorm:"column:trigger_source"`
		Summary       json.RawMessage `json:"summary" gorm:"column:summary"`
		CreatedAt     time.Time       `json:"created_at" gorm:"column:created_at"`
		CompletedAt   *time.Time      `json:"completed_at" gorm:"column:completed_at"`
	}
	result := db.Raw(`
		SELECT id, report_type, status, trigger_source, summary, created_at, completed_at
		FROM ops_audit_reports WHERE report_type = ? ORDER BY created_at DESC LIMIT 1
	`, args.ReportType).Scan(&report)
	if result.Error != nil || result.RowsAffected == 0 {
		return map[string]any{"message": "暂无审计报告"}, nil
	}
	return report, nil
}

// toolListAuditItems lists audit items with optional filters.
func (mgr *AgentMgr) toolListAuditItems(_ *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("audit items require admin privileges")
	}
	var args struct {
		ActionType string `json:"action_type"`
		Severity   string `json:"severity"`
		Handled    *bool  `json:"handled"`
		ReportID   string `json:"report_id"`
		Limit      int    `json:"limit"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Limit <= 0 || args.Limit > 100 {
		args.Limit = 30
	}

	db := query.GetDB()
	q := db.Table("ops_audit_items")
	if strings.TrimSpace(args.ActionType) != "" {
		q = q.Where("action_type = ?", strings.TrimSpace(args.ActionType))
	}
	if strings.TrimSpace(args.Severity) != "" {
		q = q.Where("severity = ?", strings.TrimSpace(args.Severity))
	}
	if args.Handled != nil {
		q = q.Where("handled = ?", *args.Handled)
	}
	if strings.TrimSpace(args.ReportID) != "" {
		q = q.Where("report_id = ?", strings.TrimSpace(args.ReportID))
	}
	q = q.Order("created_at DESC").Limit(args.Limit)

	var items []map[string]any
	if err := q.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to query audit items: %w", err)
	}

	return map[string]any{
		"items": items,
		"count": len(items),
	}, nil
}

// toolSaveAuditReport creates a new audit report with its items.
func (mgr *AgentMgr) toolSaveAuditReport(_ *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("saving audit reports requires admin privileges")
	}
	var args struct {
		ReportType    string           `json:"report_type"`
		TriggerSource string           `json:"trigger_source"`
		Summary       json.RawMessage  `json:"summary"`
		Items         []map[string]any `json:"items"`
		// Extended fields for ops reports
		ReportJSON  json.RawMessage `json:"report_json"`
		PeriodStart *string         `json:"period_start"`
		PeriodEnd   *string         `json:"period_end"`
		JobTotal    int             `json:"job_total"`
		JobSuccess  int             `json:"job_success"`
		JobFailed   int             `json:"job_failed"`
		JobPending  int             `json:"job_pending"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(args.ReportType) == "" {
		args.ReportType = "gpu_audit"
	}
	if strings.TrimSpace(args.TriggerSource) == "" {
		args.TriggerSource = "agent"
	}
	if args.Summary == nil {
		args.Summary = json.RawMessage("{}")
	}

	db := query.GetDB()

	// Insert report — try with extended fields first, fall back to base columns
	var reportJSON *string
	if len(args.ReportJSON) > 0 && string(args.ReportJSON) != "null" {
		s := string(args.ReportJSON)
		reportJSON = &s
	}

	var reportID string
	insertResult := db.Raw(`
		INSERT INTO ops_audit_reports
			(report_type, status, trigger_source, summary, report_json,
			 period_start, period_end, job_total, job_success, job_failed, job_pending,
			 completed_at)
		VALUES (?, 'completed', ?, ?, ?,
			CASE WHEN ? = '' THEN NULL ELSE ?::timestamptz END,
			CASE WHEN ? = '' THEN NULL ELSE ?::timestamptz END,
			?, ?, ?, ?, NOW())
		RETURNING id
	`,
		args.ReportType, args.TriggerSource, string(args.Summary), reportJSON,
		ptrToStr(args.PeriodStart), ptrToStr(args.PeriodStart),
		ptrToStr(args.PeriodEnd), ptrToStr(args.PeriodEnd),
		args.JobTotal, args.JobSuccess, args.JobFailed, args.JobPending,
	).Scan(&reportID)
	if insertResult.Error != nil {
		// Fallback: insert with base columns only (migration not applied yet)
		insertResult = db.Raw(`
			INSERT INTO ops_audit_reports (report_type, status, trigger_source, summary, completed_at)
			VALUES (?, 'completed', ?, ?, NOW())
			RETURNING id
		`, args.ReportType, args.TriggerSource, string(args.Summary)).Scan(&reportID)
		if insertResult.Error != nil {
			return nil, fmt.Errorf("failed to create audit report: %w", insertResult.Error)
		}
	}

	// Insert items with extended fields
	itemCount := 0
	for _, item := range args.Items {
		jobName, _ := item["job_name"].(string)
		if strings.TrimSpace(jobName) == "" {
			continue
		}
		actionType, _ := item["action_type"].(string)
		if strings.TrimSpace(actionType) == "" {
			actionType = "stop"
		}
		severity, _ := item["severity"].(string)
		if strings.TrimSpace(severity) == "" {
			severity = "warning"
		}
		analysisDetail, _ := json.Marshal(item["analysis_detail"])
		if string(analysisDetail) == "null" {
			analysisDetail = []byte("{}")
		}
		resourceRequested, _ := json.Marshal(item["resource_requested"])
		if string(resourceRequested) == "null" {
			resourceRequested = []byte("{}")
		}
		resourceActual, _ := json.Marshal(item["resource_actual"])
		if string(resourceActual) == "null" {
			resourceActual = []byte("{}")
		}

		err := db.Exec(`
			INSERT INTO ops_audit_items
				(report_id, job_name, user_id, account_id, username, action_type, severity,
				 gpu_utilization, gpu_requested, gpu_actual_used, analysis_detail,
				 category, job_type, owner, namespace, duration_seconds,
				 resource_requested, resource_actual, exit_code, failure_reason)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			reportID,
			jobName,
			getMapStringValue(item, "user_id"),
			getMapStringValue(item, "account_id"),
			getMapStringValue(item, "username"),
			actionType,
			severity,
			getMapFloatValue(item, "gpu_utilization"),
			getMapIntValue(item, "gpu_requested"),
			getMapIntValue(item, "gpu_actual_used"),
			string(analysisDetail),
			getMapStringValue(item, "category"),
			getMapStringValue(item, "job_type"),
			getMapStringValue(item, "owner"),
			getMapStringValue(item, "namespace"),
			getMapIntValue(item, "duration_seconds"),
			string(resourceRequested),
			string(resourceActual),
			getMapIntValue(item, "exit_code"),
			getMapStringValue(item, "failure_reason"),
		).Error
		if err != nil {
			// Fallback: insert with base columns only
			err = db.Exec(`
				INSERT INTO ops_audit_items
					(report_id, job_name, user_id, account_id, username, action_type, severity,
					 gpu_utilization, gpu_requested, gpu_actual_used, analysis_detail)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
				reportID, jobName,
				getMapStringValue(item, "user_id"),
				getMapStringValue(item, "account_id"),
				getMapStringValue(item, "username"),
				actionType, severity,
				getMapFloatValue(item, "gpu_utilization"),
				getMapIntValue(item, "gpu_requested"),
				getMapIntValue(item, "gpu_actual_used"),
				string(analysisDetail),
			).Error
		}
		if err == nil {
			itemCount++
		}
	}

	return map[string]any{
		"report_id":  reportID,
		"item_count": itemCount,
		"status":     "completed",
	}, nil
}

func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getMapStringValue(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func getMapFloatValue(m map[string]any, key string) *float64 {
	switch v := m[key].(type) {
	case float64:
		return &v
	case float32:
		f := float64(v)
		return &f
	}
	return nil
}

func getMapIntValue(m map[string]any, key string) *int {
	switch v := m[key].(type) {
	case float64:
		i := int(v)
		return &i
	case int:
		return &v
	}
	return nil
}
