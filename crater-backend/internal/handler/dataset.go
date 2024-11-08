package handler

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/logutils"
	"gorm.io/datatypes"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewDatasetMgr)
}

type DatasetMgr struct {
	name string
}

func NewDatasetMgr(_ RegisterConfig) Manager {
	return &DatasetMgr{
		name: "dataset",
	}
}

func (mgr *DatasetMgr) GetName() string { return mgr.name }

func (mgr *DatasetMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *DatasetMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/mydataset", mgr.GetMyDataset)
	g.GET("/:datasetId/usersNotIn", mgr.ListUsersOutOfDataset)
	g.GET("/:datasetId/usersIn", mgr.ListUserOfDataset)
	g.GET("/:datasetId/queuesNotIn", mgr.ListQueuesOutOfDataset)
	g.GET("/:datasetId/queuesIn", mgr.ListQueueOfDataset)
	g.POST("/create", mgr.CreateDataset)
	g.DELETE("/delete/:id", mgr.DeleteDataset)
	g.POST("/share/user", mgr.ShareDatasetWithUser)
	g.POST("/share/queue", mgr.ShareDatasetWithQueue)
	g.POST("/cancelshare/user", mgr.CancelShareDatasetWithUser)
	g.POST("/cancelshare/queue", mgr.CancelShareDatasetWithQueue)
	g.POST("/rename", mgr.RemaneDatset)
	g.GET("/detail/:datasetId", mgr.GetDatasetByID)
}

func (mgr *DatasetMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/alldataset", mgr.GetAllDataset)
	g.POST("/share/user", mgr.AdminShareDatasetWithUser)
	g.POST("/share/queue", mgr.AdminShareDatasetWithQueue)
	g.POST("/cancelshare/user", mgr.AdminCancelShareDatasetWithUser)
	g.POST("/cancelshare/queue", mgr.AdmincancelShareDatasetWithQueue)
}

type DatasetResp struct {
	Name      string    `json:"name"`
	ID        uint      `json:"id"`
	UserName  string    `json:"username"`
	URL       string    `json:"url"`
	Describe  string    `json:"describe"`
	CreatedAt time.Time `json:"createdAt"`
}

// GetMyDataset godoc
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
func (mgr *DatasetMgr) GetMyDataset(c *gin.Context) {
	token := util.GetToken(c)
	datasets := make(map[uint]DatasetResp)
	ud := query.UserDataset
	d := query.Dataset
	userDataset, err := ud.WithContext(c).Where(ud.UserID.Eq(token.UserID)).Find()
	if err != nil {
		logutils.Log.Infof("Can't get , err: %v", err)
		resputil.Error(c, "Can't get mydatasets", resputil.NotSpecified)
		return
	}
	for i := range userDataset {
		dataset, uderr := d.WithContext(c).Where(d.ID.Eq(userDataset[i].DatasetID)).First()
		if uderr != nil {
			resputil.Error(c, fmt.Sprintf("Get user's dataset failed, err %v", uderr), resputil.NotSpecified)
			return
		}
		tmp, terr := mgr.generateDataseResponse(c, dataset)
		if terr == nil {
			datasets[dataset.ID] = tmp
		}
	}
	err = mgr.generateQueueDataseResponse(c, 1, datasets)
	if err != nil {
		resputil.Error(c, "Can't get public datasets", resputil.NotSpecified)
		return
	}
	if token.QueueID != 0 && token.QueueID != 1 {
		err = mgr.generateQueueDataseResponse(c, token.QueueID, datasets)
		if err != nil {
			resputil.Error(c, "Can't get queue datasets", resputil.NotSpecified)
			return
		}
	}
	result := make([]DatasetResp, 0, len(datasets))
	for _, data := range datasets {
		result = append(result, data)
	}
	resputil.Success(c, result)
}

