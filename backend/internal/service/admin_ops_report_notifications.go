package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/alert"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/patrol"
)

type opsAuditReportRow struct {
	ID         string          `gorm:"column:id"`
	Summary    json.RawMessage `gorm:"column:summary"`
	ReportJSON json.RawMessage `gorm:"column:report_json"`
	JobTotal   int             `gorm:"column:job_total"`
	JobFailed  int             `gorm:"column:job_failed"`
	JobPending int             `gorm:"column:job_pending"`
}

type opsAuditItemRow struct {
	ID            string `gorm:"column:id"`
	JobName       string `gorm:"column:job_name"`
	Username      string `gorm:"column:username"`
	ActionType    string `gorm:"column:action_type"`
	Severity      string `gorm:"column:severity"`
	Category      string `gorm:"column:category"`
	FailureReason string `gorm:"column:failure_reason"`
}

type opsReportNotificationSignal struct {
	ShouldNotifyAdmins  bool
	ShouldNotifyOwners  bool
	Reasons             []string
	FailedJobs          int
	TotalJobs           int
	FailureRatePercent  int
	UnhealthyNodes      int
	NetworkAlerts       int
	HighRiskNetworkJobs int
	ExecutiveSummary    string
	Recommendations     []string
}

func (s *AdminOpsReportService) notifyAdminOpsReport(
	ctx context.Context,
	req patrol.TriggerAdminOpsReportRequest,
	pipelineResult map[string]any,
) map[string]any {
	policy := normalizeServiceOpsNotificationPolicy(req.Notification)
	if !policy.Enabled || req.DryRun {
		return nil
	}

	reportID := strings.TrimSpace(fmt.Sprint(pipelineResult["report_id"]))
	if reportID == "" || reportID == "<nil>" {
		return map[string]any{"enabled": true, "status": "skipped", "reason": "report_id_missing"}
	}

	report, err := loadOpsAuditReport(ctx, reportID)
	if err != nil {
		klog.Warningf("admin ops notification: failed to load report %s: %v", reportID, err)
		return map[string]any{"enabled": true, "status": "error", "message": err.Error()}
	}
	items, err := loadOpsAuditItems(ctx, reportID)
	if err != nil {
		klog.Warningf("admin ops notification: failed to load report items %s: %v", reportID, err)
		return map[string]any{"enabled": true, "status": "error", "message": err.Error()}
	}

	signal := buildOpsReportNotificationSignal(report, policy)
	result := map[string]any{
		"enabled":              true,
		"report_id":            reportID,
		"failed_jobs":          signal.FailedJobs,
		"total_jobs":           signal.TotalJobs,
		"failure_rate_percent": signal.FailureRatePercent,
		"reasons":              signal.Reasons,
	}

	if policy.NotifyAdmins && signal.ShouldNotifyAdmins {
		adminResults, adminErr := alert.GetAlertMgr().NotifyPlatformAdmins(
			ctx,
			"智能巡检发现高风险问题",
			buildAdminOpsNotificationMessage(signal),
			buildAdminOpsReportURL(),
			"ops-report:admin_ops_report:admins",
			policy.CooldownHours,
		)
		result["admin_notifications"] = adminResults
		if adminErr != nil {
			result["admin_error"] = adminErr.Error()
		}
	}

	if policy.NotifyJobOwners && signal.ShouldNotifyOwners {
		ownerResults := notifyOpsReportJobOwners(ctx, reportID, signal, items, policy)
		result["job_owner_notifications"] = ownerResults
	}

	if _, ok := result["admin_notifications"]; !ok {
		result["admin_notifications"] = []any{}
	}
	if _, ok := result["job_owner_notifications"]; !ok {
		result["job_owner_notifications"] = []any{}
	}
	return result
}

func normalizeServiceOpsNotificationPolicy(policy *patrol.OpsReportNotificationPolicy) patrol.OpsReportNotificationPolicy {
	if policy == nil {
		return patrol.OpsReportNotificationPolicy{}
	}
	normalized := *policy
	if normalized.FailureJobThreshold <= 0 {
		normalized.FailureJobThreshold = 10
	}
	if normalized.FailureRateThresholdPercent <= 0 {
		normalized.FailureRateThresholdPercent = 15
	}
	if normalized.UnhealthyNodeThreshold <= 0 {
		normalized.UnhealthyNodeThreshold = 1
	}
	if normalized.NetworkAlertThreshold <= 0 {
		normalized.NetworkAlertThreshold = 3
	}
	if normalized.HighRiskNetworkJobThreshold <= 0 {
		normalized.HighRiskNetworkJobThreshold = 1
	}
	if normalized.MaxJobOwnerEmails <= 0 {
		normalized.MaxJobOwnerEmails = 10
	}
	if normalized.CooldownHours <= 0 {
		normalized.CooldownHours = 12
	}
	return normalized
}

func loadOpsAuditReport(ctx context.Context, reportID string) (*opsAuditReportRow, error) {
	var report opsAuditReportRow
	result := query.GetDB().WithContext(ctx).Raw(`
		SELECT id, summary, report_json, job_total, job_failed, job_pending
		FROM ops_audit_reports
		WHERE id = ? AND report_type = 'admin_ops_report'
	`, reportID).Scan(&report)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &report, nil
}

