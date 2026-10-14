package operations

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

type cronJobRecordPageQuery struct {
	Page      int        `form:"page,default=1" binding:"min=1"`
	PageSize  int        `form:"page_size,default=10" binding:"min=1,max=200"`
	Search    string     `form:"search"`
	Sort      string     `form:"sort"`
	StartTime *time.Time `form:"start_time" time_format:"2006-01-02T15:04:05Z07:00"`
	EndTime   *time.Time `form:"end_time" time_format:"2006-01-02T15:04:05Z07:00"`
}

var cronJobRecordSortFields = map[string]string{
	"name":        "name",
	"executeTime": "execute_time",
	"status":      "status",
	"createdAt":   "created_at",
}

var cronJobRecordStatuses = map[string]struct{}{
	string(model.CronJobRecordStatusUnknown): {},
	string(model.CronJobRecordStatusSuccess): {},
	string(model.CronJobRecordStatusFailed):  {},
}

func cronJobRecordSortClauses(sortValue string) []string {
	return pagination.SortClauses(sortValue, cronJobRecordSortFields, "id DESC")
}

func bindCronJobRecordPageQuery(c *gin.Context) (cronJobRecordPageQuery, []string, error) {
	var request cronJobRecordPageQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		return cronJobRecordPageQuery{}, nil, bizerr.BadRequest.ParameterError.Wrap(
			err,
			"invalid cronjob record list query",
		)
	}
	var err error
	request.Search, err = pagination.NormalizePageQuery(request.Search, request.Page, request.PageSize)
	if err != nil {
		return cronJobRecordPageQuery{}, nil, err
	}
	request.Sort, err = pagination.DefaultAndValidateSort(request.Sort, "-executeTime", cronJobRecordSortFields)
	if err != nil {
		return cronJobRecordPageQuery{}, nil, err
	}

	statuses := c.QueryArray("status")
	for _, status := range statuses {
		if _, ok := cronJobRecordStatuses[status]; !ok {
			return cronJobRecordPageQuery{}, nil, bizerr.BadRequest.ParameterError.New(
				fmt.Sprintf("unsupported cronjob record status %q", status),
			)
		}
	}
	return request, statuses, nil
}

// GetCronjobRecordsPage returns cronjob records using the shared page protocol.
//
//	@Summary	获取定时任务执行记录分页
//	@Description	按统一分页协议获取执行记录，搜索、筛选、排序和分页在数据库中执行
//	@Tags		Operations
//	@Produce	json
//	@Security	Bearer
//	@Param		page	query	int	false	"Page number"
//	@Param		page_size	query	int	false	"Page size, 1-200"
//	@Param		search	query	string	false	"Search job name or message"
//	@Param		status	query	string	false	"Filter by status; repeatable"
//	@Param		scope_name	query	string	false	"Restrict to job names; repeatable"
//	@Param		start_time	query	string	false	"Start time, RFC3339"
//	@Param		end_time	query	string	false	"End time, RFC3339"
//	@Param		sort	query	string	false	"Sort fields"
//	@Success	200	{object}	resputil.Response[resputil.Page[model.CronJobRecord]]
//	@Router		/v1/admin/operations/cronjob/record/page [get]
func (mgr *OperationsMgr) GetCronjobRecordsPage(c *gin.Context) {
	request, statuses, err := bindCronJobRecordPageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	db := query.GetDB().WithContext(c).Model(&model.CronJobRecord{})
	if scopeNames := c.QueryArray("scope_name"); len(scopeNames) > 0 {
		db = db.Where("name IN ?", scopeNames)
	}
	if request.StartTime != nil {
		db = db.Where("execute_time >= ?", *request.StartTime)
	}
	if request.EndTime != nil {
		db = db.Where("execute_time <= ?", *request.EndTime)
	}
	if len(statuses) > 0 {
		db = db.Where("status IN ?", statuses)
	}
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where("LOWER(name) LIKE ? ESCAPE '\\' OR LOWER(message) LIKE ? ESCAPE '\\'", pattern, pattern)
	}
	for _, clause := range cronJobRecordSortClauses(request.Sort) {
		db = db.Order(clause)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		klog.Errorf("failed to count cronjob records: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count cronjob records failed"))
		return
	}
	records := make([]*model.CronJobRecord, 0, request.PageSize)
	if err := db.Offset((request.Page - 1) * request.PageSize).
		Limit(request.PageSize).
		Find(&records).Error; err != nil {
		klog.Errorf("failed to list cronjob records: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list cronjob records failed"))
		return
	}

	resputil.Success(c, resputil.NewPage(records, total, request.Page, request.PageSize))
}