// 函数名称 GetDatasetByID
// @Summary 通过数据集id获取数据集信息
// @Description 通过数据集id获取数据集信息
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param req path DatasetGetReq true "数据集ID"
// @Success 200 {object} resputil.Response[any] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/detail/{datasetId} [get]
func (mgr *DatasetMgr) GetDatasetByID(c *gin.Context) {
	var req DatasetGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("get dataset failed, detail: %v", err))
		return
	}
	d := query.Dataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(req.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "this dataset not exist", resputil.InvalidRequest)
		return
	}
	res, err := mgr.generateDataseResponse(c, dataset)
	if err != nil {
		resputil.Error(c, "Can't get this dataset", resputil.NotSpecified)
		return
	}
	var result []DatasetResp
	result = append(result, res)
	resputil.Success(c, result)
}

// GetAllDataset godoc
// @Summary 获取所有数据集
// @Description 获取所有数据集
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/dataset/alldataset [get]
func (mgr *DatasetMgr) GetAllDataset(c *gin.Context) {
	datasets := make(map[uint]DatasetResp)
	d := query.Dataset
	dataset, err := d.WithContext(c).Where(d.ID.IsNotNull()).Find()
	if err != nil {
		resputil.Error(c, "Can't get datasets", resputil.NotSpecified)
		return
	}
	for i := range dataset {
		tmp, terr := mgr.generateDataseResponse(c, dataset[i])
		if terr == nil {
			datasets[tmp.ID] = tmp
		}
	}
	result := make([]DatasetResp, 0, len(datasets))
	for _, data := range datasets {
		result = append(result, data)
	}
	resputil.Success(c, result)
}

func (mgr *DatasetMgr) generateQueueDataseResponse(c *gin.Context, queueid uint, data map[uint]DatasetResp) error {
	d := query.Dataset
	qd := query.AccountDataset
	queueDataset, err := qd.WithContext(c).Where(qd.AccountID.Eq(queueid)).Find()
	if err != nil {
		return err
	}
	for i := range queueDataset {
		dataset, err := d.WithContext(c).Where(d.ID.Eq(queueDataset[i].DatasetID)).First()
		if err != nil {
			return err
		}
		tmp, terr := mgr.generateDataseResponse(c, dataset)
		if terr != nil {
			return terr
		}
		data[tmp.ID] = tmp
	}
	return nil
}