func loadOpsAuditItems(ctx context.Context, reportID string) ([]opsAuditItemRow, error) {
	var items []opsAuditItemRow
	err := query.GetDB().WithContext(ctx).Raw(`
		SELECT id, job_name, username, action_type, severity, category, failure_reason
		FROM ops_audit_items
		WHERE report_id = ?
		ORDER BY created_at ASC
	`, reportID).Scan(&items).Error
	return items, err
}

func buildOpsReportNotificationSignal(
	report *opsAuditReportRow,
	policy patrol.OpsReportNotificationPolicy,
) opsReportNotificationSignal {
	summary := decodeRawJSONMap(report.Summary)
	reportJSON := decodeRawJSONMap(report.ReportJSON)
	jobOverview := getNestedMap(reportJSON, "job_overview")
	nodeSummary := getNestedMap(summary, "node_summary")
	networkSummary := getNestedMap(summary, "network_summary")

	totalJobs := firstPositiveInt(report.JobTotal, getMapInt(jobOverview, "total"))
	failedJobs := firstPositiveInt(report.JobFailed, getMapInt(jobOverview, "failed"))
	failureRate := 0
	if totalJobs > 0 {
		failureRate = int(math.Round(float64(failedJobs) / float64(totalJobs) * 100))
	}

	unhealthyNodes := getMapListLen(nodeSummary, "unhealthy_nodes")
	if unhealthyNodes == 0 {
		totalNodes := getMapInt(nodeSummary, "total_nodes")
		readyNodes := getMapInt(nodeSummary, "ready_nodes")
		if totalNodes > readyNodes {
			unhealthyNodes = totalNodes - readyNodes
		}
	}
	networkAlerts := getMapInt(networkSummary, "network_alerts")
	highRiskNetworkJobs := getMapInt(networkSummary, "high_risk_jobs")

	reasons := make([]string, 0, 5)
	if failedJobs >= policy.FailureJobThreshold {
		reasons = append(reasons, fmt.Sprintf("失败作业数 %d 达到阈值 %d", failedJobs, policy.FailureJobThreshold))
	}
	if failureRate >= policy.FailureRateThresholdPercent {
		reasons = append(reasons, fmt.Sprintf("失败率 %d%% 达到阈值 %d%%", failureRate, policy.FailureRateThresholdPercent))
	}
	if unhealthyNodes >= policy.UnhealthyNodeThreshold {
		reasons = append(reasons, fmt.Sprintf("异常节点数 %d 达到阈值 %d", unhealthyNodes, policy.UnhealthyNodeThreshold))
	}
	if networkAlerts >= policy.NetworkAlertThreshold {
		reasons = append(reasons, fmt.Sprintf("网络告警数 %d 达到阈值 %d", networkAlerts, policy.NetworkAlertThreshold))
	}
	if highRiskNetworkJobs >= policy.HighRiskNetworkJobThreshold {
		reasons = append(reasons, fmt.Sprintf("高风险网络作业数 %d 达到阈值 %d", highRiskNetworkJobs, policy.HighRiskNetworkJobThreshold))
	}

	recommendations, hasHighRecommendation := extractHighRecommendations(reportJSON)
	if hasHighRecommendation && len(reasons) == 0 {
		reasons = append(reasons, "LLM/确定性巡检建议包含 high 级别事项")
	}

	return opsReportNotificationSignal{
		ShouldNotifyAdmins:  len(reasons) > 0,
		ShouldNotifyOwners:  failedJobs >= policy.FailureJobThreshold || failureRate >= policy.FailureRateThresholdPercent,
		Reasons:             reasons,
		FailedJobs:          failedJobs,
		TotalJobs:           totalJobs,
		FailureRatePercent:  failureRate,
		UnhealthyNodes:      unhealthyNodes,
		NetworkAlerts:       networkAlerts,
		HighRiskNetworkJobs: highRiskNetworkJobs,
		ExecutiveSummary:    strings.TrimSpace(fmt.Sprint(reportJSON["executive_summary"])),
		Recommendations:     recommendations,
	}
}

