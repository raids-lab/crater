package handler

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/packer"
	"github.com/raids-lab/crater/pkg/utils"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewImagePackMgr)
}

type ImagePackMgr struct {
	name            string
	imagepackClient *crclient.ImagePackController
	imagePacker     packer.ImagePackerInterface
	imageRegistry   imageregistry.ImageRegistryInterface
}

func (mgr *ImagePackMgr) GetName() string { return mgr.name }

func (mgr *ImagePackMgr) RegisterPublic(_ *gin.RouterGroup) {}

//nolint:dupl// 路由注册
func (mgr *ImagePackMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/kaniko", mgr.UserListKaniko)
	g.POST("/kaniko", mgr.UserCreateKaniko)
	g.POST("/dockerfile", mgr.UserCreateByDockerfile)
	g.DELETE("/kaniko/:id", mgr.DeleteKanikoByID)

	g.GET("/image", mgr.UserListImage)
	g.POST("/image", mgr.UserUploadImage)
	g.DELETE("/image/:id", mgr.DeleteImageByID)

	g.GET("/available", mgr.ListAvailableImages)
	g.GET("/getbyid", mgr.GetKanikoByID)
	g.POST("/quota", mgr.UpdateProjectQuota)
	g.POST("/change/:id", mgr.UpdateImagePublicStatus)

	g.GET("/podname", mgr.GetImagepackPodName)
	g.POST("/credential", mgr.UserGetProjectCredential)
}

func (mgr *ImagePackMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/kaniko", mgr.AdminListKaniko)
	g.POST("/kaniko", mgr.AdminCreate)
	g.DELETE("/kaniko/:id", mgr.DeleteKanikoByID)
	g.POST("/change/:id", mgr.UpdateImagePublicStatus)
}

func NewImagePackMgr(conf *RegisterConfig) Manager {
	return &ImagePackMgr{
		name:            "images",
		imagepackClient: &crclient.ImagePackController{Client: conf.Client},
		imagePacker:     conf.ImagePacker,
		imageRegistry:   conf.ImageRegistry,
	}
}

type (
	CreateKanikoRequest struct {
		SourceImage        string `json:"image"`
		PythonRequirements string `json:"requirements"`
		APTPackages        string `json:"packages"`
		Description        string `json:"description"`
	}

	CreateByDockerfileRequest struct {
		Description string `json:"description"`
		Dockerfile  string `json:"dockerfile"`
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
		ID          uint              `json:"ID"`
		ImageLink   string            `json:"imageLink"`
		Status      model.BuildStatus `json:"status"`
		CreatedAt   time.Time         `json:"createdAt"`
		Size        int64             `json:"size"`
		Description string            `json:"description"`
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

	GetProjectCredentialResponse struct {
		Name     *string `json:"name"`
		Password *string `json:"password"`
	}
)

var (
	UserNameSpace   = config.GetConfig().Workspace.ImageNamespace
	ProjectIsPublic = true
	//nolint:mnd // default project quota: 20GB
	DefaultQuotaSize = int64(20 * math.Pow(2, 30))
)

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
	mgr.buildFromDockerfile(c, token, req.SourceImage, req.Description, dockerfile)
}

