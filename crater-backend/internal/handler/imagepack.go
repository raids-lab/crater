package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	imagepackv1 "github.com/raids-lab/crater/pkg/apis/imagepack/v1"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/logutils"
	payload "github.com/raids-lab/crater/pkg/server/payload"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	ImagePackCreateRequest struct {
		GitRepository   string              `json:"gitRepository"`
		AccessToken     string              `json:"accessToken"`
		RegistryServer  string              `json:"registryServer"`
		RegistryUser    string              `json:"registryUser"`
		RegistryPass    string              `json:"registryPass"`
		RegistryProject string              `json:"registryProject"`
		ImageName       string              `json:"imageName"`
		ImageTag        string              `json:"imageTag"`
		NeedProfile     bool                `json:"needProfile"`
		TaskType        model.ImageTaskType `json:"taskType"`
		Alias           string              `json:"alias"`
		Description     string              `json:"description"`
	}

	ImagePackDeleteByIDRequest struct {
		ID uint `json:"id"`
	}

	ImagePackParamsUpdateRequest struct {
		ImagePackName string                   `json:"imagepackname"`
		Data          model.ImageProfileParams `json:"data"`
	}
	ImagePackAdminListRequest struct {
		Type int `form:"type"`
	}

	ImagePackLogRequest struct {
		ImagePackName string `form:"imagepackname"`
	}
)

type (
	ImagePackListResponse struct {
		ID            uint                     `json:"ID"`
		ImageLink     string                   `json:"imagelink"`
		Status        string                   `json:"status"`
		CreatedAt     time.Time                `json:"createdAt"`
		NameTag       string                   `json:"nametag"`
		CreaterName   string                   `json:"creatername"`
		ImagePackName string                   `json:"imagepackname"`
		Params        model.ImageProfileParams `json:"params"`
	}
)

const (
	UserNameSpace = "crater-jobs"
	AdminUserName = "admin"
	PublicQueueID = 0

	ImagePackInitial  = string(imagepackv1.PackJobInitial)
	ImagePackPending  = string(imagepackv1.PackJobPending)
	ImagePackRunning  = string(imagepackv1.PackJobRunning)
	ImagePackFinished = string(imagepackv1.PackJobFinished)
	ImagePackFailed   = string(imagepackv1.PackJobFailed)
	ImagePackDeleted  = "Deleted"
)

type ImagePackMgr struct {
	logClient       *crclient.LogClient
	imagepackClient *crclient.ImagePackController
	harborClient    *crclient.HarborClient
}

func NewImagePackMgr(
	logClient *crclient.LogClient,
	imagepackClient *crclient.ImagePackController,
	harborClient *crclient.HarborClient,
) *ImagePackMgr {
	return &ImagePackMgr{
		logClient:       logClient,
		imagepackClient: imagepackClient,
		harborClient:    harborClient,
	}
}

func (mgr *ImagePackMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/list", mgr.UserListAll)
	g.POST("/create", mgr.UserCreate)
	g.POST("/delete", mgr.DeleteByID)
	g.GET("/available", mgr.ListAvailableImages)
	g.POST("/params", mgr.UpdateParams)
	g.POST("/get", mgr.GetImagePackByName)
	g.GET("/log", mgr.GetImagePackLogByName)
}

func (mgr *ImagePackMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/list", mgr.AdminListAll)
	g.POST("/create", mgr.AdminCreate)
	g.POST("/delete", mgr.DeleteByID)
	g.GET("/log", mgr.GetImagePackLogByName)
}

// UserCreate godoc
// @Summary 创建ImagePack CRD和数据库entity
// @Description 获取参数，生成变量，调用接口
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param data body ImagePackCreateRequest true "创建ImagePack"
// @Router /images/create [post]
func (mgr *ImagePackMgr) UserCreate(c *gin.Context) {
	req := &ImagePackCreateRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	logutils.Log.Infof("create params: %+v", req)
	mgr.requestDefaultValue(req)
	mgr.createImagePack(c, req, token, token.QueueID)
	resputil.Success(c, "")
}

// AdminCreate godoc
// @Summary 创建ImagePack CRD和数据库entity
// @Description 获取参数，生成变量，调用接口
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param data body ImagePackCreateRequest true "创建ImagePack"
// @Router /images/create [post]
func (mgr *ImagePackMgr) AdminCreate(c *gin.Context) {
	req := &ImagePackCreateRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	logutils.Log.Infof("create params: %+v", req)
	mgr.requestDefaultValue(req)
	mgr.createImagePack(c, req, token, 1)
	resputil.Success(c, "")
}

