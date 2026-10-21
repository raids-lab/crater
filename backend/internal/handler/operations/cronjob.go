package operations

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/patrol"
	"github.com/samber/lo"
	"gorm.io/gen/field"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

// UpdateCronjobConfig godoc
//
//	@Summary		Update cronjob config
//	@Description	Update one cronjob config
//	@Tags			Operations
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			use	body		model.CronJobConfig			true	"CronjobConfig"
//	@Success		200	{object}	resputil.Response[any]	"Success"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/operations/cronjob [put]
func (mgr *OperationsMgr) UpdateCronjobConfig(c *gin.Context) {
	var req model.CronJobConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}

	jobTypePtr, specPtr, statusPtr, configPtr, err := buildCronUpdateParams(&req)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	if err := mgr.validateCronEnableGate(c, &req); err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}

	if err := mgr.cronJobManager.UpdateJobConfig(c, req.Name, jobTypePtr, specPtr, statusPtr, configPtr); err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	if req.Name == patrol.TRIGGER_BILLING_BASE_LOOP_JOB &&
		req.Status == model.CronJobConfigStatusSuspended &&
		mgr.billingService != nil {
		if err := mgr.billingService.HandleBaseLoopCronSuspended(c.Request.Context()); err != nil {
			resputil.Error(c, err.Error(), resputil.ServiceError)
			return
		}
	}
	resputil.Success(c, "Successfully update cronjob config")
}

func buildCronUpdateParams(
	req *model.CronJobConfig,
) (
	jobTypePtr *model.CronJobType,
	specPtr *string,
	statusPtr *model.CronJobConfigStatus,
	configPtr *string,
	err error,
) {
	if req.Type != "" {
		jobTypePtr = ptr.To(req.Type)
	}
	if req.Spec != "" {
		specPtr = ptr.To(req.Spec)
	}
	if req.Status != "" {
		statusPtr = ptr.To(req.Status)
	}
	if len(req.Config) > 0 {
		var configJSON []byte
		configJSON, err = json.Marshal(req.Config)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		configPtr = ptr.To(string(configJSON))
	}
	return jobTypePtr, specPtr, statusPtr, configPtr, nil
}

func (mgr *OperationsMgr) validateCronEnableGate(c *gin.Context, req *model.CronJobConfig) error {
	if req.Status == "" || req.Status == model.CronJobConfigStatusSuspended {
		return nil
	}
	if req.Name == patrol.TRIGGER_GPU_ANALYSIS_JOB {
		if !mgr.configService.IsGpuAnalysisEnabled(c.Request.Context()) {
			return fmt.Errorf("cannot enable the GPU analysis cron job because the main GPU analysis feature is disabled in system settings")
		}
		return nil
	}
	if req.Name == patrol.TRIGGER_BILLING_BASE_LOOP_JOB {
		if mgr.billingService == nil || !mgr.billingService.IsFeatureEnabled(c.Request.Context()) {
			return fmt.Errorf("cannot enable billing cron job because billing feature is disabled in system settings")
		}
		if !mgr.billingService.IsActive(c.Request.Context()) {
			return fmt.Errorf("cannot enable billing cron job because billing active is disabled in system settings")
		}
	}
	return nil
}

// GetCronjobConfigs godoc
//
//	@Summary		Get all cronjob configs
//	@Description	Get all cronjob configs
//	@Tags			Operations
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"Success"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/operations/cronjob [get]
func (mgr *OperationsMgr) GetCronjobConfigs(c *gin.Context) {
	configs, err := mgr.cronJobManager.GetCronjobConfigs(
		c, nil, nil, nil, nil,
	)
	if err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	resputil.Success(c, map[string]any{
		"configs": configs,
		"total":   len(configs),
	})
}

func (mgr *OperationsMgr) GetCronjobNames(c *gin.Context) {
	configs, err := mgr.cronJobManager.GetCronjobConfigs(
		c, nil, nil, nil, nil,
	)
	if err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	names := lo.Map(configs, func(item *model.CronJobConfig, _ int) string {
		return item.Name
	})
	resputil.Success(c, names)
}

