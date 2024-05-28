package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/logutils"
)

type FileMgr struct {
}

func NewFileMgr() Manager {
	return &FileMgr{}
}

func (mgr *FileMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *FileMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/mydataset", mgr.GetAllDataset)
	g.POST("/create", mgr.CreateDataset)
	g.DELETE("/delete/:id", mgr.DeleteDataset)
	g.POST("/share/user", mgr.ShareDatasetWithUser)
	g.POST("/share/queue", mgr.ShareDatasetWithQueue)
}

func (mgr *FileMgr) RegisterAdmin(_ *gin.RouterGroup) {
}

type DatasetResp []*model.Dataset

// GetAllDataset godoc
// @Summary 获取数据集
// @Description 获取数据集
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/mydataset [get]
func (mgr *FileMgr) GetAllDataset(c *gin.Context) {
	token := util.GetToken(c)
	datasets := make(map[uint]*model.Dataset)
	ud := query.UserDataset
	d := query.Dataset
	qd := query.QueueDataset
	userDataset, err := ud.WithContext(c).Where(ud.UserID.Eq(token.UserID)).Find()
	if err != nil {
		logutils.Log.Infof("Can't get , err: %v", err)
		resputil.Error(c, "Can't get mydatasets", resputil.NotSpecified)
		return
	}
	for i := range userDataset {
		dataset, uderr := d.WithContext(c).Where(d.ID.Eq(userDataset[i].DatasetID)).First()
		if uderr != nil {
			resputil.Error(c, fmt.Sprintf("Get user's dataset failed, err %v", err), resputil.NotSpecified)
			return
		}
		datasets[dataset.ID] = dataset
	}
	queueDataset, err := qd.WithContext(c).Where(qd.QueueID.Eq(1)).Find()
	if err != nil {
		resputil.Error(c, "Can't get public queuedatasets", resputil.NotSpecified)
		return
	}
	for i := range queueDataset {
		dataset, qderr := d.WithContext(c).Where(d.ID.Eq(queueDataset[i].DatasetID)).First()
		if qderr != nil {
			resputil.Error(c, fmt.Sprintf("Get public dataset failed, err %v", err), resputil.NotSpecified)
			return
		}
		datasets[dataset.ID] = dataset
	}
	if token.QueueID != 0 && token.QueueID != 1 {
		myqueueDataset, err := qd.WithContext(c).Where(qd.QueueID.Eq(token.QueueID)).Find()
		if err != nil {
			resputil.Error(c, "Can't get my queuedatasets", resputil.NotSpecified)
			return
		}
		for i := range myqueueDataset {
			dataset, err := d.WithContext(c).Where(d.ID.Eq(myqueueDataset[i].DatasetID)).First()
			if err != nil {
				resputil.Error(c, fmt.Sprintf("Get my queue's dataset failed, err %v", err), resputil.NotSpecified)
				return
			}
			datasets[dataset.ID] = dataset
		}
	}
	result := make([]*model.Dataset, 0, len(datasets))
	for _, data := range datasets {
		result = append(result, data)
	}
	resputil.Success(c, result)
}

type DatasetReq struct {
	Name     string `json:"name" binding:"required"`
	URL      string `json:"url" binding:"required"`
	Describe string `json:"describe"`
}