func (mgr *ImagePackMgr) requestDefaultValue(req *ImagePackCreateRequest) {
	if req.RegistryServer == "" {
		req.RegistryServer = mgr.harborClient.RegistryServer
		req.RegistryUser = mgr.harborClient.RegistryUser
		req.RegistryPass = mgr.harborClient.RegistryPass
		req.RegistryProject = mgr.harborClient.RegistryProject
	}
}

func (mgr *ImagePackMgr) createImagePack(c *gin.Context, req *ImagePackCreateRequest, token util.JWTMessage, queueID uint) {
	userQuery := query.User
	user, err := userQuery.WithContext(c).Where(userQuery.ID.Eq(token.UserID)).First()
	if err != nil {
		logutils.Log.Errorf("fetch user failed, params: %+v err:%v", req, err)
		return
	}
	queueQuery := query.Queue
	queue, err := queueQuery.WithContext(c).Where(queueQuery.ID.Eq(queueID)).First()
	if err != nil {
		logutils.Log.Errorf("fetch project failed, params: %+v err:%v", req, err)
		return
	}
	imagepackQuery := query.Image
	imagepackName := fmt.Sprintf("%s-%s", user.Name, uuid.New().String())
	// create ImagePack CRD
	imagepackCRD := &imagepackv1.ImagePack{
		ObjectMeta: v1.ObjectMeta{
			Name:      imagepackName,
			Namespace: UserNameSpace,
		},
		Spec: imagepackv1.ImagePackSpec{
			GitRepository:   req.GitRepository,
			AccessToken:     req.AccessToken,
			RegistryServer:  req.RegistryServer,
			RegistryUser:    req.RegistryUser,
			RegistryPass:    req.RegistryPass,
			RegistryProject: req.RegistryProject,
			UserName:        user.Name,
			ImageName:       req.ImageName,
			ImageTag:        req.ImageTag,
			NeedProfile:     req.NeedProfile,
		},
	}
	if err := mgr.imagepackClient.CreateImagePack(c, imagepackCRD); err != nil {
		logutils.Log.Errorf("create imagepack CRD failed, params: %+v err:%v", imagepackCRD, err)
		return
	}
	imageLink := fmt.Sprintf("%s/%s/%s:%s", req.RegistryServer, req.RegistryProject, req.ImageName, req.ImageTag)
	imagepackEntity := &model.Image{
		UserID:        token.UserID,
		User:          *user,
		QueueID:       queueID,
		Queue:         *queue,
		ImagePackName: imagepackName,
		ImageLink:     imageLink,
		NameSpace:     UserNameSpace,
		Status:        ImagePackInitial,
		NameTag:       fmt.Sprintf("%s:%s", req.ImageName, req.ImageTag),
		Params:        model.ImageProfileParams{},
		NeedProfile:   req.NeedProfile,
		TaskType:      req.TaskType,
		Alias:         req.Alias,
		Description:   req.Description,
		CreatorName:   user.Name,
	}

	if err := imagepackQuery.WithContext(c).Create(imagepackEntity); err != nil {
		logutils.Log.Errorf("create imagepack entity failed, params: %+v", imagepackEntity)
	}
}

// UserListAll godoc
// @Summary 用户和管理员获取相关镜像的功能
// @Description 返回该用户所有的镜像
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param type query int true "管理员获取镜像的类型"
// @Router /images/list [GET]
func (mgr *ImagePackMgr) UserListAll(c *gin.Context) {
	imagepackQuery := query.Image
	var imagepacks []*model.Image
	var err error
	token := util.GetToken(c)
	imagepacks, err = imagepackQuery.WithContext(c).
		Where(imagepackQuery.QueueID.Eq(token.QueueID)).
		Where(imagepackQuery.Status.Neq(ImagePackDeleted)).
		Find()
	if err != nil {
		logutils.Log.Errorf("fetch imagepacks entity failed, err:%v", err)
	}
	responses := []ImagePackListResponse{}
	for i := range imagepacks {
		imagepack := imagepacks[i]
		if imagepack.Status != ImagePackFinished && imagepack.Status != ImagePackFailed {
			mgr.updateImagePackStatus(c, imagepack)
		}
		responses = append(responses, mgr.generateImageListResponse(imagepack))
	}
	resputil.Success(c, responses)
}

