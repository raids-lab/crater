package handler

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	harbormodelv2 "github.com/mittwald/goharbor-client/v5/apiv2/model"
	corev1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/packer"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewImagePackMgr)
}

type ImagePackMgr struct {
	name            string
	imagepackClient *crclient.ImagePackController
	harborClient    *crclient.HarborClient
	imagePacker     packer.ImagePackerInterface
}

func NewImagePackMgr(conf *RegisterConfig) Manager {
	harborClient := crclient.NewHarborClient()
	return &ImagePackMgr{
		name:            "images",
		imagepackClient: &crclient.ImagePackController{Client: conf.Client},
		harborClient:    &harborClient,
		imagePacker:     conf.ImagePacker,
	}
}

type (
	CreateKanikoRequest struct {
		SourceImage        string `json:"image"`
		PythonRequirements string `json:"requirements"`
		APTPackages        string `json:"packages"`
		Description        string `json:"description"`
	}

	UploadImageRequest struct {
		ImageLink   string        `json:"imageLink"`
		TaskType    model.JobType `json:"taskType"`
		Description string        `json:"description"`
	}

	DeleteKanikoByIDRequest struct {
		ID uint `uri:"id" binding:"required"`
	}
	DeleteImageByIDRequest struct {
		ID uint `uri:"id" binding:"required"`
	}

	GetKanikoRequest struct {
		ID uint `form:"id" binding:"required"`
	}

	GetKanikoPodRequest struct {
		ID uint `form:"id" binding:"required"`
	}

	ListAvailableImageRequest struct {
		Type model.JobType `form:"type" binding:"required"`
	}

	UpdateProjectQuotaRequest struct {
		Size int64 `json:"size" binding:"required"`
	}

	ChangeImagePublicStatusRequest struct {
		ID uint `uri:"id" binding:"required"`
	}
)

type (
	KanikoInfo struct {
		ID        uint              `json:"ID"`
		ImageLink string            `json:"imageLink"`
		Status    model.BuildStatus `json:"status"`
		CreatedAt time.Time         `json:"createdAt"`
		Size      int64             `json:"size"`
	}
	ListKanikoResponse struct {
		KanikoInfoList []KanikoInfo `json:"kanikoList"`
		TotalSize      int64        `json:"totalSize"`
	}

	ImageInfo struct {
		ID          uint          `json:"ID"`
		ImageLink   string        `json:"imageLink"`
		Description *string       `json:"description"`
		CreatedAt   time.Time     `json:"createdAt"`
		TaskType    model.JobType `json:"taskType"`
		IsPublic    bool          `json:"isPublic"`
		CreatorName string        `json:"creatorName"`
	}
	ListImageResponse struct {
		ImageInfoList []ImageInfo `json:"imageList"`
	}

	GetKanikoResponse struct {
		ID            uint              `json:"ID"`
		ImageLink     string            `json:"imageLink"`
		Status        model.BuildStatus `json:"status"`
		CreatedAt     time.Time         `json:"createdAt"`
		ImagePackName string            `json:"imagepackName"`
		Description   string            `json:"description"`
		Dockerfile    string            `json:"dockerfile"`
		PodName       string            `json:"podName"`
		PodNameSpace  string            `json:"podNameSpace"`
	}

	GetKanikoPodResponse struct {
		PodName      string `json:"name"`
		PodNameSpace string `json:"namespace"`
	}

	ListAvailableImageResponse struct {
		Images []ImageInfo `json:"images"`
	}
)

var (
	UserNameSpace   = config.GetConfig().Workspace.ImageNameSpace
	ProjectIsPublic = true
	//nolint:mnd // default project quota: 20GB
	DefaultQuotaSize = int64(20 * math.Pow(2, 30))
	ImageLinkRegExp  = `([^/]+/){2}([^:]+):([^/]+)$`
)

func (mgr *ImagePackMgr) GetName() string { return mgr.name }

func (mgr *ImagePackMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ImagePackMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/kaniko", mgr.UserListKaniko)
	g.POST("/kaniko", mgr.UserCreateKaniko)
	g.DELETE("/kaniko/:id", mgr.DeleteKanikoByID)

	g.GET("/image", mgr.UserListImage)
	g.POST("/image", mgr.UserUploadImage)
	g.DELETE("/image/:id", mgr.DeleteImageByID)

	g.GET("/available", mgr.ListAvailableImages)
	g.GET("/getbyid", mgr.GetKanikoByID)
	g.POST("/quota", mgr.UpdateProjectQuota)
	g.POST("/change/:id", mgr.UpdateImagePublicStatus)

	g.GET("/podname", mgr.GetImagepackPodName)
}