// UserCreateByDockerfile godoc
// @Summary 接受用户传入的Dockerfile和描述，创建镜像
// @Description 获取参数，提取Dockerfile中的基础镜像，调用接口
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param data body CreateByDockerfileRequest true "创建ImagePack CRD"
// @Router /v1/images/dockerfile [POST]
func (mgr *ImagePackMgr) UserCreateByDockerfile(c *gin.Context) {
	req := &CreateByDockerfileRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	baseImage, err := extractBaseImageFromDockerfile(req.Dockerfile)
	if err != nil {
		msg := fmt.Sprintf("failed to extract base image from Dockerfile, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}

	mgr.buildFromDockerfile(c, token, baseImage, req.Description, req.Dockerfile)
}

func extractBaseImageFromDockerfile(dockerfile string) (string, error) {
	lines := strings.Split(dockerfile, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "FROM") {
			for strings.HasSuffix(line, "\\") {
				line = line[:len(line)-1]
				line = strings.TrimSpace(line)
				// last line check
				if i+1 >= len(lines) {
					return "", fmt.Errorf("unexpected end of Dockerfile after line: %s", line)
				}
				nextLine := strings.TrimSpace(lines[i+1])
				line += " " + nextLine
				i++ // move to next line
			}

			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
			return "", fmt.Errorf("invalid FROM instruction: %s", line)
		}
	}
	return "", fmt.Errorf("no FROM instruction found in Dockerfile")
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
	mgr.buildFromDockerfile(c, token, req.SourceImage, req.Description, dockerfile)
}

func (mgr *ImagePackMgr) generateDockerfile(req *CreateKanikoRequest) string {
	// Handle APT packages
	aptInstallSection := "\n# No APT packages specified"
	if req.APTPackages != "" {
		aptPackages := strings.Fields(req.APTPackages) // split by space
		aptInstallSection = fmt.Sprintf(`
# Install APT packages
RUN apt-get update && apt-get install -y %s && \
    rm -rf /var/lib/apt/lists/*`, strings.Join(aptPackages, " "))
	}

	// Generate requirements.txt and install Python dependencies
	requirementsSection := "\n# No Python dependencies specified"
	if req.PythonRequirements != "" {
		requirementsSection = fmt.Sprintf(`
# Install Python dependencies
RUN echo -e "%s" > /requirements.txt && \
    pip install --extra-index-url https://mirrors.aliyun.com/pypi/simple/ --no-cache-dir -r /requirements.txt`,
			strings.ReplaceAll(req.PythonRequirements, "\n", "\\n"))
	}

	// Generate Dockerfile
	dockerfile := fmt.Sprintf(`FROM %s
USER root
%s
%s
`, req.SourceImage, aptInstallSection, requirementsSection)

	return dockerfile
}

func (mgr *ImagePackMgr) buildFromDockerfile(c *gin.Context, token util.JWTMessage, sourceImage, description, dockerfile string) {
	if err := mgr.imageRegistry.CheckOrCreateProjectForUser(c, token.Username); err != nil {
		resputil.Error(c, "create harbor project failed", resputil.NotSpecified)
		return
	}
	imagepackName := fmt.Sprintf("%s-%s", token.Username, uuid.New().String()[:5])
	imageLink, err := utils.GenerateNewImageLink(sourceImage, token.Username)
	if err != nil {
		resputil.Error(c, "generate new image link failed", resputil.NotSpecified)
		return
	}

	// create ImagePack CRD
	buildkitData := &packer.BuildKitReq{
		JobName:     imagepackName,
		Namespace:   UserNameSpace,
		Dockerfile:  &dockerfile,
		ImageLink:   imageLink,
		UserID:      token.UserID,
		Description: &description,
	}

	if err := mgr.imagePacker.CreateFromDockerfile(c, buildkitData); err != nil {
		resputil.Error(c, "create imagepack failed", resputil.NotSpecified)
		return
	}

	resputil.Success(c, "")
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
			ID:          kaniko.ID,
			ImageLink:   kaniko.ImageLink,
			Status:      kaniko.Status,
			CreatedAt:   kaniko.CreatedAt,
			Size:        kaniko.Size,
			Description: *kaniko.Description,
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
		Or(imageQuery.IsPublic).
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
	var deleteKanikoRequest DeleteKanikoByIDRequest
	if err = c.ShouldBindUri(&deleteKanikoRequest); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	kanikoID := deleteKanikoRequest.ID
	var kaniko *model.Kaniko
	k := query.Kaniko
	if kaniko, err = k.WithContext(c).
		Preload(query.Image.User).
		Where(k.ID.Eq(kanikoID)).
		Where(k.UserID.Eq(token.UserID)).First(); err != nil {
		logutils.Log.Errorf("image not exist or have no permission%+v", err)
		resputil.Error(c, "failed to find imagepack or entity", resputil.NotSpecified)
		return
	}
	if _, err = k.WithContext(c).Delete(kaniko); err != nil {
		logutils.Log.Errorf("delete kaniko entity failed! err:%v", err)
		resputil.Error(c, "failed to delete kaniko", resputil.NotSpecified)
		return
	}
	if kaniko.Status == model.BuildJobFinished {
		if err = mgr.imageRegistry.DeleteImageFromProject(c, kaniko.ImageLink); err != nil {
			logutils.Log.Errorf("delete imagepack artifact failed! err:%+v", err)
		}
	}
	resputil.Success(c, "")
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
	var deleteImageRequest DeleteImageByIDRequest
	if err = c.ShouldBindUri(&deleteImageRequest); err != nil {
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
	if err := mgr.imageRegistry.UpdateQuotaForProject(c, projectName, DefaultQuotaSize); err != nil {
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
	token := util.GetToken(c)
	imageQuery := query.Image
	if _, err = imageQuery.WithContext(c).
		Preload(query.Image.User).
		Where(imageQuery.ID.Eq(req.ID)).
		Where(imageQuery.UserID.Eq(token.UserID)).
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
	if err != nil || pod == nil {
		return "", UserNameSpace
	}
	return pod.Name, UserNameSpace
}

// UserGetProjectCredential godoc
// @Summary 创建用户的harbor项目，并返回用户的harbor项目的凭证
// @Description 获取参数，生成变量，调用接口
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/images/credential [POST]
func (mgr *ImagePackMgr) UserGetProjectCredential(c *gin.Context) {
	token := util.GetToken(c)
	if err := mgr.imageRegistry.CheckOrCreateProjectForUser(c, token.Username); err != nil {
		logutils.Log.Errorf("check project failed")
		resputil.Error(c, "check or create project failed", resputil.NotSpecified)
		return
	}
	if exist := mgr.imageRegistry.CheckUserExist(c, token.Username); exist {
		err := mgr.imageRegistry.DeleteUser(c, token.Username)
		if err != nil {
			logutils.Log.Errorf("delete user failed")
			resputil.Error(c, "delete user failed", resputil.NotSpecified)
			return
		}
	}
	password, err := mgr.imageRegistry.CreateUser(c, token.Username)
	if err != nil {
		logutils.Log.Errorf("create user failed: %+v", err)
		resputil.Error(c, "create user failed", resputil.NotSpecified)
		return
	}
	if err = mgr.imageRegistry.AddProjectMember(c, token.Username); err != nil {
		logutils.Log.Errorf("add project member failed")
		resputil.Error(c, "add project member failed", resputil.NotSpecified)
		return
	}
	resp := GetProjectCredentialResponse{
		Name:     &token.Username,
		Password: &password,
	}
	resputil.Success(c, resp)
}