// CreateDataset godoc
// @Summary 创建数据集
// @Description 输入数据集名字和URL，创建数据集
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param datasetReq body DatasetReq true "参数描述"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/create [post]
func (mgr *FileMgr) CreateDataset(c *gin.Context) {
	token := util.GetToken(c)
	var datasetReq DatasetReq
	if err := c.ShouldBindJSON(&datasetReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	db := query.Use(query.DB)
	err := db.Transaction(func(tx *query.Query) error {
		d := tx.Dataset
		ud := tx.UserDataset
		var dataset model.Dataset
		dataset.Name = datasetReq.Name
		dataset.Describe = datasetReq.Describe
		dataset.URL = datasetReq.URL
		dataset.UserID = token.UserID
		if err := d.WithContext(c).Create(&dataset); err != nil {
			return err
		}
		var userDataset model.UserDataset
		userDataset.UserID = token.UserID
		userDataset.DatasetID = dataset.ID
		if err := ud.WithContext(c).Create(&userDataset); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	} else {
		resputil.Success(c, "created dataset successfully")
	}
}

type SharedUserReq struct {
	DatasetID uint `json:"datasetID" binding:"required"`
	UserID    uint `json:"userID" binding:"required"`
}

// ShareDatasetWithUser godoc
// @Summary 跟用户共享数据集
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param userReq body SharedUserReq true "共享数据集用户"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/share/user [post]
func (mgr *FileMgr) ShareDatasetWithUser(c *gin.Context) {
	token := util.GetToken(c)
	var userReq SharedUserReq
	if err := c.ShouldBindJSON(&userReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	u := query.User
	d := query.Dataset
	_, err := d.WithContext(c).Where(d.UserID.Eq(token.UserID), d.ID.Eq(userReq.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "you has no permission or this dataset not exist", resputil.InvalidRequest)
		return
	}
	_, err = u.WithContext(c).Where(u.ID.Eq(userReq.UserID)).First()
	if err != nil {
		resputil.Error(c, "user not exist", resputil.InvalidRequest)
		return
	}
	ud := query.UserDataset
	hud, _ := ud.WithContext(c).Where(ud.UserID.Eq(userReq.UserID), ud.DatasetID.Eq(userReq.DatasetID)).First()
	if hud != nil {
		resputil.Error(c, "user has shared dataset", resputil.InvalidRequest)
		return
	}
	userDataset := model.UserDataset{
		UserID:    userReq.UserID,
		DatasetID: userReq.DatasetID,
	}
	if err := ud.WithContext(c).Create(&userDataset); err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}
	resputil.Success(c, "Shared successfully")
}

type SharedQueueReq struct {
	DatasetID uint `json:"datasetID" binding:"required"`
	QueueID   uint `json:"queueID" binding:"required"`
}

// ShareDatasetWithQueue godoc
// @Summary 跟队列共享数据集
// @Description 跟队列共享数据集
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param queueReq body SharedQueueReq true "共享数据集队列"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/share/queue [post]
func (mgr *FileMgr) ShareDatasetWithQueue(c *gin.Context) {
	token := util.GetToken(c)
	var queueReq SharedQueueReq
	if err := c.ShouldBindJSON(&queueReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	q := query.Queue
	d := query.Dataset
	qd := query.QueueDataset
	_, err := d.WithContext(c).Where(d.UserID.Eq(token.UserID), d.ID.Eq(queueReq.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "you has no permission or this dataset not exist", resputil.InvalidRequest)
		return
	}
	_, err = q.WithContext(c).Where(q.ID.Eq(queueReq.QueueID)).First()
	if err != nil {
		resputil.Error(c, "queue not exist", resputil.InvalidRequest)
		return
	}
	hqd, _ := qd.WithContext(c).Where(qd.QueueID.Eq(queueReq.QueueID), qd.DatasetID.Eq(queueReq.DatasetID)).First()
	if hqd != nil {
		resputil.Error(c, "queue has shared dataset", resputil.InvalidRequest)
		return
	}
	queuedataset := model.QueueDataset{
		QueueID:   queueReq.QueueID,
		DatasetID: queueReq.DatasetID,
	}
	if err := qd.WithContext(c).Create(&queuedataset); err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}
	resputil.Success(c, "Shared successfully")
}

type DeleteDatasetReq struct {
	ID uint `uri:"id" binding:"required"`
}

// DeleteDataset godoc
// @Summary 删除数据集
// @Description 删除数据集
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param req path DeleteDatasetReq true "删除数据集ID"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/delete/{id} [delete]
func (mgr *FileMgr) DeleteDataset(c *gin.Context) {
	token := util.GetToken(c)
	var req DeleteDatasetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate delete parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	db := query.Use(query.DB)
	err := db.Transaction(func(tx *query.Query) error {
		d := tx.Dataset
		ud := tx.UserDataset
		qd := tx.QueueDataset
		dataset, err := d.WithContext(c).Where(d.ID.Eq(req.ID)).First()
		if err != nil {
			return err
		}
		if dataset.UserID != token.UserID && token.RolePlatform != model.RoleAdmin {
			return fmt.Errorf("you has no permission")
		}
		userDataset, err := ud.WithContext(c).Where(ud.DatasetID.Eq(req.ID)).Find()
		if err != nil {
			return err
		}
		queueDataset, err := qd.WithContext(c).Where(qd.DatasetID.Eq(req.ID)).Find()
		if err != nil {
			return err
		}
		if _, err := ud.WithContext(c).Delete(userDataset...); err != nil {
			return err
		}
		if _, err := qd.WithContext(c).Delete(queueDataset...); err != nil {
			return err
		}
		if _, err := d.WithContext(c).Delete(dataset); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	} else {
		resputil.Success(c, "Deleted successfully")
	}
}
