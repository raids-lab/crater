package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	basehandler "github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/util"
)

const (
	agentDefaultJobLogTailLines        = 200
	agentMaxJobLogTailLines            = 2000
	agentDefaultDiagnosticTailLines    = 120
	agentMaxDiagnosticTailLines        = 1000
	agentDefaultSimilarFailureLimit    = 5
	agentMaxSimilarFailureLimit        = 20
	agentDefaultSimilarFailureDays     = 30
	agentMaxSimilarFailureDays         = 180
	agentSimilarFailureQueryMultiplier = 4
	agentDefaultListImagesLimit        = 30
	agentMaxListImagesLimit            = 100
	agentDefaultListJobsLimit          = 20
	agentMaxListJobsLimit              = 100
)

func agentReadonlyErrorf(format string, args ...any) error {
	return agentDispatchErrorf(format, args...)
}

func (mgr *AgentMgr) requireJobReader() (basehandler.JobInsightReader, error) {
	if mgr.jobReader == nil {
		return nil, agentReadonlyErrorf("job reader is not available")
	}
	return mgr.jobReader, nil
}

func (mgr *AgentMgr) findScopedJob(ctx *gin.Context, token util.JWTMessage, jobName string) (*model.Job, error) {
	reader, err := mgr.requireJobReader()
	if err != nil {
		return nil, err
	}
	return reader.FindScopedJob(ctx.Request.Context(), token, strings.TrimSpace(jobName))
}

func getJobNameArg(rawArgs json.RawMessage) (string, error) {
	args := parseToolArgsMap(rawArgs)
	jobName := getToolArgString(args, "job_name", "")
	if jobName == "" {
		jobName = getToolArgString(args, "jobName", "")
	}
	if jobName == "" {
		return "", agentReadonlyErrorf("job_name is required")
	}
	return jobName, nil
}

func (mgr *AgentMgr) toolGetJobDetail(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	jobName, err := getJobNameArg(rawArgs)
	if err != nil {
		return nil, err
	}
	reader, err := mgr.requireJobReader()
	if err != nil {
		return nil, err
	}
	job, err := reader.FindScopedJob(c.Request.Context(), token, jobName)
	if err != nil {
		return nil, err
	}
	return reader.BuildJobDetail(job), nil
}

func (mgr *AgentMgr) toolGetJobEvents(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	jobName, err := getJobNameArg(rawArgs)
	if err != nil {
		return nil, err
	}
	reader, err := mgr.requireJobReader()
	if err != nil {
		return nil, err
	}
	return reader.GetJobEvents(c.Request.Context(), token, jobName)
}

func (mgr *AgentMgr) toolGetJobLogs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	jobName, err := getJobNameArg(rawArgs)
	if err != nil {
		return nil, err
	}
	tailLines := int64(getToolArgInt(args, "tail", agentDefaultJobLogTailLines))
	if tailLines <= 0 || tailLines > agentMaxJobLogTailLines {
		tailLines = agentDefaultJobLogTailLines
	}
	reader, err := mgr.requireJobReader()
	if err != nil {
		return nil, err
	}
	return reader.GetJobLog(
		c.Request.Context(),
		token,
		jobName,
		tailLines,
		getToolArgString(args, "keyword", ""),
	)
}

func (mgr *AgentMgr) toolDiagnoseJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	jobName, err := getJobNameArg(rawArgs)
	if err != nil {
		return nil, err
	}
	job, err := mgr.findScopedJob(c, token, jobName)
	if err != nil {
		return nil, err
	}
	return basehandler.PerformDiagnosis(job), nil
}

func (mgr *AgentMgr) toolGetDiagnosticContext(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	jobName, err := getJobNameArg(rawArgs)
	if err != nil {
		return nil, err
	}
	reader, err := mgr.requireJobReader()
	if err != nil {
		return nil, err
	}
	tailLines := int64(getToolArgInt(args, "tail", agentDefaultDiagnosticTailLines))
	if tailLines <= 0 || tailLines > agentMaxDiagnosticTailLines {
		tailLines = agentDefaultDiagnosticTailLines
	}
	return reader.GetDiagnosticContext(
		c.Request.Context(),
		token,
		jobName,
		getToolArgBool(args, "include_log", true),
		tailLines,
	)
}

func (mgr *AgentMgr) toolQueryJobMetrics(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	jobName, err := getJobNameArg(rawArgs)
	if err != nil {
		return nil, err
	}
	job, err := mgr.findScopedJob(c, token, jobName)
	if err != nil {
		return nil, err
	}
	if job.ProfileData == nil {
		return map[string]any{"job_name": job.JobName, "metrics": map[string]any{}}, nil
	}
	return map[string]any{"job_name": job.JobName, "metrics": job.ProfileData.Data()}, nil
}

func (mgr *AgentMgr) toolSearchSimilarFailures(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	jobName, err := getJobNameArg(rawArgs)
	if err != nil {
		return nil, err
	}
	target, err := mgr.findScopedJob(c, token, jobName)
	if err != nil {
		return nil, err
	}
	category := basehandler.CategorizeFailure(target).TypeName
	limit := getToolArgInt(args, "limit", agentDefaultSimilarFailureLimit)
	if limit <= 0 || limit > agentMaxSimilarFailureLimit {
		limit = agentDefaultSimilarFailureLimit
	}
	days := getToolArgInt(args, "days", agentDefaultSimilarFailureDays)
	if days <= 0 || days > agentMaxSimilarFailureDays {
		days = agentDefaultSimilarFailureDays
	}

	j := query.Job
	dbQuery := j.WithContext(c.Request.Context()).
		Where(j.Status.Eq(string(v1alpha1.Failed))).
		Where(j.CreationTimestamp.Gte(time.Now().AddDate(0, 0, -days)))
	if token.RolePlatform != model.RoleAdmin {
		dbQuery = dbQuery.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	jobs, err := dbQuery.Order(j.CreationTimestamp.Desc()).Limit(limit * agentSimilarFailureQueryMultiplier).Find()
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, limit)
	for _, job := range jobs {
		if job.JobName == target.JobName {
			continue
		}
		if basehandler.CategorizeFailure(job).TypeName != category {
			continue
		}
		items = append(items, map[string]any{
			"job_name":  job.JobName,
			"name":      job.Name,
			"status":    job.Status,
			"createdAt": job.CreationTimestamp,
			"category":  category,
		})
		if len(items) >= limit {
			break
		}
	}
	return map[string]any{"category": category, "items": items}, nil
}