func (mgr *ImagePackMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/kaniko", mgr.AdminListKaniko)
	g.POST("/kaniko", mgr.AdminCreate)
	g.DELETE("/kaniko/:id", mgr.DeleteKanikoByID)
	g.POST("/change/:id", mgr.UpdateImagePublicStatus)
}

// UserCreateKaniko godoc
// @Summary 创建ImagePack CRD和数据库Kaniko entity
// @Description 获取参数，生成变量，调用接口
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param data body CreateKanikoRequest true "创建ImagePack CRD & Kaniko entity"
// @Router /v1/images/kaniko [POST]
func (mgr *ImagePackMgr) UserCreateKaniko(c *gin.Context) {
	req := &CreateKanikoRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	dockerfile := mgr.generateDockerfile(req)
	mgr.buildFromDockerfile(c, req, token, dockerfile)

	resputil.Success(c, "")
}

// UserUploadImage godoc
// @Summary 用户上传镜像链接
// @Description 获取上传镜像的参数，生成变量，调用接口
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param data body UploadImageRequest true "创建Image entity"
// @Router /v1/images/image [POST]
func (mgr *ImagePackMgr) UserUploadImage(c *gin.Context) {
	req := &UploadImageRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate upload parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	userQuery := query.User
	user, err := userQuery.WithContext(c).Where(userQuery.ID.Eq(token.UserID)).First()
	if err != nil {
		logutils.Log.Errorf("fetch user failed, params: %+v err:%v", req, err)
		return
	}
	imageEntity := &model.Image{
		UserID:      token.UserID,
		User:        *user,
		ImageLink:   req.ImageLink,
		TaskType:    req.TaskType,
		IsPublic:    false,
		Description: &req.Description,
	}
	imageQuery := query.Image
	if err := imageQuery.WithContext(c).Create(imageEntity); err != nil {
		logutils.Log.Errorf("create imagepack entity failed, params: %+v", imageEntity)
	}
	resputil.Success(c, "")
}

// AdminCreate godoc
// @Summary 创建ImagePack CRD和数据库kaniko entity
// @Description 获取参数，生成变量，调用接口
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param data body CreateKanikoRequest true "创建ImagePack"
// @Router /v1/admin/images/kaniko [POST]
func (mgr *ImagePackMgr) AdminCreate(c *gin.Context) {
	req := &CreateKanikoRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	logutils.Log.Infof("create params: %+v", req)
	dockerfile := mgr.generateDockerfile(req)
	mgr.buildFromDockerfile(c, req, token, dockerfile)
	resputil.Success(c, "")
}

func (mgr *ImagePackMgr) generateDockerfile(req *CreateKanikoRequest) string {
	// TODO: lzy
	fmt.Printf("%+v", req)
	return `
			# 使用ubuntu作为基础镜像
			FROM ***REMOVED***/docker.io/library/ubuntu:latest
			# 安装Python3和pip3
			RUN apt  update \
				&&  apt -y install python3 

	# 设置工作目录
	WORKDIR /app
	# 容器启动时执行的命令，可以根据需要修改
	CMD ["bash"]
	`
}

func (mgr *ImagePackMgr) buildFromDockerfile(c *gin.Context, req *CreateKanikoRequest, token util.JWTMessage, dockerfile string) {
	if err := mgr.checkExistProject(c, token); err != nil {
		logutils.Log.Errorf("check project exist failed")
		return
	}
	imageName, _ := GetImageNameAndTag(req.SourceImage)
	registryProject := fmt.Sprintf("user-%s", token.Username)
	imagepackName := fmt.Sprintf("%s-%s", token.Username, uuid.New().String()[:5])
	now := time.Now()
	imageTag := fmt.Sprintf("%d%d-%d%d-%s", now.Month(), now.Day(), now.Hour(), now.Minute(), uuid.New().String()[:4])
	imageLink := fmt.Sprintf("%s/%s/%s:%s", mgr.harborClient.RegistryServer, registryProject, imageName, imageTag)
	// create ImagePack CRD
	buildkitData := &packer.BuildKitReq{
		JobName:    imagepackName,
		Namespace:  UserNameSpace,
		Dockerfile: dockerfile,
		ImageLink:  imageLink,
	}
	kanikoQuery := query.Kaniko
	kanikoEntity := &model.Kaniko{
		UserID:        token.UserID,
		ImagePackName: imagepackName,
		ImageLink:     imageLink,
		NameSpace:     UserNameSpace,
		Status:        model.BuildJobInitial,
		Dockerfile:    &dockerfile,
		Description:   &req.Description,
		BuildSource:   model.BuildKit,
	}

	if err := kanikoQuery.WithContext(c).Create(kanikoEntity); err != nil {
		logutils.Log.Errorf("create imagepack entity failed, params: %+v", kanikoEntity)
	}

	err := mgr.imagePacker.CreateFromDockerfile(c, buildkitData)
	if err != nil {
		logutils.Log.Errorf("create buildkit job failed, params: %+v err:%v", req, err)
		return
	}
}