// AdminListAll godoc
// @Summary 管理员获取相关镜像的功能
// @Description 根据GET参数type来决定搜索私有or公共镜像
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param type query int true "管理员获取镜像的类型"
// @Router /images/list [GET]
func (mgr *ImagePackMgr) AdminListAll(c *gin.Context) {
	imagepackQuery := query.Image
	var imagepacks []*model.Image
	var err error
	var req ImagePackAdminListRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate list parameters failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	listType := req.Type
	if listType == 0 {
		imagepacks, err = imagepackQuery.WithContext(c).
			Where(imagepackQuery.QueueID.Neq(PublicQueueID)).
			Where(imagepackQuery.Status.Neq(ImagePackDeleted)).
			Find()
	} else if listType == 1 {
		imagepacks, err = imagepackQuery.WithContext(c).
			Where(imagepackQuery.QueueID.Eq(PublicQueueID)).
			Where(imagepackQuery.Status.Neq(ImagePackDeleted)).
			Find()
	} else {
		logutils.Log.Errorf("admin list image type error, err:%v", err)
		resputil.Error(c, "admin list image type error", resputil.NotSpecified)
		return
	}
	if err != nil {
		logutils.Log.Errorf("admin fetch personal or public imagepack failed, err:%v", err)
		resputil.Error(c, "list image type error", resputil.NotSpecified)
		return
	}
	responses := []ImagePackListResponse{}
	for i := range imagepacks {
		imagepack := imagepacks[i]
		if imagepack.Status != ImagePackFinished && imagepack.Status != ImagePackFailed {
			mgr.updateImagePackStatus(c, imagepack)
		}
		responses = append(responses, mgr.generateImageListResponse(imagepack))
	}
	resputil.Success(c, responses)
}

func (mgr *ImagePackMgr) generateImageListResponse(imagepack *model.Image) ImagePackListResponse {
	return ImagePackListResponse{
		ID:            imagepack.ID,
		ImageLink:     imagepack.ImageLink,
		Status:        imagepack.Status,
		CreatedAt:     imagepack.CreatedAt,
		NameTag:       imagepack.NameTag,
		CreaterName:   imagepack.User.Name,
		ImagePackName: imagepack.ImagePackName,
		Params:        imagepack.Params,
	}
}

func (mgr *ImagePackMgr) updateImagePackStatus(c *gin.Context, imagepack *model.Image) {
	imagepackQuery := query.Image
	imagepackCRD, err := mgr.imagepackClient.GetImagePack(c, imagepack.ImagePackName, UserNameSpace)
	if err != nil {
		logutils.Log.Errorf("fetch imagepack CRD failed, err:%v", err)
		return
	}
	logutils.Log.Infof("current stage:%s ----- new stage: %s", imagepack.Status, string(imagepackCRD.Status.Stage))

	if _, err := imagepackQuery.WithContext(c).
		Where(imagepackQuery.ID.Eq(imagepack.ID)).
		Update(imagepackQuery.Status, string(imagepackCRD.Status.Stage)); err != nil {
		logutils.Log.Errorf("save imagepack status failed, err:%v status:%v", err, *imagepack)
	}
	resputil.Success(c, "")
}

// ListAvailableImages godoc
// @Summary 用户在运行作业时选择镜像需要调用此接口，来获取可以用的镜像
// @Description 用queueID来过滤已完成的镜像
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /images/available [GET]
func (mgr *ImagePackMgr) ListAvailableImages(c *gin.Context) {
	token := util.GetToken(c)
	imagepackQuery := query.Image
	imagepacks, err := imagepackQuery.WithContext(c).
		Where(imagepackQuery.QueueID.Eq(token.QueueID)).
		Where(imagepackQuery.Status.Eq(ImagePackFinished)).
		Find()
	if err != nil {
		logutils.Log.Errorf("fetch available imagepack failed, err:%v", err)
		resputil.Error(c, "fetch available imagepack failed", resputil.NotSpecified)
		return
	}
	imageLinks := make([]string, len(imagepacks))
	for i := range imagepacks {
		imageLinks[i] = imagepacks[i].ImageLink
	}

	// manually add public imagelink
	imageLinks = append(imageLinks, "***REMOVED***/crater-workspace/jupyter-base-notebook:ubuntu-22.04")

	resp := payload.GetImagesResp{Images: imageLinks}
	resputil.Success(c, resp)
}

