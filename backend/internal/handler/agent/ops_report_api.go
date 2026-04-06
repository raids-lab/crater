package agent

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
)

// ─── Ops report admin API (read-only report display) ─────────────────────────

// OpsReportListItem is the summary shape returned for report lists.
type OpsReportListItem struct {
	ID            string          `json:"id"            gorm:"column:id"`
	ReportType    string          `json:"report_type"   gorm:"column:report_type"`
	Status        string          `json:"status"        gorm:"column:status"`
	TriggerSource *string         `json:"trigger_source" gorm:"column:trigger_source"`
	Summary       json.RawMessage `json:"summary"       gorm:"column:summary"`
	PeriodStart   *time.Time      `json:"period_start"  gorm:"column:period_start"`
	PeriodEnd     *time.Time      `json:"period_end"    gorm:"column:period_end"`
	JobTotal      int             `json:"job_total"     gorm:"column:job_total"`
	JobSuccess    int             `json:"job_success"   gorm:"column:job_success"`
	JobFailed     int             `json:"job_failed"    gorm:"column:job_failed"`
	JobPending    int             `json:"job_pending"   gorm:"column:job_pending"`
	CreatedAt     time.Time       `json:"created_at"    gorm:"column:created_at"`
}

// OpsReportDetail extends the list item with the full report JSON.
type OpsReportDetail struct {
	OpsReportListItem
	ReportJSON json.RawMessage `json:"report_json" gorm:"column:report_json"`
}

// OpsAuditItem represents a single audit item row.
type OpsAuditItem struct {
	ID                int64           `json:"id"                 gorm:"column:id"`
	ReportID          string          `json:"report_id"          gorm:"column:report_id"`
	JobName           string          `json:"job_name"           gorm:"column:job_name"`
	Username          *string         `json:"username"           gorm:"column:username"`
	ActionType        string          `json:"action_type"        gorm:"column:action_type"`
	Severity          string          `json:"severity"           gorm:"column:severity"`
	Category          *string         `json:"category"           gorm:"column:category"`
	JobType           *string         `json:"job_type"           gorm:"column:job_type"`
	Owner             *string         `json:"owner"              gorm:"column:owner"`
	Namespace         *string         `json:"namespace"          gorm:"column:namespace"`
	DurationSeconds   *int            `json:"duration_seconds"   gorm:"column:duration_seconds"`
	GPUUtilization    *float64        `json:"gpu_utilization"    gorm:"column:gpu_utilization"`
	GPURequested      *int            `json:"gpu_requested"      gorm:"column:gpu_requested"`
	GPUActualUsed     *int            `json:"gpu_actual_used"    gorm:"column:gpu_actual_used"`
	ResourceRequested json.RawMessage `json:"resource_requested" gorm:"column:resource_requested"`
	ResourceActual    json.RawMessage `json:"resource_actual"    gorm:"column:resource_actual"`
	ExitCode          *int            `json:"exit_code"          gorm:"column:exit_code"`
	FailureReason     *string         `json:"failure_reason"     gorm:"column:failure_reason"`
	AnalysisDetail    json.RawMessage `json:"analysis_detail"    gorm:"column:analysis_detail"`
	Handled           bool            `json:"handled"            gorm:"column:handled"`
	CreatedAt         time.Time       `json:"created_at"         gorm:"column:created_at"`
}