// UserListKaniko godoc
// @Summary 用户获取镜像构建信息
// @Description 返回该用户所有的镜像构建数据
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/images/kaniko [GET]
func (mgr *ImagePackMgr) UserListKaniko(c *gin.Context) {
	var kanikos []*model.Kaniko
	var err error
	token := util.GetToken(c)
	kanikoQuery := query.Kaniko
	kanikos, err = kanikoQuery.WithContext(c).Where(kanikoQuery.UserID.Eq(token.UserID)).Find()
	if err != nil {
		logutils.Log.Errorf("fetch kaniko entity failed, err:%v", err)
	}
	kanikoInfos := []KanikoInfo{}
	var totalSize int64
	for i := range kanikos {
		kaniko := kanikos[i]
		kanikoInfo := KanikoInfo{
			ID:        kaniko.ID,
			ImageLink: kaniko.ImageLink,
			Status:    kaniko.Status,
			CreatedAt: kaniko.CreatedAt,
			Size:      kaniko.Size,
		}
		totalSize += kaniko.Size
		kanikoInfos = append(kanikoInfos, kanikoInfo)
	}
	response := ListKanikoResponse{
		KanikoInfoList: kanikoInfos,
		TotalSize:      totalSize,
	}
	resputil.Success(c, response)
}

// UserListImage godoc
// @Summary 用户获取所有镜像数据
// @Description 返回该用户所有的镜像数据
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/images/image [GET]
func (mgr *ImagePackMgr) UserListImage(c *gin.Context) {
	var images []*model.Image
	var err error
	token := util.GetToken(c)
	imageQuery := query.Image
	if images, err = imageQuery.WithContext(c).
		Preload(query.Image.User).
		Where(imageQuery.UserID.Eq(token.UserID)).
		Or(imageQuery.IsPublic).
		Find(); err != nil {
		logutils.Log.Errorf("fetch kaniko entity failed, err:%v", err)
	}
	imageInfos := []ImageInfo{}
	for i := range images {
		image := images[i]
		imageInfo := ImageInfo{
			ID:          image.ID,
			ImageLink:   image.ImageLink,
			Description: image.Description,
			CreatedAt:   image.CreatedAt,
			IsPublic:    image.IsPublic,
			TaskType:    image.TaskType,
			CreatorName: image.User.Name,
		}
		imageInfos = append(imageInfos, imageInfo)
	}
	response := ListImageResponse{
		ImageInfoList: imageInfos,
	}
	resputil.Success(c, response)
}

// AdminListKaniko godoc
// @Summary 管理员获取相关镜像的功能
// @Description 根据GET参数type来决定搜索私有or公共镜像
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param type query int true "管理员获取镜像的类型"
// @Router /v1/admin/images/kaniko [GET]
func (mgr *ImagePackMgr) AdminListKaniko(c *gin.Context) {
	var response KanikoInfo
	resputil.Success(c, response)
}

// ListAvailableImages godoc
// @Summary 用户在运行作业时选择镜像需要调用此接口，来获取可以用的镜像
// @Description 用userID & jobType 来过滤已完成的镜像
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param type query ListAvailableImageRequest true "包含了镜像类型"
// @Router /v1/images/available [GET]
func (mgr *ImagePackMgr) ListAvailableImages(c *gin.Context) {
	token := util.GetToken(c)
	var err error
	var req ListAvailableImageRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate available image parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	imageQuery := query.Image
	var images []*model.Image
	if images, err = imageQuery.WithContext(c).
		Preload(query.Image.User).
		Where(imageQuery.UserID.Eq(token.UserID)).
		Where(imageQuery.TaskType.Eq(string(req.Type))).
		Or(imageQuery.IsPublic).
		Where(imageQuery.TaskType.Eq(string(req.Type))).
		Find(); err != nil {
		logutils.Log.Errorf("fetch available image failed, err:%v", err)
		resputil.Error(c, "fetch available image failed", resputil.NotSpecified)
		return
	}
	imageInfos := []ImageInfo{}
	for i := range images {
		image := images[i]
		imageInfo := ImageInfo{
			ID:          image.ID,
			ImageLink:   image.ImageLink,
			Description: image.Description,
			CreatedAt:   image.CreatedAt,
			IsPublic:    image.IsPublic,
			TaskType:    image.TaskType,
			CreatorName: image.User.Name,
		}
		imageInfos = append(imageInfos, imageInfo)
	}
	resp := ListAvailableImageResponse{Images: imageInfos}
	resputil.Success(c, resp)
}