// DeleteByID godoc
// @Summary 根据ID删除ImagePack
// @Description 根据ID更新ImagePack的状态为Deleted，起到删除的功能
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param ID body uint true "删除镜像的ID"
// @Router /images/delete [POST]
func (mgr *ImagePackMgr) DeleteByID(c *gin.Context) {
	imagepackQuery := query.Image
	token := util.GetToken(c)
	var err error
	imagePackDeleteRequest := &ImagePackDeleteByIDRequest{}
	if err = c.ShouldBindJSON(imagePackDeleteRequest); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	imagepackID := imagePackDeleteRequest.ID
	var imagepack *model.Image
	if imagepack, err = imagepackQuery.WithContext(c).
		Where(imagepackQuery.ID.Eq(imagepackID)).
		Where(imagepackQuery.QueueID.Eq(token.QueueID)).First(); err != nil {
		logutils.Log.Errorf("image not exist or have no permission%+v", err)
		resputil.Error(c, "failed to find imagepack or entity", resputil.NotSpecified)
		return
	}
	if _, err = imagepackQuery.WithContext(c).
		Where(imagepackQuery.ID.Eq(imagepackID)).
		Update(imagepackQuery.Status, ImagePackDeleted); err != nil {
		logutils.Log.Errorf("delete imagepack entity failed! err:%v", err)
		resputil.Error(c, "failed to find imagepack or entity", resputil.NotSpecified)
		return
	}
	logutils.Log.Info(mgr.harborClient.GetPing(c))
	name, tag := strings.Split(imagepack.NameTag, ":")[0], strings.Split(imagepack.NameTag, ":")[1]
	if err = mgr.harborClient.DeleteArtifact(c, mgr.harborClient.RegistryProject, name, tag); err != nil {
		logutils.Log.Errorf("delete imagepack artifact failed! err:%+v", err)
	}
	resputil.Success(c, "")
}

// UpdateParams godoc
// @Summary 用于更新镜像的Profile参数
// @Description 接受参数，序列化为string，更新imagepack相应字段
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param data body ImagePackParamsUpdateRequest true "更新ImagePack的name"
// @Router /images/params [post]
func (mgr *ImagePackMgr) UpdateParams(c *gin.Context) {
	imagePackParamsUpdateRequest := &ImagePackParamsUpdateRequest{}
	imagepackQuery := query.Image
	if err := c.ShouldBindJSON(imagePackParamsUpdateRequest); err != nil {
		msg := fmt.Sprintf("validate update parameters failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	logutils.Log.Infof("UpdateParams's input request body: %+v", imagePackParamsUpdateRequest)
	imagepackname := imagePackParamsUpdateRequest.ImagePackName
	data, _ := json.Marshal(imagePackParamsUpdateRequest.Data)
	params := string(data)
	if _, err := imagepackQuery.WithContext(c).Where(
		imagepackQuery.ImagePackName.Eq(imagepackname),
	).Update(imagepackQuery.Params, params); err != nil {
		logutils.Log.Errorf("update imagepack params entity failed! err:%v", err)
		resputil.Error(c, "failed to find imagepack or entity", resputil.NotSpecified)
		return
	}
	resputil.Success(c, "")
}

// GetImagePackByName godoc
// @Summary 获取imagepack的详细信息，主要用于调试
// @Description 获取imagepackname，搜索到imagepack
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param imagepackname query string true "获取ImagePack的name"
// @Router /images/get [GET]
func (mgr *ImagePackMgr) GetImagePackByName(c *gin.Context) {
	imagepackQuery := query.Image
	imagePackName := c.DefaultQuery("imagepackname", "")
	imagepack, err := imagepackQuery.WithContext(c).
		Where(imagepackQuery.ImagePackName.Eq(imagePackName)).
		First()
	if err != nil {
		msg := fmt.Sprintf("fetch imagepack by name failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	resputil.Success(c, imagepack)
}

// GetImagePackLogByName godoc
// @Summary 获取imagepack的日志信息，展示在镜像详情页
// @Description 获取imagepackname，搜索到imagepack
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param imagepackname query string true "获取ImagePack的name"
// @Router /images/log [GET]
func (mgr *ImagePackMgr) GetImagePackLogByName(c *gin.Context) {
	var req ImagePackLogRequest
	var kanikoPod *corev1.Pod
	var err error
	var podLogs string
	if err = c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate list parameters failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	if kanikoPod, err = mgr.imagepackClient.GetImagePackPod(c, req.ImagePackName, UserNameSpace); err != nil {
		msg := fmt.Sprintf("couldn't fetch imagepack pod by podName %s, err: %+v", req.ImagePackName, err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
	}
	if podLogs, err = mgr.logClient.GetPodLogs(kanikoPod); err != nil {
		msg := fmt.Sprintf("couldn't fetch log of imagepack pod, err: %+v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
	}
	resputil.Success(c, podLogs)
}
