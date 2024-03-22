package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	recommenddljobapi "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
	"github.com/raids-lab/crater/pkg/crclient"
	usersvc "github.com/raids-lab/crater/pkg/db/user"
	payload "github.com/raids-lab/crater/pkg/server/payload"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	"github.com/raids-lab/crater/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DataSetMgr struct {
	userService   usersvc.DBService
	datasetClient *crclient.DataSetClient
}

func NewDataSetMgr(userSvc usersvc.DBService, client client.Client) *DataSetMgr {
	return &DataSetMgr{
		userService:   userSvc,
		datasetClient: &crclient.DataSetClient{Client: client},
	}
}

func (mgr *DataSetMgr) RegisterRoute(g *gin.RouterGroup) {
	g.GET("/list", mgr.List)
	g.GET("/info", mgr.Get)
}

func (mgr *DataSetMgr) List(c *gin.Context) {
	userContext, _ := util.GetUserFromGinContext(c)
	var datasetList []*recommenddljobapi.DataSet
	var err error
	if datasetList, err = mgr.datasetClient.ListDataSets(c, userContext.Namespace); err != nil {
		resputil.Error(c, fmt.Sprintf("list dataset failed, err:%v", err), 500)
		return
	}
	ret := make(payload.ListDatasetResp, 0, len(datasetList))
	for _, dataset := range datasetList {
		ret = append(ret, payload.GetDatasetResp{
			ObjectMeta: dataset.ObjectMeta,
			Spec: &payload.DatasetSpec{
				PVC:         dataset.Spec.PVC,
				DownloadURL: dataset.Spec.DownloadURL,
				Size:        dataset.Spec.Size,
			},
			Status: &payload.DatasetStaus{
				Phase: string(dataset.Status.Phase),
			},
		})
	}
	resputil.Success(c, ret)
}

func (mgr *DataSetMgr) Get(c *gin.Context) {
	userContext, _ := util.GetUserFromGinContext(c)
	req := &payload.GetDataSetReq{}
	if err := c.ShouldBindQuery(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request query failed, err:%v", err), 500)
		return
	}
	var dataset *recommenddljobapi.DataSet
	var err error
	if dataset, err = mgr.datasetClient.GetDataSet(c, req.Name, userContext.Namespace); err != nil {
		resputil.Error(c, fmt.Sprintf("get dataset failed, err:%v", err), 500)
		return
	}
	ret := &payload.GetDatasetResp{
		ObjectMeta: dataset.ObjectMeta,
		Spec: &payload.DatasetSpec{
			PVC:         dataset.Spec.PVC,
			DownloadURL: dataset.Spec.DownloadURL,
			Size:        dataset.Spec.Size,
		},
		Status: &payload.DatasetStaus{
			Phase: string(dataset.Status.Phase),
		},
	}
	resputil.Success(c, ret)
}
