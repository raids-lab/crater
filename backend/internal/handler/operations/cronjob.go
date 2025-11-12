package operations

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
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
	var (
		jobTypePtr *model.CronJobType
		specPtr    *string
		statusPtr  *model.CronJobConfigStatus
		configPtr  *string
	)
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
		configJson, err := json.Marshal(req.Config)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.ServiceError)
		}
		configPtr = ptr.To(string(configJson))
	}
	if err := mgr.cronJobManager.UpdateJobConfig(c, req.Name, jobTypePtr, specPtr, statusPtr, configPtr); err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	resputil.Success(c, "Successfully update cronjob config")
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

type GetCronJobRecordsReq struct {
	Name      []string   `json:"name" form:"name"`
	StartTime *time.Time `json:"startTime" form:"startTime"`
	EndTime   *time.Time `json:"endTime" form:"endTime"`
	Status    *string    `json:"status" form:"status"`
}

func (cm *OperationsMgr) GetCronjobRecords(c *gin.Context) {
	req := &GetCronJobRecordsReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}

	records, err := cm.cronJobManager.GetCronjobRecords(
		c,
		req.Name,
		req.StartTime,
		req.EndTime,
		req.Status,
	)
	if err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	resputil.Success(c, map[string]any{
		"records": records,
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
