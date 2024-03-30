package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	imagepackv1 "github.com/raids-lab/crater/pkg/apis/imagepack/v1"
	"github.com/raids-lab/crater/pkg/crclient"
	imagepacksvc "github.com/raids-lab/crater/pkg/db/imagepack"
	quotasvc "github.com/raids-lab/crater/pkg/db/quota"
	tasksvc "github.com/raids-lab/crater/pkg/db/task"
	usersvc "github.com/raids-lab/crater/pkg/db/user"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	payload "github.com/raids-lab/crater/pkg/server/payload"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	"github.com/raids-lab/crater/pkg/util"
	uuid "github.com/satori/go.uuid"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AdminMgr struct {
	quotaService     quotasvc.DBService
	userService      usersvc.DBService
	taskServcie      tasksvc.DBService
	imagepackService imagepacksvc.DBService
	taskController   *aitaskctl.TaskController
	nodeClient       *crclient.NodeClient
	imagepackClient  *crclient.ImagePackController
}

func (mgr *AdminMgr) RegisterRoute(g *gin.RouterGroup) {
	users := g.Group("/users")
	users.GET("", mgr.ListUser)
	users.GET("/:name", mgr.GetUser)
	users.DELETE("/:name", mgr.DeleteUser)
	users.PUT("/:name/role", mgr.UpdateRole)

	quotas := g.Group("/quotas")
	quotas.GET("", mgr.ListQuota)
	quotas.PUT("/:name", mgr.UpdateQuota)

	tasks := g.Group("/tasks")
	tasks.GET("", mgr.ListTaskByTaskType)
	tasks.GET("/stats", mgr.GetTaskStats)

	nodes := g.Group("/nodes")
	nodes.GET("", mgr.ListNode)

	images := g.Group("/images")
	images.POST("/create", mgr.CreateImages)
	images.POST("/delete", mgr.DeleteByID)
	images.GET("/list", mgr.ListImages)
}

func NewAdminMgr(
	taskController *aitaskctl.TaskController, nodeClient *crclient.NodeClient, imagepackClient *crclient.ImagePackController,
) *AdminMgr {
	return &AdminMgr{
		quotaService:     quotasvc.NewDBService(),
		userService:      usersvc.NewDBService(),
		taskServcie:      tasksvc.NewDBService(),
		imagepackService: imagepacksvc.NewDBService(),
		taskController:   taskController,
		nodeClient:       nodeClient,
		imagepackClient:  imagepackClient,
	}
}

func (mgr *AdminMgr) DeleteUser(c *gin.Context) {
	logutils.Log.Infof("User Delete, url: %s", c.Request.URL)
	name := c.Param("name")
	err := mgr.userService.DeleteByUserName(name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete user failed, err %v", err), resputil.NotSpecified)
		return
	}
	// TODO: delete resource
	logutils.Log.Infof("delete user success, username: %s", name)
	resputil.Success(c, "")
}

