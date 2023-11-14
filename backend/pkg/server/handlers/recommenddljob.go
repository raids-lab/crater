package handlers

import (
	"fmt"

	recommenddljobapi "github.com/aisystem/ai-protal/pkg/apis/recommenddljob/v1"
	"github.com/aisystem/ai-protal/pkg/crclient"
	usersvc "github.com/aisystem/ai-protal/pkg/db/user"
	"github.com/aisystem/ai-protal/pkg/models"
	"github.com/aisystem/ai-protal/pkg/server/payload"
	resputil "github.com/aisystem/ai-protal/pkg/server/response"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RecommendDLJobMgr struct {
	userService usersvc.DBService
	jobclient   *crclient.RecommendDLJobController
}

func NewRecommendDLJobMgr(userSvc usersvc.DBService, client client.Client) *RecommendDLJobMgr {
	return &RecommendDLJobMgr{
		userService: userSvc,
		jobclient:   &crclient.RecommendDLJobController{Client: client},
	}
}

func (mgr *RecommendDLJobMgr) RegisterRoute(g *gin.RouterGroup) {
	g.POST("/create", mgr.Create)
	g.POST("/delete", mgr.Delete)
	g.GET("/list", mgr.List)
	g.GET("/info", mgr.GetByName)
	g.GET("/pods", mgr.GetPodsByName)
}

func (mgr *RecommendDLJobMgr) Create(c *gin.Context) {
	userObject, exists := c.Get("x-user-object")
	if !exists {
		resputil.WrapFailedResponse(c, "user not exist", 400)
		return
	}
	user, ok := userObject.(*models.User)
	if !ok {
		resputil.WrapFailedResponse(c, "user object not exist", 400)
		return
	}
	req := &payload.CreateRecommendDLJobReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.WrapFailedResponse(c, fmt.Sprintf("bind request body failed, err:%v", err), 500)
		return
	}
	for i := range req.RelationShips {
		req.RelationShips[i].JobNamespace = user.NameSpace
	}
	job := &recommenddljobapi.RecommendDLJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      req.Name,
			Namespace: user.NameSpace,
		},
		Spec: req.RecommendDLJobSpec,
	}
	if err := mgr.jobclient.CreateRecommendDLJob(c, job); err != nil {
		resputil.WrapFailedResponse(c, fmt.Sprintf("create recommenddljob failed, err:%v", err), 500)
		return
	}
	resputil.WrapSuccessResponse(c, job)
}

func (mgr *RecommendDLJobMgr) List(c *gin.Context) {
	userObject, exists := c.Get("x-user-object")
	if !exists {
		resputil.WrapFailedResponse(c, "user not exist", 400)
		return
	}
	user, ok := userObject.(*models.User)
	if !ok {
		resputil.WrapFailedResponse(c, "user object not exist", 400)
		return
	}
	var jobList []*recommenddljobapi.RecommendDLJob
	var err error
	if jobList, err = mgr.jobclient.ListRecommendDLJob(c, user.NameSpace); err != nil {
		resputil.WrapFailedResponse(c, fmt.Sprintf("list recommenddljob failed, err:%v", err), 500)
		return
	}
	resputil.WrapSuccessResponse(c, jobList)
}

func (mgr *RecommendDLJobMgr) GetByName(c *gin.Context) {
	userObject, exists := c.Get("x-user-object")
	if !exists {
		resputil.WrapFailedResponse(c, "user not exist", 400)
		return
	}
	user, ok := userObject.(*models.User)
	if !ok {
		resputil.WrapFailedResponse(c, "user object not exist", 400)
		return
	}
	req := &payload.GetRecommendDLJobReq{}
	if err := c.ShouldBindQuery(req); err != nil {
		resputil.WrapFailedResponse(c, fmt.Sprintf("bind request query failed, err:%v", err), 500)
		return
	}
	var job *recommenddljobapi.RecommendDLJob
	var err error
	if job, err = mgr.jobclient.GetRecommendDLJob(c, req.Name, user.NameSpace); err != nil {
		resputil.WrapFailedResponse(c, fmt.Sprintf("get recommenddljob failed, err:%v", err), 500)
		return
	}
	resputil.WrapSuccessResponse(c, job)
}

func (mgr *RecommendDLJobMgr) GetPodsByName(c *gin.Context) {
	userObject, exists := c.Get("x-user-object")
	if !exists {
		resputil.WrapFailedResponse(c, "user not exist", 400)
		return
	}
	user, ok := userObject.(*models.User)
	if !ok {
		resputil.WrapFailedResponse(c, "user object not exist", 400)
		return
	}
	req := &payload.GetRecommendDLJobPodListReq{}
	if err := c.ShouldBindQuery(req); err != nil {
		resputil.WrapFailedResponse(c, fmt.Sprintf("bind request query failed, err:%v", err), 500)
		return
	}
	var podList []*corev1.Pod
	var err error
	if podList, err = mgr.jobclient.GetRecommendDLJobPodList(c, req.Name, user.NameSpace); err != nil {
		resputil.WrapFailedResponse(c, fmt.Sprintf("get recommenddljob pods failed, err:%v", err), 500)
		return
	}
	resputil.WrapSuccessResponse(c, podList)
}

func (mgr *RecommendDLJobMgr) Delete(c *gin.Context) {
	userObject, exists := c.Get("x-user-object")
	if !exists {
		resputil.WrapFailedResponse(c, "user not exist", 400)
		return
	}
	user, ok := userObject.(*models.User)
	if !ok {
		resputil.WrapFailedResponse(c, "user object not exist", 400)
		return
	}
	req := &payload.DeleteRecommendDLJobReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.WrapFailedResponse(c, fmt.Sprintf("bind request body failed, err:%v", err), 500)
		return
	}
	if err := mgr.jobclient.DeleteRecommendDLJob(c, req.Name, user.NameSpace); err != nil {
		resputil.WrapFailedResponse(c, fmt.Sprintf("delete recommenddljob failed, err:%v", err), 500)
		return
	}
	resputil.WrapSuccessResponse(c, nil)
}