// ListOpsReports returns paginated admin_ops_report records.
func (mgr *AgentMgr) ListOpsReports(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	db := query.GetDB()
	var total int64
	db.Raw(`SELECT count(*) FROM ops_audit_reports WHERE report_type = 'admin_ops_report'`).Scan(&total)

	var items []OpsReportListItem
	result := db.Raw(`
		SELECT id, report_type, status, trigger_source, summary,
		       period_start, period_end, job_total, job_success, job_failed, job_pending,
		       created_at
		FROM ops_audit_reports
		WHERE report_type = 'admin_ops_report'
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, pageSize, (page-1)*pageSize).Scan(&items)

	if result.Error != nil {
		// Fallback: try without extended columns
		db.Raw(`
			SELECT id, report_type, status, trigger_source, summary, created_at
			FROM ops_audit_reports
			WHERE report_type = 'admin_ops_report'
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`, pageSize, (page-1)*pageSize).Scan(&items)
	}

	resputil.Success(c, gin.H{
		"total": total,
		"items": items,
	})
}

// GetLatestOpsReport returns the most recent admin_ops_report.
func (mgr *AgentMgr) GetLatestOpsReport(c *gin.Context) {
	db := query.GetDB()
	var report OpsReportDetail
	result := db.Raw(`
		SELECT id, report_type, status, trigger_source, summary, report_json,
		       period_start, period_end, job_total, job_success, job_failed, job_pending,
		       created_at
		FROM ops_audit_reports
		WHERE report_type = 'admin_ops_report' AND status = 'completed'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&report)

	if result.Error != nil || result.RowsAffected == 0 {
		// Fallback: try without extended columns (migration may not be applied)
		result = db.Raw(`
			SELECT id, report_type, status, trigger_source, summary, created_at
			FROM ops_audit_reports
			WHERE report_type = 'admin_ops_report' AND status = 'completed'
			ORDER BY created_at DESC
			LIMIT 1
		`).Scan(&report)
		if result.Error != nil || result.RowsAffected == 0 {
			resputil.Success(c, gin.H{"message": "暂无巡检报告"})
			return
		}
	}
	resputil.Success(c, report)
}

// GetOpsReportDetail returns a single report by ID with full report_json.
func (mgr *AgentMgr) GetOpsReportDetail(c *gin.Context) {
	reportID := c.Param("id")
	if reportID == "" {
		resputil.BadRequestError(c, "missing report id")
		return
	}

	db := query.GetDB()
	var report OpsReportDetail
	result := db.Raw(`
		SELECT id, report_type, status, trigger_source, summary, report_json,
		       period_start, period_end, job_total, job_success, job_failed, job_pending,
		       created_at
		FROM ops_audit_reports
		WHERE id = ? AND report_type = 'admin_ops_report'
	`, reportID).Scan(&report)

	if result.Error != nil || result.RowsAffected == 0 {
		resputil.Error(c, "report not found", resputil.NotSpecified)
		return
	}
	resputil.Success(c, report)
}

// GetOpsReportItems returns audit items for a specific report.
func (mgr *AgentMgr) GetOpsReportItems(c *gin.Context) {
	reportID := c.Param("id")
	if reportID == "" {
		resputil.BadRequestError(c, "missing report id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	category := c.Query("category")
	severity := c.Query("severity")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	db := query.GetDB()

	// Build WHERE clause
	where := "report_id = ?"
	params := []any{reportID}
	if category != "" {
		where += " AND category = ?"
		params = append(params, category)
	}
	if severity != "" {
		where += " AND severity = ?"
		params = append(params, severity)
	}

	var total int64
	db.Raw("SELECT count(*) FROM ops_audit_items WHERE "+where, params...).Scan(&total)

	params = append(params, pageSize, (page-1)*pageSize)
	var items []OpsAuditItem
	db.Raw(`
		SELECT id, report_id, job_name, username, action_type, severity,
		       category, job_type, owner, namespace, duration_seconds,
		       gpu_utilization, gpu_requested, gpu_actual_used,
		       resource_requested, resource_actual, exit_code, failure_reason,
		       analysis_detail, handled, created_at
		FROM ops_audit_items
		WHERE `+where+`
		ORDER BY CASE severity
		           WHEN 'critical' THEN 1
		           WHEN 'warning' THEN 2
		           WHEN 'info' THEN 3
		           ELSE 4 END,
		         created_at DESC
		LIMIT ? OFFSET ?
	`, params...).Scan(&items)

	resputil.Success(c, gin.H{
		"total": total,
		"items": items,
	})
}