// DeleteKanikoByID godoc
// @Summary 根据ID删除Kaniko entity
// @Description 根据ID更新Kaniko的状态为Deleted，起到删除的功能
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param ID body uint true "删除镜像的ID"
// @Router /v1/images/kaniko/{id} [DELETE]
func (mgr *ImagePackMgr) DeleteKanikoByID(c *gin.Context) {
	token := util.GetToken(c)
	var err error
	deleteKanikoRequest := &DeleteKanikoByIDRequest{}
	if err = c.ShouldBindUri(deleteKanikoRequest); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	kanikoID := deleteKanikoRequest.ID
	var kaniko *model.Kaniko
	kanikoQuery := query.Kaniko
	if kaniko, err = kanikoQuery.WithContext(c).Where(kanikoQuery.ID.Eq(kanikoID)).First(); err != nil {
		logutils.Log.Errorf("image not exist or have no permission%+v", err)
		resputil.Error(c, "failed to find imagepack or entity", resputil.NotSpecified)
		return
	}
	if _, err = kanikoQuery.WithContext(c).Delete(kaniko); err != nil {
		logutils.Log.Errorf("delete kaniko entity failed! err:%v", err)
		resputil.Error(c, "failed to delete kaniko", resputil.NotSpecified)
		return
	}
	if kaniko.Status == model.BuildJobFinished {
		projectName := fmt.Sprintf("user-%s", token.Username)
		name, tag := GetImageNameAndTag(kaniko.ImageLink)
		if err = mgr.harborClient.DeleteArtifact(c, projectName, name, tag); err != nil {
			logutils.Log.Errorf("delete imagepack artifact failed! err:%+v", err)
		}
	}
	resputil.Success(c, "")
}

func GetImageNameAndTag(imageLink string) (name, tag string) {
	re := regexp.MustCompile(ImageLinkRegExp)
	matches := re.FindStringSubmatch(imageLink)
	name, tag = matches[2], matches[3]
	return name, tag
}

// DeleteImageByID godoc
// @Summary 根据ID删除Image
// @Description 根据ID更新Image的状态为Deleted，起到删除的功能
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param ID body uint true "删除镜像的ID"
// @Router /v1/images/image/{id} [POST]
func (mgr *ImagePackMgr) DeleteImageByID(c *gin.Context) {
	var err error
	deleteImageRequest := &DeleteImageByIDRequest{}
	if err = c.ShouldBindUri(deleteImageRequest); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	imageID := deleteImageRequest.ID
	imageQuery := query.Image
	if _, err = imageQuery.WithContext(c).Where(imageQuery.ID.Eq(imageID)).Delete(); err != nil {
		logutils.Log.Errorf("delete image entity failed! err:%v", err)
		resputil.Error(c, "failed to delete image", resputil.NotSpecified)
	}
	resputil.Success(c, "")
}