func (mgr *DatasetMgr) generateDataseResponse(c *gin.Context, dataset *model.Dataset) (DatasetResp, error) {
	u := query.User
	user, uerr := u.WithContext(c).Where(u.ID.Eq(dataset.UserID)).First()
	if uerr != nil {
		resputil.Error(c, fmt.Sprintf("Dataset has no creator, err %v", uerr), resputil.NotSpecified)
		return DatasetResp{}, uerr
	}
	return DatasetResp{
		Name:      dataset.Name,
		Describe:  dataset.Describe,
		URL:       dataset.URL,
		UserName:  user.Nickname,
		ID:        dataset.ID,
		CreatedAt: dataset.CreatedAt,
	}, nil
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
func (mgr *DatasetMgr) CreateDataset(c *gin.Context) {
	token := util.GetToken(c)
	var datasetReq DatasetReq
	if err := c.ShouldBindJSON(&datasetReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	regex := regexp.MustCompile("^/+")
	url := regex.ReplaceAllString(datasetReq.URL, "")
	db := query.Use(query.GetDB())
	err := db.Transaction(func(tx *query.Query) error {
		d := tx.Dataset
		ud := tx.UserDataset
		var dataset model.Dataset
		dataset.Name = datasetReq.Name
		dataset.Describe = datasetReq.Describe
		dataset.URL = url
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
	DatasetID uint   `json:"datasetID" binding:"required"`
	UserIDs   []uint `json:"userIDs" binding:"required"`
}
type cancelsharedUserReq struct {
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
func (mgr *DatasetMgr) ShareDatasetWithUser(c *gin.Context) {
	token := util.GetToken(c)
	var userReq SharedUserReq
	if err := c.ShouldBindJSON(&userReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	d := query.Dataset
	_, err := d.WithContext(c).Where(d.UserID.Eq(token.UserID), d.ID.Eq(userReq.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "you has no permission or this dataset not exist", resputil.InvalidRequest)
		return
	}
	err = mgr.shareWithUser(c, userReq)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}
	resputil.Success(c, "Shared successfully")
}

// AdminShareDatasetWithUser godoc
// @Summary 管理员对用户分享数据集
// @Description 管理员对用户分享数据集
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param userReq body SharedUserReq true "共享数据集用户"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/dataset/share/user [post]
func (mgr *DatasetMgr) AdminShareDatasetWithUser(c *gin.Context) {
	var userReq SharedUserReq
	if err := c.ShouldBindJSON(&userReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	d := query.Dataset
	_, err := d.WithContext(c).Where(d.ID.Eq(userReq.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "this dataset not exist", resputil.InvalidRequest)
		return
	}
	err = mgr.shareWithUser(c, userReq)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}
	resputil.Success(c, "Shared with user successfully")
}

func (mgr *DatasetMgr) shareWithUser(c *gin.Context, userReq SharedUserReq) error {
	u := query.User
	if len(userReq.UserIDs) == 0 {
		return fmt.Errorf("need to choose users to share")
	}
	for _, uid := range userReq.UserIDs {
		_, err := u.WithContext(c).Where(u.ID.Eq(uid)).First()
		if err != nil {
			return fmt.Errorf("user not exist")
		}
		ud := query.UserDataset
		hud, _ := ud.WithContext(c).Where(ud.UserID.Eq(uid), ud.DatasetID.Eq(userReq.DatasetID)).First()
		if hud != nil {
			return fmt.Errorf("user has shared dataset")
		}
		userDataset := model.UserDataset{
			UserID:    uid,
			DatasetID: userReq.DatasetID,
		}
		if err := ud.WithContext(c).Create(&userDataset); err != nil {
			return err
		}
	}
	return nil
}

// 函数名称 CancelShareDatasetWithUser
// @Summary 普通用户取消数据集共享用户
// @Description 普通用户取消数据集共享
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param Req body cancelsharedUserReq true "共享数据集用户"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/cancelshare/user [post]
func (mgr *DatasetMgr) CancelShareDatasetWithUser(c *gin.Context) {
	var Req cancelsharedUserReq
	if err := c.ShouldBindJSON(&Req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	d := query.Dataset
	token := util.GetToken(c)
	if _, err := d.WithContext(c).Where(d.UserID.Eq(token.UserID), d.ID.Eq(Req.DatasetID)).First(); err != nil {
		resputil.Error(c, "you has no permission to cancel share with user", resputil.NotSpecified)
		return
	}
	if err := mgr.cancelShareWithUser(c, Req); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "cancel share with user successfully")
}

// 函数名称 AdminCancelShareDatasetWithUser
// @Summary 管理员取消数据集共享用户
// @Description 管理员取消数据集共享用户
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param Req body cancelsharedUserReq true "共享数据集用户"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/dataset/cancelshare/user [post]
func (mgr *DatasetMgr) AdminCancelShareDatasetWithUser(c *gin.Context) {
	var Req cancelsharedUserReq
	if err := c.ShouldBindJSON(&Req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	d := query.Dataset
	if _, err := d.WithContext(c).Where(d.ID.Eq(Req.DatasetID)).First(); err != nil {
		resputil.Error(c, "this dataset not exist", resputil.NotSpecified)
		return
	}
	if err := mgr.cancelShareWithUser(c, Req); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "Shared with user successfully")
}
func (mgr *DatasetMgr) cancelShareWithUser(c *gin.Context, userReq cancelsharedUserReq) error {
	ud := query.UserDataset
	u := query.User
	if _, err := u.WithContext(c).Where(u.ID.Eq(userReq.UserID)).First(); err != nil {
		return fmt.Errorf("can't find user")
	}
	d := query.Q.Dataset
	if _, err := d.WithContext(c).Where(d.ID.Eq(userReq.DatasetID), d.UserID.Eq(userReq.UserID)).First(); err == nil {
		return fmt.Errorf("can't cancel share with creator")
	}
	uud, _ := ud.WithContext(c).Where(ud.UserID.Eq(userReq.UserID), ud.DatasetID.Eq(userReq.DatasetID)).First()
	if uud == nil {
		return fmt.Errorf("user doesn't shared dataset")
	}
	if _, err := ud.WithContext(c).Delete(uud); err != nil {
		return err
	}
	return nil
}

type SharedQueueReq struct {
	DatasetID uint   `json:"datasetID" binding:"required"`
	QueueIDs  []uint `json:"queueIDs" binding:"required"`
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
func (mgr *DatasetMgr) ShareDatasetWithQueue(c *gin.Context) {
	var queueReq SharedQueueReq
	if err := c.ShouldBindJSON(&queueReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	token := util.GetToken(c)
	d := query.Dataset
	_, err := d.WithContext(c).Where(d.UserID.Eq(token.UserID), d.ID.Eq(queueReq.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "you has no permission or this dataset not exist", resputil.InvalidRequest)
		return
	}
	err = mgr.shareWithQueue(c, queueReq)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}
	resputil.Success(c, "Shared successfully")
}

// AdminShareDatasetWithQueue godoc
// @Summary 管理员对队列分享数据集
// @Description 管理员对队列分享数据集
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param queueReq body SharedQueueReq true "共享数据集队列"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/dataset/share/queue [post]
func (mgr *DatasetMgr) AdminShareDatasetWithQueue(c *gin.Context) {
	var queueReq SharedQueueReq
	d := query.Dataset
	err := c.ShouldBindJSON(&queueReq)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	if _, err = d.WithContext(c).Where(d.ID.Eq(queueReq.DatasetID)).First(); err != nil {
		resputil.Error(c, "can't find this dataset", resputil.InvalidRequest)
		return
	}
	err = mgr.shareWithQueue(c, queueReq)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}
	resputil.Success(c, "Shared with queue successfully")
}

func (mgr *DatasetMgr) shareWithQueue(c *gin.Context, queueReq SharedQueueReq) error {
	q := query.Account
	qd := query.AccountDataset
	if len(queueReq.QueueIDs) == 0 {
		return fmt.Errorf("need to choose queues to share")
	}
	for _, QueueID := range queueReq.QueueIDs {
		_, err := q.WithContext(c).Where(q.ID.Eq(QueueID)).First()
		if err != nil {
			return fmt.Errorf("queue not exist")
		}
		hqd, _ := qd.WithContext(c).Where(qd.AccountID.Eq(QueueID), qd.DatasetID.Eq(queueReq.DatasetID)).First()
		if hqd != nil {
			return fmt.Errorf("queue has shared dataset")
		}
		queuedataset := model.AccountDataset{
			AccountID: QueueID,
			DatasetID: queueReq.DatasetID,
		}
		if err := qd.WithContext(c).Create(&queuedataset); err != nil {
			return err
		}
	}
	return nil
}

type cancelSharedQueueReq struct {
	DatasetID uint `json:"datasetID" binding:"required"`
	QueueID   uint `json:"queueID" binding:"required"`
}

// 函数名称 CancelShareDatasetWithQueue
// @Summary 普通用户取消数据共享队列
// @Description 普通用户取消数据共享队列
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param queueReq body cancelSharedQueueReq true "共享数据集队列"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router  /v1/dataset/cancelshare/queue [post]
func (mgr *DatasetMgr) CancelShareDatasetWithQueue(c *gin.Context) {
	var queueReq cancelSharedQueueReq
	if err := c.ShouldBindJSON(&queueReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	d := query.Dataset
	token := util.GetToken(c)
	_, err := d.WithContext(c).Where(d.UserID.Eq(token.UserID), d.ID.Eq(queueReq.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "you can't cancel share with queue", resputil.InvalidRequest)
		return
	}
	err = mgr.cancelShareWithQueue(c, queueReq)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}
	resputil.Success(c, "cancel successfully")
}

// 函数名称 AdmincancelShareDatasetWithQueue
// @Summary 管理员取消数据共享队列
// @Description 管理员取消数据共享队列
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param queueReq body cancelSharedQueueReq true "共享数据集队列"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/dataset/cancelshare/queue [post]
func (mgr *DatasetMgr) AdmincancelShareDatasetWithQueue(c *gin.Context) {
	var queueReq cancelSharedQueueReq
	err := c.ShouldBindJSON(&queueReq)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	d := query.Dataset
	_, err = d.WithContext(c).Where(d.ID.Eq(queueReq.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "dataset does not exist", resputil.InvalidRequest)
		return
	}
	err = mgr.cancelShareWithQueue(c, queueReq)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}
	resputil.Success(c, "cancel successfully")
}

func (mgr *DatasetMgr) cancelShareWithQueue(c *gin.Context, cancelQueueReq cancelSharedQueueReq) error {
	q := query.Account
	qd := query.AccountDataset
	_, err := q.WithContext(c).Where(q.ID.Eq(cancelQueueReq.QueueID)).First()
	if err != nil {
		return fmt.Errorf("queue not exist")
	}
	hqd, _ := qd.WithContext(c).Where(qd.AccountID.Eq(cancelQueueReq.QueueID), qd.DatasetID.Eq(cancelQueueReq.DatasetID)).First()
	if hqd == nil {
		return fmt.Errorf("the dataset was not shared with the queue")
	}
	if _, err := qd.WithContext(c).Delete(hqd); err != nil {
		return err
	}
	return nil
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
func (mgr *DatasetMgr) DeleteDataset(c *gin.Context) {
	token := util.GetToken(c)
	var req DeleteDatasetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("validate delete parameters failed, detail: %v", err))
		return
	}
	db := query.Use(query.GetDB())
	err := db.Transaction(func(tx *query.Query) error {
		d := tx.Dataset
		ud := tx.UserDataset
		qd := tx.AccountDataset
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
		if len(userDataset) > 0 {
			if _, err := ud.WithContext(c).Delete(userDataset...); err != nil {
				return err
			}
		}

		if len(queueDataset) > 0 {
			if _, err := qd.WithContext(c).Delete(queueDataset...); err != nil {
				return err
			}
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

type RenameReq struct {
	DatasetID uint   `json:"datasetID" binding:"required"`
	Name      string `json:"name" binding:"required"`
}

// RemaneDatset godoc
// @Summary 数据集重命名
// @Description 数据集重命名
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param req body RenameReq true "参数描述"
// @Success 200 {object} resputil.Response[string] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/rename [post]
func (mgr *DatasetMgr) RemaneDatset(c *gin.Context) {
	token := util.GetToken(c)
	var req RenameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	d := query.Dataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(req.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "this dataset not exist", resputil.InvalidRequest)
		return
	}
	if dataset.UserID != token.UserID && token.RolePlatform != model.RoleAdmin {
		resputil.Error(c, "you has no permission to rename", resputil.InvalidRequest)
		return
	}
	_, err = d.WithContext(c).Where(d.ID.Eq(req.DatasetID)).Update(d.Name, req.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to rename dataset: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "Successfully rename dataset")
}

type DatasetGetReq struct {
	DatasetID uint `uri:"datasetId" binding:"required"`
}

type UserDatasetGetResp struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// ListUsersOutOfDataset godoc
// @Summary 没有该数据集权限的用户列表
// @Description 没有该数据集权限的用户列表
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param req path DatasetGetReq true "数据集ID"
// @Success 200 {object} resputil.Response[any] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/{datasetId}/usersNotIn [get]
func (mgr *DatasetMgr) ListUsersOutOfDataset(c *gin.Context) {
	var req DatasetGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("get users out of dataset failed, detail: %v", err))
		return
	}
	d := query.Dataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(req.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "this dataset not exist", resputil.InvalidRequest)
		return
	}
	u := query.User
	ud := query.UserDataset
	var uids []uint
	if err := ud.WithContext(c).Select(ud.UserID).Where(ud.DatasetID.Eq(dataset.ID)).Scan(&uids); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to scan user IDs: %v", err), resputil.NotSpecified)
		return
	}
	var attributes []model.UserAttributes
	if err := u.WithContext(c).Where(u.Status.Eq(uint8(model.StatusActive))).
		Where(u.ID.NotIn(uids...)).Distinct().Select(u.Attributes).Scan(&attributes); err != nil {
		resputil.Error(c, fmt.Sprintf("Get UserDataset failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	resp := make([]model.UserAttribute, len(attributes))
	for i := range attributes {
		resp[i] = attributes[i].Attributes.Data()
	}
	resputil.Success(c, resp)
}

type UserOfDatasetResp struct {
	ID       uint                                    `json:"id"`
	Name     string                                  `json:"name"`
	IsOwner  bool                                    `json:"isowner"`
	UserInfo datatypes.JSONType[model.UserAttribute] `json:"userInfo"`
}

// 函数名称 ListUserOfDataset
// @Summary 获取该数据集共享用户
// @Description 获取该数据集共享用户
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param req path DatasetGetReq true "数据集ID"
// @Success 200 {object} resputil.Response[any] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/{datasetId}/usersIn [get]
func (mgr *DatasetMgr) ListUserOfDataset(c *gin.Context) {
	var req DatasetGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("get dataset failed, detail: %v", err))
		return
	}
	d := query.Dataset
	u := query.User
	ud := query.UserDataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(req.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "this dataset not exist", resputil.InvalidRequest)
		return
	}
	var uids []uint
	if err := ud.WithContext(c).Select(ud.UserID).Where(ud.DatasetID.Eq(dataset.ID)).Distinct().Scan(&uids); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to scan user IDs: %v", err), resputil.NotSpecified)
		return
	}

	var resp []UserOfDatasetResp
	for i := range uids {
		user, err := u.WithContext(c).Where(u.ID.Eq(uids[i])).First()
		if err != nil {
			resputil.Error(c, fmt.Sprintf("Can't find user :%v", err), resputil.InvalidRequest)
		}
		res := UserOfDatasetResp{
			ID:       uids[i],
			IsOwner:  uids[i] == dataset.UserID,
			Name:     user.Name,
			UserInfo: user.Attributes,
		}
		resp = append(resp, res)
	}

	resputil.Success(c, resp)
}

type QueueDatasetGetResp struct {
	ID         uint                                    `json:"id"`
	Nickname   string                                  `json:"name"`
	Attributes datatypes.JSONType[model.UserAttribute] `json:"attributes"`
}

// ListQueuesOutOfDataset godoc
// @Summary 没有该数据集权限的队列列表
// @Description 没有该数据集权限的队列列表
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param req path DatasetGetReq true "数据集ID"
// @Success 200 {object} resputil.Response[any] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/{datasetId}/queuesNotIn [get]
func (mgr *DatasetMgr) ListQueuesOutOfDataset(c *gin.Context) {
	var req DatasetGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("get queues out of dataset failed, detail: %v", err))
		return
	}
	d := query.Dataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(req.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "this dataset not exist", resputil.InvalidRequest)
		return
	}
	q := query.Account
	qd := query.AccountDataset
	var qids []uint
	if err := qd.WithContext(c).Select(qd.AccountID).Where(qd.DatasetID.Eq(dataset.ID)).Scan(&qids); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to scan queue IDs: %v", err), resputil.NotSpecified)
		return
	}
	var resp []QueueDatasetGetResp
	exec := q.WithContext(c).Where(q.ID.NotIn(qids...)).Distinct()
	if err := exec.Scan(&resp); err != nil {
		resputil.Error(c, fmt.Sprintf("Get QueueDataset failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, resp)
}

// 函数名称 ListQueueOfDataset
// @Summary 数据集的共享队列
// @Description 数据集的共享队列
// @Tags Dataset
// @Accept json
// @Produce json
// @Security Bearer
// @Param req path DatasetGetReq true "数据集ID"
// @Success 200 {object} resputil.Response[any] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/dataset/{datasetId}/queuesIn [get]
func (mgr *DatasetMgr) ListQueueOfDataset(c *gin.Context) {
	var req DatasetGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("get queues of dataset failed, detail: %v", err))
		return
	}
	d := query.Dataset
	q := query.Account
	qd := query.AccountDataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(req.DatasetID)).First()
	if err != nil {
		resputil.Error(c, "this dataset not exist", resputil.InvalidRequest)
		return
	}
	var qids []uint
	if err := qd.WithContext(c).Select(qd.AccountID).Where(qd.DatasetID.Eq(dataset.ID)).Distinct().Scan(&qids); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to scan queue IDs: %v", err), resputil.NotSpecified)
		return
	}
	var resp []QueueDatasetGetResp
	exec := q.WithContext(c).Where(q.ID.In(qids...)).Distinct()
	if err := exec.Scan(&resp); err != nil {
		resputil.Error(c, fmt.Sprintf("Get QueueDataset failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, resp)
}
