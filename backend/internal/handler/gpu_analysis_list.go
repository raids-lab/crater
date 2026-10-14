package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

const (
	lowRiskThreshold  = 3
	highRiskThreshold = 7
)

type gpuAnalysisListPageQuery struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=200"`
	Search   string `form:"search"`
	Sort     string `form:"sort"`
}

var gpuAnalysisSortFields = map[string]string{
	"JobName":      "gpu_analyses.job_name",
	"UserName":     "gpu_analyses.user_name",
	"JobType":      "jobs.job_type",
	"ReviewStatus": "gpu_analyses.review_status",
	"Phase2Score":  "gpu_analyses.phase2_score",
	"CreatedAt":    "gpu_analyses.created_at",
}

func bindGpuAnalysisListPageQuery(
	c *gin.Context,
) (request gpuAnalysisListPageQuery, statuses []int, riskLevels []string, err error) {
	if err := c.ShouldBindQuery(&request); err != nil {
		return gpuAnalysisListPageQuery{}, nil, nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid gpu analysis list query")
	}
	request.Search, err = pagination.NormalizePageQuery(request.Search, request.Page, request.PageSize)
	if err != nil {
		return gpuAnalysisListPageQuery{}, nil, nil, err
	}

	statuses = make([]int, 0)
	seenStatuses := make(map[int]struct{})
	for _, raw := range c.QueryArray("ReviewStatus") {
		status, err := strconv.Atoi(raw)
		if err != nil || status < int(model.ReviewStatusPending) || status > int(model.ReviewStatusIgnored) {
			return gpuAnalysisListPageQuery{}, nil, nil, bizerr.BadRequest.ParameterError.New(fmt.Sprintf("unsupported review status %q", raw))
		}
		if _, seen := seenStatuses[status]; seen {
			continue
		}
		seenStatuses[status] = struct{}{}
		statuses = append(statuses, status)
	}

	riskLevels = make([]string, 0)
	seenRisks := make(map[string]struct{})
	for _, risk := range c.QueryArray("Phase2Score") {
		if risk != "high" && risk != "medium" && risk != "low" {
			return gpuAnalysisListPageQuery{}, nil, nil, bizerr.BadRequest.ParameterError.New(fmt.Sprintf("unsupported risk level %q", risk))
		}
		if _, seen := seenRisks[risk]; seen {
			continue
		}
		seenRisks[risk] = struct{}{}
		riskLevels = append(riskLevels, risk)
	}

	request.Sort, err = pagination.DefaultAndValidateSort(
		request.Sort,
		"-Phase2Score,-CreatedAt",
		gpuAnalysisSortFields,
	)
	if err != nil {
		return gpuAnalysisListPageQuery{}, nil, nil, err
	}
	return request, statuses, riskLevels, nil
}

func gpuAnalysisSortClauses(sortValue string) []string {
	return pagination.SortClauses(sortValue, gpuAnalysisSortFields, "gpu_analyses.id DESC")
}

// ListAnalysesPage returns GPU analysis records with joined job information in the shared page shape.
// The legacy array endpoint remains available for compatibility.
//
//	@Summary	管理员分页获取 GPU 分析记录
//	@Description	按搜索、审核状态和风险等级过滤后排序分页返回 GPU 分析记录
//	@Tags		GpuAnalysis
//	@Produce	json
//	@Security	Bearer
//	@Param		page		query	int		false	"Page number"
//	@Param		page_size	query	int		false	"Page size, 1-200"
//	@Param		search		query	string	false	"Search job or user name"
//	@Param		ReviewStatus	query	int		false	"Filter by review status; repeatable"
//	@Param		Phase2Score	query	string	false	"Filter by risk level; repeatable"
//	@Param		sort		query	string	false	"Sort fields"
//	@Success	200	{object}	resputil.Response[resputil.Page[GpuAnalysisWithJobInfo]]
//	@Failure	400	{object}	resputil.Response[any]
//	@Failure	500	{object}	resputil.Response[any]
//	@Router		/v1/admin/gpu-analysis/page [get]
func (mgr *GpuAnalysisMgr) ListAnalysesPage(c *gin.Context) {
	request, statuses, riskLevels, err := bindGpuAnalysisListPageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	db := query.GpuAnalysis.WithContext(c).UnderlyingDB().Model(&model.GpuAnalysis{})
	if len(statuses) > 0 {
		db = db.Where("gpu_analyses.review_status IN ?", statuses)
	}
	if len(riskLevels) > 0 {
		riskConditions := make([]string, 0, len(riskLevels))
		riskArgs := make([]any, 0, len(riskLevels)*2)
		for _, risk := range riskLevels {
			switch risk {
			case "high":
				riskConditions = append(riskConditions, "gpu_analyses.phase2_score >= ?")
				riskArgs = append(riskArgs, highRiskThreshold)
			case "medium":
				riskConditions = append(riskConditions, "gpu_analyses.phase2_score > ? AND gpu_analyses.phase2_score < ?")
				riskArgs = append(riskArgs, lowRiskThreshold, highRiskThreshold)
			case "low":
				riskConditions = append(riskConditions, "gpu_analyses.phase2_score <= ?")
				riskArgs = append(riskArgs, lowRiskThreshold)
			}
		}
		db = db.Where("("+strings.Join(riskConditions, " OR ")+")", riskArgs...)
	}
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where(
			"(LOWER(gpu_analyses.job_name) LIKE ? ESCAPE '\\' OR "+
				"LOWER(gpu_analyses.user_name) LIKE ? ESCAPE '\\' OR "+
				"LOWER(jobs.name) LIKE ? ESCAPE '\\')",
			pattern,
			pattern,
			pattern,
		)
	}

	db = db.Joins("LEFT JOIN jobs ON gpu_analyses.job_name = jobs.job_name").Joins("LEFT JOIN users ON gpu_analyses.user_id = users.id")
	for _, clause := range gpuAnalysisSortClauses(request.Sort) {
		db = db.Order(clause)
	}

	var total int64
	if err := db.Distinct("gpu_analyses.id").Count(&total).Error; err != nil {
		klog.Errorf("failed to count gpu analyses: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count gpu analyses failed"))
		return
	}

	results := make([]GpuAnalysisWithJobInfo, 0)
	selectFields := "gpu_analyses.*, jobs.name, jobs.job_type, jobs.resources, jobs.nodes, jobs.status, " +
		"jobs.locked_timestamp, users.nickname AS user_nickname"
	if err := db.Select(selectFields).Offset((request.Page - 1) * request.PageSize).Limit(request.PageSize).Scan(&results).Error; err != nil {
		klog.Errorf("failed to list gpu analyses: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list gpu analyses failed"))
		return
	}

	resputil.Success(c, resputil.NewPage(results, total, request.Page, request.PageSize))
}