// GetKanikoByID godoc
// @Summary 获取imagepack的详细信息
// @Description 获取imagepackname，搜索到imagepack
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param id query string true "获取ImagePack的id"
// @Router /v1/images/get [GET]
func (mgr *ImagePackMgr) GetKanikoByID(c *gin.Context) {
	kanikoQuery := query.Kaniko
	var req GetKanikoRequest
	var err error
	if err = c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate image get parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	var kaniko *model.Kaniko
	if kaniko, err = kanikoQuery.WithContext(c).
		Where(kanikoQuery.ID.Eq(req.ID)).
		First(); err != nil {
		msg := fmt.Sprintf("fetch kaniko by name failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	podName, podNameSpace := mgr.getPodName(c, req.ID)
	getKanikoResponse := GetKanikoResponse{
		ID:            kaniko.ID,
		ImageLink:     kaniko.ImageLink,
		Status:        kaniko.Status,
		CreatedAt:     kaniko.CreatedAt,
		ImagePackName: kaniko.ImagePackName,
		Description:   *kaniko.Description,
		Dockerfile:    *kaniko.Dockerfile,
		PodName:       podName,
		PodNameSpace:  podNameSpace,
	}
	resputil.Success(c, getKanikoResponse)
}

func (mgr *ImagePackMgr) checkExistProject(c *gin.Context, token util.JWTMessage) error {
	projectName := fmt.Sprintf("user-%s", token.Username)
	if exist, _ := mgr.harborClient.ProjectExists(c, projectName); !exist {
		projectRequest := &harbormodelv2.ProjectReq{
			ProjectName:  projectName,
			Public:       &ProjectIsPublic,
			StorageLimit: &DefaultQuotaSize,
		}
		userQuery := query.User
		var err error
		if _, err = userQuery.WithContext(c).
			Where(userQuery.ID.Eq(token.UserID)).
			Update(userQuery.ImageQuota, DefaultQuotaSize); err != nil {
			logutils.Log.Errorf("save user imageQuota failed, err:%v", err)
			return err
		}
		if err = mgr.harborClient.NewProject(c, projectRequest); err != nil {
			logutils.Log.Errorf("create harbor project failed! err:%+v", err)
			return err
		}
	}
	return nil
}

// UpdateProjectQuota godoc
// @Summary 更新project的配额
// @Description 传入int64参数，查找用户的project，并更新镜像存储的配额
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param req body UpdateProjectQuotaRequest true "更新镜像的ID和存储大小"
// @Router /v1/images/quota [POST]
func (mgr *ImagePackMgr) UpdateProjectQuota(c *gin.Context) {
	req := &UpdateProjectQuotaRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate update project quota failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	projectName := fmt.Sprintf("user-%s", token.Username)
	var err error
	var project *harbormodelv2.Project
	if project, err = mgr.harborClient.GetProject(c, projectName); err != nil {
		resputil.Error(c, "get harbor project failed", resputil.NotSpecified)
	}
	if err = mgr.harborClient.UpdateStorageQuotaByProjectID(c, int64(project.ProjectID), DefaultQuotaSize); err != nil {
		resputil.Error(c, "update harbor project quota failed", resputil.NotSpecified)
	}
	resputil.Success(c, "")
}

// UpdateImagePublicStatus godoc
// @Summary 更新镜像的公共或私有状态
// @Description 传入uint参数
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param req body ChangeImagePublicStatusRequest true "更新镜像的ID"
// @Router /v1/admin/images/change [POST]
func (mgr *ImagePackMgr) UpdateImagePublicStatus(c *gin.Context) {
	req := &ChangeImagePublicStatusRequest{}
	var err error
	if err = c.ShouldBindUri(req); err != nil {
		logutils.Log.Errorf("validate update image public status params failed, err %v", err)
		resputil.Error(c, "validate params", resputil.NotSpecified)
		return
	}
	imageQuery := query.Image
	if _, err = imageQuery.WithContext(c).
		Where(imageQuery.ID.Eq(req.ID)).
		Update(imageQuery.IsPublic, imageQuery.IsPublic.Not()); err != nil {
		logutils.Log.Errorf("update image image public status failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "update status failed", resputil.NotSpecified)
	}

	resputil.Success(c, "")
}

func (mgr *ImagePackMgr) GetImagepackPodName(c *gin.Context) {
	var req GetKanikoPodRequest
	var err error
	if err = c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate image get parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	podName, podNameSpace := mgr.getPodName(c, req.ID)
	resp := GetKanikoPodResponse{
		PodName:      podName,
		PodNameSpace: podNameSpace,
	}
	resputil.Success(c, resp)
}

func (mgr *ImagePackMgr) getPodName(c *gin.Context, kanikoID uint) (name, ns string) {
	var kaniko *model.Kaniko
	var err error
	kanikoQuery := query.Kaniko
	if kaniko, err = kanikoQuery.WithContext(c).
		Where(kanikoQuery.ID.Eq(kanikoID)).
		First(); err != nil {
		msg := fmt.Sprintf("fetch kaniko by name failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return "", UserNameSpace
	}
	var pod *corev1.Pod
	pod, err = mgr.imagepackClient.GetImagePackPod(c, kaniko.ImagePackName, UserNameSpace)
	if err != nil {
		msg := fmt.Sprintf("fetch kaniko pod by name failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return "", UserNameSpace
	}
	return pod.Name, UserNameSpace
}
