package operations

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/utils"
)

type SetKeepRequest struct {
	Name string `uri:"name" binding:"required"`
}

type SetLockTimeRequest struct {
	Name        string `json:"name" binding:"required"`
	IsPermanent bool   `json:"isPermanent"`
	Days        int    `json:"days"`
	Hours       int    `json:"hours"`
	Minutes     int    `json:"minutes"`
}

type ClearLockTimeRequest struct {
	Name string `json:"name" binding:"required"`
}

// SetKeepWhenLowResourceUsage godoc
// @Summary set KeepWhenLowResourceUsage of the job to the opposite value
// @Description set KeepWhenLowResourceUsage of the job to the opposite value
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/keep/{name} [put]
func (mgr *OperationsMgr) SetKeepWhenLowResourceUsage(c *gin.Context) {
	var req SetKeepRequest
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	jobDB := query.Job
	j, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.Name)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	pre := j.KeepWhenLowResourceUsage
	if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.Name)).Update(jobDB.KeepWhenLowResourceUsage, !pre); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	message := fmt.Sprintf("Set %s keepWhenLowResourceUsage to %t", req.Name, !j.KeepWhenLowResourceUsage)
	resputil.Success(c, message)
}

// SetLockTime godoc
// @Summary set LockTime of the job
// @Description set LockTime of the job
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/add/locktime [put]
func (mgr *OperationsMgr) AddLockTime(c *gin.Context) {
	var req SetLockTimeRequest

	// JSON 参数
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	jobDB := query.Job

	// 检查是否已经永久锁定
	j, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.Name)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	// 若永久锁定，则不允许延长锁定时间
	if j.LockedTimestamp == utils.GetPermanentTime() {
		resputil.Error(c, "Job is already permanently locked", resputil.NotSpecified)
		return
	}

	// 初始值
	lockTime := utils.GetLocalTime()
	if j.LockedTimestamp.After(utils.GetLocalTime()) {
		lockTime = j.LockedTimestamp
	}

	if req.IsPermanent {
		lockTime = utils.GetPermanentTime()
	} else {
		lockTime = lockTime.Add(
			time.Duration(req.Days)*24*time.Hour + time.Duration(req.Hours)*time.Hour + time.Duration(req.Minutes)*time.Minute,
		)
	}

	if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.Name)).Update(jobDB.LockedTimestamp, lockTime); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	message := fmt.Sprintf("Parmanently lock %s", req.Name)
	if !req.IsPermanent {
		message = fmt.Sprintf("Set %s lockTime to %s", req.Name, lockTime.Format("2006-01-02 15:04:05"))
	}
	resputil.Success(c, message)
}

// ClearLockTime godoc
// @Summary clear LockTime of the job
// @Description clear LockTime of the job
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/clear/locktime [put]
func (mgr *OperationsMgr) ClearLockTime(c *gin.Context) {
	var req ClearLockTimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	jobDB := query.Job
	if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.Name)).Update(jobDB.LockedTimestamp, nil); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	message := fmt.Sprintf("Clear %s lockTime", req.Name)
	resputil.Success(c, message)
}

// GetWhiteList godoc
// @Summary Get job white list
// @Description get job white list
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/whitelist [get]
func (mgr *OperationsMgr) GetWhiteList(c *gin.Context) {
	whiteList, err := mgr.getJobWhiteList(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, whiteList)
}

func (mgr *OperationsMgr) getJobWhiteList(c *gin.Context) ([]string, error) {
	var cleanList []string
	jobDB := query.Job
	curTime := utils.GetLocalTime()

	data, err := jobDB.WithContext(c).Where(jobDB.LockedTimestamp.Gt(curTime)).Find()

	if err != nil {
		return nil, err
	}
	for _, item := range data {
		cleanList = append(cleanList, item.JobName)
	}
	return cleanList, nil
}