type GetCronjobConfigStatusReq struct {
	Name []string `json:"name" form:"name"`
}

type GetCronjobConfigStatusRespItem struct {
	Name    string                    `json:"name"`
	Status  model.CronJobConfigStatus `json:"status"`
	EntryID int                       `json:"entry_id"`
}

func (mgr *OperationsMgr) GetCronjobConfigStatus(c *gin.Context) {
	req := &GetCronjobConfigStatusReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}
	configs, err := mgr.cronJobManager.GetCronjobConfigs(
		c, req.Name, nil, nil, nil,
	)
	if err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	status := lo.SliceToMap(configs, func(item *model.CronJobConfig) (string, *GetCronjobConfigStatusRespItem) {
		return item.Name, &GetCronjobConfigStatusRespItem{
			Name:    item.Name,
			Status:  item.Status,
			EntryID: item.EntryID,
		}
	})
	resputil.Success(c, status)
}

func (mgr *OperationsMgr) GetCronjobRecordTimeRange(c *gin.Context) {
	startTime, endTime, err := mgr.cronJobManager.GetCronjobRecordTimeRange(c)
	if err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	resputil.Success(c, map[string]any{
		"startTime": startTime,
		"endTime":   endTime,
	})
}

type DeleteCronJobRecordsReq struct {
	ID        []uint     `json:"id"`
	StartTime *time.Time `json:"startTime"`
	EndTime   *time.Time `json:"endTime"`
}

func (cm *OperationsMgr) DeleteCronjobRecords(c *gin.Context) {
	req := &DeleteCronJobRecordsReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}

	if len(req.ID) == 0 && req.StartTime == nil && req.EndTime == nil {
		resputil.Error(c, "id or startTime or endTime is required", resputil.InvalidRequest)
		return
	}

	deleted, err := cm.cronJobManager.DeleteCronjobRecords(c, req.ID, req.StartTime, req.EndTime)
	if err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	resputil.Success(c, map[string]string{
		"deleted": fmt.Sprintf("%d", deleted),
	})
}

type GetLastCronjobRecordReq struct {
	Name      []string   `json:"name"`
	Status    *string    `json:"status"`
	StartTime *time.Time `json:"startTime"`
	EndTime   *time.Time `json:"endTime"`
}

func (cm *OperationsMgr) GetLastCronjobRecord(c *gin.Context) {
	req := &GetLastCronjobRecordReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}

	records, err := cm.cronJobManager.GetLastCronjobRecord(c, req.Name, req.Status, req.StartTime, req.EndTime)
	if err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	resputil.Success(c, records)
}

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

// GetCronjobRecords returns cronjob records using the shared page protocol.
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
//	@Router		/v1/admin/operations/cronjob/record/list [get]
func (mgr *OperationsMgr) GetCronjobRecords(c *gin.Context) {
	request, statuses, err := bindCronJobRecordPageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	q := query.CronJobRecord
	db := q.WithContext(c)
	if names := c.QueryArray("scope_name"); len(names) > 0 {
		db = db.Where(q.Name.In(names...))
	}
	if request.StartTime != nil {
		db = db.Where(q.ExecuteTime.Gte(*request.StartTime))
	}
	if request.EndTime != nil {
		db = db.Where(q.ExecuteTime.Lte(*request.EndTime))
	}
	if len(statuses) > 0 {
		db = db.Where(q.Status.In(statuses...))
	}
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where(field.Or(q.Name.Lower().Like(pattern), q.Message.Lower().Like(pattern)))
	}
	records, total, err := db.Order(pagination.SortFields(request.Sort, cronJobRecordSortFields, "id DESC")...).
		FindByPage((request.Page-1)*request.PageSize, request.PageSize)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list cronjob records failed"))
		return
	}
	resputil.Success(c, resputil.NewPage(records, total, request.Page, request.PageSize))
}
