package operations

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
)

type CronjobConfigs struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Schedule string         `json:"schedule"`
	Suspend  bool           `json:"suspend"`
	Configs  map[string]any `json:"configs"`
}

// UpdateCronjobConfig godoc
//
//	@Summary		Update cronjob config
//	@Description	Update one cronjob config
//	@Tags			Operations
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			use	body		CronjobConfigs			true	"CronjobConfigs"
//	@Success		200	{object}	resputil.Response[any]	"Success"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/operations/cronjob [put]
func (mgr *OperationsMgr) UpdateCronjobConfig(c *gin.Context) {
	var req CronjobConfigs
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	var (
		jobTypePtr *model.CronJobType
		specPtr    *string
		configPtr  *string
	)
	if req.Type != "" {
		jobTypePtr = ptr.To(model.CronJobType(req.Type))
	}
	if req.Schedule != "" {
		specPtr = ptr.To(req.Schedule)
	}

	if len(req.Configs) > 0 {
		configJson, err := json.Marshal(req.Configs)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
		}
		configPtr = ptr.To(string(configJson))
	}
	if err := mgr.cronJobManager.UpdateJobConfig(c, req.Name, jobTypePtr, specPtr, &req.Suspend, configPtr); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
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
	jobs, err := mgr.cronJobManager.GetAllCronJobs(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	configs := lo.Map(jobs, func(job *model.CronJobConfig, _ int) CronjobConfigs {
		config := make(map[string]any)
		if err := json.Unmarshal(job.Config, &config); err != nil {
			config = map[string]any{}
		}
		ret := CronjobConfigs{
			Name:     job.Name,
			Type:     string(job.Type),
			Schedule: job.Spec,
			Suspend:  job.GetSuspend(),
			Configs:  config,
		}
		return ret
	})
	resputil.Success(c, configs)
}

func (mgr *OperationsMgr) GetCronjobNames(c *gin.Context) {
	names, err := mgr.cronJobManager.GetCronjobNames(c)
	if err != nil {
		klog.Error(err)
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}
	resputil.Success(c, names)
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
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	records, total, err := cm.cronJobManager.GetCronjobRecords(
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
		"total":   total,
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