func (mgr *AdminMgr) ListUser(c *gin.Context) {
	logutils.Log.WithFields(logutils.Fields{
		"url": c.Request.URL,
	}).Info("User List")
	userQuotas, err := mgr.userService.ListAllUserQuotas()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list user failed, err %v", err), resputil.NotSpecified)
		return
	}

	resp := payload.ListUserResp{
		Users: make([]payload.GetUserResp, 0),
	}
	for i := range userQuotas {
		userQuota := &userQuotas[i]
		userResp := payload.GetUserResp{
			UserID:    userQuota.User.ID,
			UserName:  userQuota.User.UserName,
			Role:      userQuota.User.Role,
			CreatedAt: userQuota.User.CreatedAt,
			UpdatedAt: userQuota.User.UpdatedAt,
		}
		resList, err := models.JSONToResourceList(userQuota.Quota.HardQuota)
		if err == nil {
			userResp.QuotaHard = resList
		}
		resp.Users = append(resp.Users, userResp)
	}

	logutils.Log.Infof("list users success, taskNum: %d", len(resp.Users))
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) ListQuota(c *gin.Context) {
	logutils.Log.Infof("Quota List, url: %s", c.Request.URL)
	userQuotas := mgr.taskController.ListQuotaInfoSnapshot()

	resp := payload.ListUserQuotaResp{
		Quotas: make([]payload.GetQuotaResp, 0),
	}
	for i := 0; i < len(userQuotas); i++ {
		quotaInfo := &userQuotas[i]
		r := payload.GetQuotaResp{
			User:     quotaInfo.Name,
			Hard:     quotaInfo.Hard,
			HardUsed: quotaInfo.HardUsed,
			SoftUsed: quotaInfo.SoftUsed,
		}
		resp.Quotas = append(resp.Quotas, r)
	}

	logutils.Log.Infof("list users success, taskNum: %d", len(resp.Quotas))
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) GetUser(c *gin.Context) {
	name := c.Param("name")
	var user *models.User
	var err error
	user, err = mgr.userService.GetByUserName(name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get user failed, err %v", err), resputil.NotSpecified)
		return
	}

	resp := payload.GetUserResp{
		UserID:    user.ID,
		UserName:  user.UserName,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	logutils.Log.Infof("get user success, user: %s", resp.UserName)
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) UpdateQuota(c *gin.Context) {
	logutils.Log.Infof("Quota Update, url: %s", c.Request.URL)
	var req payload.UpdateQuotaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	name := c.Param("name")
	user, err := mgr.userService.GetByUserName(name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get user failed, err %v", err), resputil.NotSpecified)
		return
	}
	quota, err := mgr.quotaService.GetByUserName(name)
	if err != nil {
		quota = &models.Quota{
			// UserID:    user.ID,
			UserName:  user.UserName,
			NameSpace: fmt.Sprintf("user-%s", user.UserName),
		}
	}
	quota.HardQuota = models.ResourceListToJSON(req.HardQuota)
	err = mgr.quotaService.Update(quota)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("update quota failed, err %v", err), resputil.NotSpecified)
		return
	}
	// notify taskController to update quota
	mgr.taskController.AddOrUpdateQuotaInfo(quota.UserName, quota)

	logutils.Log.Infof("update quota success, user: %s, quota:%v", name, req.HardQuota)
	resputil.Success(c, "")
}

func (mgr *AdminMgr) UpdateRole(c *gin.Context) {
	logutils.Log.Infof("Role Update, url: %s", c.Request.URL)
	var req payload.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	name := c.Param("name")
	if req.Role == models.RoleAdmin || req.Role == models.RoleUser {
		// do nothing
	} else {
		resputil.Error(c, fmt.Sprintf("role %s is not valid", req.Role), resputil.NotSpecified)
		return
	}
	err := mgr.userService.UpdateRole(name, req.Role)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("update user role failed, err %v", err), resputil.NotSpecified)
		return
	}
	logutils.Log.Infof("update user role success, user: %s, role: %s", name, req.Role)
	resputil.Success(c, "")
}

func (mgr *AdminMgr) ListTaskByTaskType(c *gin.Context) {
	logutils.Log.Infof("Task List, url: %s", c.Request.URL)
	var req payload.ListTaskByTypeReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate list parameters failed, err %v", err), resputil.NotSpecified)
		return
	}

	taskModels, totalRows, err := mgr.taskServcie.ListByTaskType(req.TaskType, req.PageIndex, req.PageSize)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.ListTaskResp{
		Rows:     taskModels,
		RowCount: totalRows,
	}
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) GetTaskStats(c *gin.Context) {
	logutils.Log.Infof("Task Count Statistic, url: %s", c.Request.URL)
	taskCountList, err := mgr.taskServcie.GetTaskStatusCount()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task count statistic failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.AITaskStatistic{
		TaskCount: taskCountList,
	}
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) ListNode(c *gin.Context) {
	logutils.Log.Infof("Node List, url: %s", c.Request.URL)
	// get all k8s nodes by k8s client
	nodes, err := mgr.nodeClient.ListNodes()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list nodes failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.ListNodeResp{
		Rows: nodes,
	}
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) CreateImages(c *gin.Context) {
	logutils.Log.Infof("ImagePack Create, url: %s", c.Request.URL)
	userContext, _ := util.GetUserFromGinContext(c)
	req := &payload.ImagePackCreateRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	mgr.requestDefaultValue(req)
	mgr.createImagePack(c, req, userContext)
	resputil.Success(c, "")
}

func (mgr *AdminMgr) requestDefaultValue(req *payload.ImagePackCreateRequest) {
	if req.RegistryServer == "" {
		req.RegistryServer = "***REMOVED***"
		req.RegistryUser = "***REMOVED***"
		req.RegistryPass = "***REMOVED***"
		req.RegistryProject = "crater-images"
	}
}