func notifyOpsReportJobOwners(
	ctx context.Context,
	reportID string,
	signal opsReportNotificationSignal,
	items []opsAuditItemRow,
	policy patrol.OpsReportNotificationPolicy,
) []map[string]any {
	results := make([]map[string]any, 0)
	seen := make(map[string]struct{})
	sentAttempts := 0
	subject := "作业失败巡检通知"
	for _, item := range items {
		if sentAttempts >= policy.MaxJobOwnerEmails {
			break
		}
		if !isFailureAuditItem(item) {
			continue
		}
		jobName := strings.TrimSpace(item.JobName)
		if jobName == "" {
			continue
		}
		if _, ok := seen[jobName]; ok {
			continue
		}
		seen[jobName] = struct{}{}

		entry := map[string]any{"job_name": jobName}
		suppressed, err := isNotificationSuppressed(ctx, jobName, alert.AgentJobOwnerNotificationAlertType, policy.CooldownHours)
		if err != nil {
			entry["status"] = "error"
			entry["message"] = err.Error()
			results = append(results, entry)
			continue
		}
		if suppressed {
			entry["status"] = "skipped"
			entry["reason"] = "cooldown"
			results = append(results, entry)
			continue
		}

		message := buildJobOwnerOpsNotificationMessage(reportID, signal, item)
		err = alert.GetAlertMgr().NotifyJobOwner(ctx, jobName, subject, message)
		sentAttempts++
		if err == nil {
			entry["status"] = "sent"
		} else if errors.Is(err, alert.ErrReceiverEmailMissing) {
			entry["status"] = "skipped"
			entry["reason"] = "owner_email_missing"
		} else {
			entry["status"] = "error"
			entry["message"] = err.Error()
		}
		results = append(results, entry)
	}
	return results
}

func isFailureAuditItem(item opsAuditItemRow) bool {
	return strings.EqualFold(strings.TrimSpace(item.ActionType), "failure_review") ||
		strings.EqualFold(strings.TrimSpace(item.Category), "failure")
}

func isNotificationSuppressed(ctx context.Context, key, alertType string, cooldownHours int) (bool, error) {
	if cooldownHours <= 0 {
		return false, nil
	}
	alertDB := query.Alert
	record, err := alertDB.WithContext(ctx).Where(alertDB.JobName.Eq(key), alertDB.AlertType.Eq(alertType)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return record.AlertTimestamp.After(time.Now().Add(-time.Duration(cooldownHours) * time.Hour)), nil
}

func buildAdminOpsNotificationMessage(signal opsReportNotificationSignal) string {
	parts := []string{
		fmt.Sprintf("智能巡检发现需要关注的问题：%s。", strings.Join(signal.Reasons, "；")),
		fmt.Sprintf("本周期作业总数 %d，失败 %d，失败率 %d%%。", signal.TotalJobs, signal.FailedJobs, signal.FailureRatePercent),
	}
	if signal.UnhealthyNodes > 0 || signal.NetworkAlerts > 0 || signal.HighRiskNetworkJobs > 0 {
		parts = append(parts, fmt.Sprintf("节点/网络摘要：异常节点 %d，网络告警 %d，高风险网络作业 %d。", signal.UnhealthyNodes, signal.NetworkAlerts, signal.HighRiskNetworkJobs))
	}
	if signal.ExecutiveSummary != "" {
		parts = append(parts, "报告摘要："+signal.ExecutiveSummary)
	}
	if len(signal.Recommendations) > 0 {
		parts = append(parts, "高优先级建议："+strings.Join(signal.Recommendations, "；"))
	}
	return strings.Join(parts, "\n\n")
}

func buildJobOwnerOpsNotificationMessage(reportID string, signal opsReportNotificationSignal, item opsAuditItemRow) string {
	parts := []string{
		fmt.Sprintf("平台智能巡检发现您的作业 %s 处于失败状态。", item.JobName),
		fmt.Sprintf("本周期共 %d 个作业，其中失败 %d 个，失败率 %d%%。", signal.TotalJobs, signal.FailedJobs, signal.FailureRatePercent),
	}
	if strings.TrimSpace(item.FailureReason) != "" {
		parts = append(parts, "识别到的失败原因："+strings.TrimSpace(item.FailureReason))
	}
	if signal.ExecutiveSummary != "" {
		parts = append(parts, "巡检摘要："+signal.ExecutiveSummary)
	}
	if reportID != "" {
		parts = append(parts, "巡检报告 ID："+reportID)
	}
	return strings.Join(parts, "\n\n")
}

func buildAdminOpsReportURL() string {
	host := strings.TrimSpace(pkgconfig.GetConfig().Host)
	if host == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/admin/aiops", host)
}

func decodeRawJSONMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return map[string]any{}
	}
	return result
}

func getNestedMap(m map[string]any, key string) map[string]any {
	if nested, ok := m[key].(map[string]any); ok {
		return nested
	}
	return map[string]any{}
}

func getMapInt(m map[string]any, key string) int {
	switch value := m[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		parsed, _ := value.Int64()
		return int(parsed)
	case string:
		var parsed int
		_, _ = fmt.Sscanf(strings.TrimSpace(value), "%d", &parsed)
		return parsed
	default:
		return 0
	}
}

func getMapListLen(m map[string]any, key string) int {
	if items, ok := m[key].([]any); ok {
		return len(items)
	}
	return 0
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func extractHighRecommendations(reportJSON map[string]any) ([]string, bool) {
	rawItems, _ := reportJSON["recommendations"].([]any)
	results := make([]string, 0, len(rawItems))
	hasHigh := false
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		severity := strings.ToLower(strings.TrimSpace(fmt.Sprint(item["severity"])))
		text := strings.TrimSpace(fmt.Sprint(item["text"]))
		if severity == "high" {
			hasHigh = true
			if text != "" {
				results = append(results, text)
			}
		}
	}
	return results, hasHigh
}