func (mgr *AgentMgr) toolAnalyzeQueueStatus(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	jobName, err := getJobNameArg(rawArgs)
	if err != nil {
		return nil, err
	}
	job, err := mgr.findScopedJob(c, token, jobName)
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"job_name": job.JobName,
		"status":   job.Status,
		"pending":  job.Status == v1alpha1.Pending,
	}
	if job.Events != nil {
		result["events"] = job.Events.Data()
	}
	if job.ScheduleData != nil {
		result["schedule_data"] = job.ScheduleData.Data()
	}
	return result, nil
}

func (mgr *AgentMgr) toolGetRealtimeCapacity(c *gin.Context, _ util.JWTMessage, _ json.RawMessage) (any, error) {
	if mgr.nodeClient == nil {
		return map[string]any{"nodes": []any{}}, nil
	}
	nodes, err := mgr.nodeClient.ListNodes(c.Request.Context())
	if err != nil {
		return nil, err
	}
	return map[string]any{"nodes": nodes}, nil
}

func (mgr *AgentMgr) toolListAvailableImages(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if mgr.imageReader == nil {
		return nil, agentReadonlyErrorf("image reader is not available")
	}
	args := parseToolArgsMap(rawArgs)
	limit := getToolArgInt(args, "limit", agentDefaultListImagesLimit)
	if limit <= 0 || limit > agentMaxListImagesLimit {
		limit = agentDefaultListImagesLimit
	}
	records, err := mgr.imageReader.ListAccessibleImages(c.Request.Context(), token)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, limit)
	for _, record := range records {
		if record.Image == nil {
			continue
		}
		imageName := record.Image.ImageLink
		if record.Image.ImagePackName != nil && strings.TrimSpace(*record.Image.ImagePackName) != "" {
			imageName = *record.Image.ImagePackName
		}
		items = append(items, map[string]any{
			"id":           record.Image.ID,
			"name":         imageName,
			"image_link":   record.Image.ImageLink,
			"description":  record.Image.Description,
			"share_status": record.ShareStatus,
		})
		if len(items) >= limit {
			break
		}
	}
	return map[string]any{"images": items, "count": len(items)}, nil
}

func (mgr *AgentMgr) toolListAvailableGPUModels(_ *gin.Context, _ util.JWTMessage, _ json.RawMessage) (any, error) {
	return map[string]any{"models": []string{"V100", "A100", "H100", "L40S", "RTX4090"}}, nil
}

func (mgr *AgentMgr) toolCheckQuota(_ *gin.Context, token util.JWTMessage, _ json.RawMessage) (any, error) {
	return map[string]any{
		"account_id": token.AccountID,
		"user_id":    token.UserID,
		"message":    "quota detail is available from the platform quota APIs",
	}, nil
}

func (mgr *AgentMgr) toolListUserJobs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	limit := getToolArgInt(args, "limit", agentDefaultListJobsLimit)
	if limit <= 0 || limit > agentMaxListJobsLimit {
		limit = agentDefaultListJobsLimit
	}
	j := query.Job
	dbQuery := j.WithContext(c.Request.Context()).Order(j.CreationTimestamp.Desc()).Limit(limit)
	if token.RolePlatform != model.RoleAdmin {
		dbQuery = dbQuery.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	jobs, err := dbQuery.Find()
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, map[string]any{
			"job_name":  job.JobName,
			"name":      job.Name,
			"type":      job.JobType,
			"status":    job.Status,
			"createdAt": job.CreationTimestamp,
		})
	}
	return map[string]any{"jobs": items, "count": len(items)}, nil
}

func (mgr *AgentMgr) toolGetJobTemplates(_ *gin.Context, _ util.JWTMessage, _ json.RawMessage) (any, error) {
	return map[string]any{
		"templates": []map[string]any{
			{"type": "jupyter", "tool": agentToolCreateJupyter},
			{"type": "webide", "tool": agentToolCreateWebIDE},
			{"type": "custom", "tool": agentToolCreateCustom},
			{"type": "pytorch", "tool": agentToolCreatePytorch},
			{"type": "tensorflow", "tool": agentToolCreateTensorflow},
		},
	}, nil
}

func (mgr *AgentMgr) toolGetResourceRecommendation(_ *gin.Context, _ util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	taskType := strings.ToLower(getToolArgString(args, "task_type", ""))
	cpu := "4"
	memory := "16Gi"
	gpuCount := 0
	if strings.Contains(taskType, "train") || strings.Contains(taskType, "训练") {
		cpu = "8"
		memory = "32Gi"
		gpuCount = 1
	}
	return map[string]any{
		"cpu":       cpu,
		"memory":    memory,
		"gpu_count": gpuCount,
		"reason":    fmt.Sprintf("basic recommendation for %q", taskType),
	}, nil
}