func (mgr *AdminMgr) createImagePack(ctx *gin.Context, req *payload.ImagePackCreateRequest, userContext util.UserContext) {
	imagepackName := fmt.Sprintf("%s-%s-%s", "admin", userContext.UserName, uuid.NewV4().String())
	// create ImagePack CRD
	imagepackCRD := &imagepackv1.ImagePack{
		ObjectMeta: v1.ObjectMeta{
			Name:      imagepackName,
			Namespace: userContext.Namespace,
		},
		Spec: imagepackv1.ImagePackSpec{
			GitRepository:   req.GitRepository,
			AccessToken:     req.AccessToken,
			RegistryServer:  req.RegistryServer,
			RegistryUser:    req.RegistryUser,
			RegistryPass:    req.RegistryPass,
			RegistryProject: req.RegistryProject,
			UserName:        userContext.UserName,
			ImageName:       req.ImageName,
			ImageTag:        req.ImageTag,
		},
	}
	if err := mgr.imagepackClient.CreateImagePack(ctx, imagepackCRD); err != nil {
		logutils.Log.Errorf("create imagepack CRD failed, params: %+v err:%v", imagepackCRD, err)
		return
	}

	// create ImagePack DataBase entity
	imageLink := fmt.Sprintf("%s/%s/%s:%s", req.RegistryServer, req.RegistryProject, req.ImageName, req.ImageTag)

	imagepackEntity := &models.ImagePack{
		ImagePackName: imagepackName,
		ImageLink:     imageLink,
		UserName:      "admin",
		NameSpace:     userContext.Namespace,
		Status:        string(imagepackv1.PackJobInitial),
		NameTag:       fmt.Sprintf("%s:%s", req.ImageName, req.ImageTag),
	}
	if err := mgr.imagepackService.Create(imagepackEntity); err != nil {
		logutils.Log.Errorf("create imagepack entity failed, params: %+v", imagepackEntity)
	}
}

func (mgr *AdminMgr) ListImages(c *gin.Context) {
	listType := c.DefaultQuery("type", "0")
	fmt.Println(listType)

	var imagepacks []models.ImagePack
	var err error
	if listType == "0" {
		imagepacks, err = mgr.imagepackService.ListAdminPersonal()
	} else if listType == "1" {
		imagepacks, err = mgr.imagepackService.ListAdminPublic()
	} else {
		logutils.Log.Errorf("list image type error, err:%v", err)
		resputil.Error(c, "list image type error", resputil.NotSpecified)
		return
	}
	if err != nil {
		logutils.Log.Errorf("admin fetch personal or public imagepack failed, err:%v", err)
		resputil.Error(c, "list image type error", resputil.NotSpecified)
		return
	}
	for i := range imagepacks {
		imagepack := &imagepacks[i]
		if imagepack.Status != string(imagepackv1.PackJobFinished) && imagepack.Status != string(imagepackv1.PackJobFailed) {
			mgr.updateImagePackStatus(c, imagepack, listType)
		}
	}
	resputil.Success(c, imagepacks)
}

func (mgr *AdminMgr) updateImagePackStatus(ctx *gin.Context, imagepack *models.ImagePack, listType string) {
	imagepackCRD, err := mgr.imagepackClient.GetImagePack(ctx, imagepack.ImagePackName, imagepack.NameSpace)
	if err != nil {
		logutils.Log.Errorf("fetch imagepack CRD failed, err:%v", err)
		return
	}
	logutils.Log.Infof("current stage:%s ----- new stage: %s", imagepack.Status, string(imagepackCRD.Status.Stage))
	if err := mgr.imagepackService.UpdateStatusByEntity(imagepack, string(imagepackCRD.Status.Stage)); err != nil {
		logutils.Log.Errorf("save imagepack status failed, err:%v status:%v", err, *imagepack)
	}
	if imagepackCRD.Status.Stage == imagepackv1.PackJobFinished && listType == "1" {
		err := mgr.imagepackClient.DeleteImagePackByEntity(ctx, imagepackCRD)
		if err != nil {
			logutils.Log.Errorf("fetch imagepack CRD failed, err:%v", err)
		}
	}
}

func (mgr *AdminMgr) DeleteByID(c *gin.Context) {
	imagePackDeleteRequest := &payload.ImagePackDeleteByIDRequest{}
	if err := c.ShouldBindJSON(imagePackDeleteRequest); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	imagepackID := imagePackDeleteRequest.ID
	if err := mgr.imagepackService.DeleteByID(imagepackID); err != nil {
		logutils.Log.Errorf("delete imagepack entity failed! err:%v", err)
		resputil.Error(c, "failed to find imagepack or entity", resputil.NotSpecified)
		return
	}
	resputil.Success(c, "")
}
