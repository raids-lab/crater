package handlers

import (
	"fmt"

	recommenddljobapi "github.com/aisystem/ai-protal/pkg/apis/recommenddljob/v1"
	"github.com/aisystem/ai-protal/pkg/crclient"
	usersvc "github.com/aisystem/ai-protal/pkg/db/user"
	payload "github.com/aisystem/ai-protal/pkg/server/payload"
	resputil "github.com/aisystem/ai-protal/pkg/server/response"
	"github.com/gin-gonic/gin"
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
	namespace, _ := c.Get("x-namespace")
	var datasetList []*recommenddljobapi.DataSet
	var err error
	if datasetList, err = mgr.datasetClient.ListDataSets(c, namespace.(string)); err != nil {
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
	namespace, _ := c.Get("x-namespace")
	req := &payload.GetDataSetReq{}
	if err := c.ShouldBindQuery(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request query failed, err:%v", err), 500)
		return
	}
	var dataset *recommenddljobapi.DataSet
	var err error
	if dataset, err = mgr.datasetClient.GetDataSet(c, req.Name, namespace.(string)); err != nil {
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
