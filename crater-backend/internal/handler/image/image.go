package image

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/logutils"
	"gorm.io/datatypes"
)

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
		Tags:        datatypes.NewJSONType(req.Tags),
	}
	imageQuery := query.Image
	if err := imageQuery.WithContext(c).Create(imageEntity); err != nil {
		logutils.Log.Errorf("create imagepack entity failed, params: %+v", imageEntity)
	}
	resputil.Success(c, "")
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
		Order(imageQuery.CreatedAt.Desc()).
		Or(imageQuery.IsPublic).
		Order(imageQuery.CreatedAt.Desc()).
		Find(); err != nil {
		logutils.Log.Errorf("fetch kaniko entity failed, err:%v", err)
	}
	response := mgr.generateImageListResponse(images)
	resputil.Success(c, response)
}

// AdminListImage godoc
// @Summary 管理员获取所有镜像数据
// @Description 所有的镜像数据
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/admin/images/image [GET]
func (mgr *ImagePackMgr) AdminListImage(c *gin.Context) {
	var images []*model.Image
	var err error
	imageQuery := query.Image
	if images, err = imageQuery.WithContext(c).
		Preload(query.Image.User).
		Order(imageQuery.CreatedAt.Desc()).
		Find(); err != nil {
		logutils.Log.Errorf("fetch image entity failed, err:%v", err)
	}
	response := mgr.generateImageListResponse(images)
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
			UserInfo: model.UserInfo{
				Username: image.User.Name,
				Nickname: image.User.Nickname,
			},
			Tags: image.Tags.Data(),
		}
		imageInfos = append(imageInfos, imageInfo)
	}
	resp := ListAvailableImageResponse{Images: imageInfos}
	resputil.Success(c, resp)
}

// DeleteImageByID godoc
// @Summary 根据ID删除Image
// @Description 根据ID更新Image的状态为Deleted，起到删除的功能
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param ID body uint true "删除镜像的ID"
// @Router /v1/images/image/{id} [DELETE]
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

// UserDeleteImageByIDList godoc
// @Summary 用户模式根据ID列表删除Image
// @Description 用户模式根据ID列表的ID更新Image的状态为Deleted，起到删除的功能
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param ID body []uint true "删除镜像的ID"
// @Router /v1/images/deleteimage [POST]
func (mgr *ImagePackMgr) UserDeleteImageByIDList(c *gin.Context) {
	var err error
	var deleteImageListRequest DeleteImageByIDListRequest
	if err = c.ShouldBindJSON(&deleteImageListRequest); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	flag := mgr.deleteImageByIDList(c, false, deleteImageListRequest.IDList)
	if flag {
		resputil.Success(c, "")
	} else {
		resputil.Error(c, "failed to delete image", resputil.NotSpecified)
	}
}

// AdminDeleteImageByIDList godoc
// @Summary 管理员模式根据ID列表删除Image
// @Description 管理员模式根据ID列表的ID更新Image的状态为Deleted，起到删除的功能
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param ID body []uint true "删除镜像的ID"
// @Router /v1/amdin/images/deleteimage [POST]
func (mgr *ImagePackMgr) AdminDeleteImageByIDList(c *gin.Context) {
	var err error
	var deleteImageListRequest DeleteImageByIDListRequest
	if err = c.ShouldBindJSON(&deleteImageListRequest); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	flag := mgr.deleteImageByIDList(c, true, deleteImageListRequest.IDList)
	if flag {
		resputil.Success(c, "")
	} else {
		resputil.Error(c, "failed to delete image", resputil.NotSpecified)
	}
}

func (mgr *ImagePackMgr) deleteImageByIDList(c *gin.Context, isAdminMode bool, imageIDList []uint) bool {
	flag := true
	iq := query.Image
	specifiedQuery := iq.WithContext(c).Preload(iq.User)
	if !isAdminMode {
		specifiedQuery = specifiedQuery.Where(iq.UserID.Eq(util.GetToken(c).UserID))
	}
	if _, err := specifiedQuery.Where(iq.ID.In(imageIDList...)).Delete(); err != nil {
		logutils.Log.Errorf("delete image entity failed! err:%v", err)
		flag = false
	}
	return flag
}

// UserUpdateImagePublicStatus godoc
// @Summary 用户模式下更新镜像的公共或私有状态
// @Description 传入uint参数
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param req body ChangeImagePublicStatusRequest true "更新镜像的ID"
// @Router /v1/images/change [POST]
func (mgr *ImagePackMgr) UserUpdateImagePublicStatus(c *gin.Context) {
	req := &ChangeImagePublicStatusRequest{}
	var err error
	if err = c.ShouldBindUri(req); err != nil {
		logutils.Log.Errorf("validate update image public status params failed, err %v", err)
		resputil.Error(c, "validate params", resputil.NotSpecified)
		return
	}
	mgr.updateImagePublicStatus(c, false, req.ID)
}

// AdminUpdateImagePublicStatus godoc
// @Summary 管理员模式下更新镜像的公共或私有状态
// @Description 传入uint参数
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param req body ChangeImagePublicStatusRequest true "更新镜像的ID"
// @Router /v1/images/change [POST]
func (mgr *ImagePackMgr) AdminUpdateImagePublicStatus(c *gin.Context) {
	req := &ChangeImagePublicStatusRequest{}
	var err error
	if err = c.ShouldBindUri(req); err != nil {
		logutils.Log.Errorf("validate update image public status params failed, err %v", err)
		resputil.Error(c, "validate params", resputil.NotSpecified)
		return
	}
	mgr.updateImagePublicStatus(c, true, req.ID)
}

func (mgr *ImagePackMgr) updateImagePublicStatus(c *gin.Context, isAdminMode bool, imageID uint) {
	imageQuery := query.Image
	specifiedQuery := imageQuery.WithContext(c).Preload(query.Image.User)
	if !isAdminMode {
		specifiedQuery = specifiedQuery.Where(imageQuery.UserID.Eq(util.GetToken(c).UserID))
	}
	if _, err := specifiedQuery.
		Where(imageQuery.ID.Eq(imageID)).
		Update(imageQuery.IsPublic, imageQuery.IsPublic.Not()); err != nil {
		logutils.Log.Errorf("update image image public status failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "update status failed", resputil.NotSpecified)
	}
	resputil.Success(c, "")
}

// UserChangeImageDescription godoc
// @Summary 更新镜像的描述
// @Description 更新描述
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/images/description [POST]
func (mgr *ImagePackMgr) UserChangeImageDescription(c *gin.Context) {
	req := &ChangeImageDescriptionRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate description failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	mgr.changeImageDescription(c, false, req.ID, req.Description)
}

// AdminChangeImageDescription godoc
// @Summary 管理员模式下更新镜像的描述
// @Description 更新描述
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/admin/images/description [POST]
func (mgr *ImagePackMgr) AdminChangeImageDescription(c *gin.Context) {
	req := &ChangeImageDescriptionRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate description failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	mgr.changeImageDescription(c, true, req.ID, req.Description)
}

func (mgr *ImagePackMgr) changeImageDescription(c *gin.Context, isAdminMode bool, imageID uint, newDescription string) {
	imageQuery := query.Image
	specifiedQuery := imageQuery.WithContext(c).Preload(query.Image.User)
	if !isAdminMode {
		specifiedQuery = specifiedQuery.Where(imageQuery.UserID.Eq(util.GetToken(c).UserID))
	}
	if _, err := specifiedQuery.
		Where(imageQuery.ID.Eq(imageID)).
		Update(imageQuery.Description, newDescription); err != nil {
		logutils.Log.Errorf("update image description failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "update description failed", resputil.NotSpecified)
	}
	resputil.Success(c, "")
}

// UserChangeImageTaskType godoc
// @Summary 更新镜像的任务类型
// @Description 更新任务类型
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/images/type [POST]
func (mgr *ImagePackMgr) UserChangeImageTaskType(c *gin.Context) {
	req := &ChangeImageTaskTypeRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate task type failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	mgr.changeImageTaskType(c, false, req.ID, req.TaskType)
}

// AdminChangeImageTaskType godoc
// @Summary 管理员更新镜像的任务类型
// @Description 更新任务类型
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/admin/images/type [POST]
func (mgr *ImagePackMgr) AdminChangeImageTaskType(c *gin.Context) {
	req := &ChangeImageTaskTypeRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate task type failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	mgr.changeImageTaskType(c, true, req.ID, req.TaskType)
}

func (mgr ImagePackMgr) changeImageTaskType(c *gin.Context, isAdminMode bool, imageID uint, newTaskType model.JobType) {
	imageQuery := query.Image
	specifiedQuery := query.Image.WithContext(c).Preload(imageQuery.User)
	if !isAdminMode {
		specifiedQuery = specifiedQuery.Where(imageQuery.UserID.Eq(util.GetToken(c).UserID))
	}
	if _, err := specifiedQuery.
		Where(imageQuery.ID.Eq(imageID)).
		Update(imageQuery.TaskType, newTaskType); err != nil {
		logutils.Log.Errorf("update image task type failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "update task type failed", resputil.NotSpecified)
	}
	resputil.Success(c, "")
}

func (mgr *ImagePackMgr) generateImageListResponse(images []*model.Image) ListImageResponse {
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
			UserInfo: model.UserInfo{
				Username: image.User.Name,
				Nickname: image.User.Nickname,
			},
			Tags: image.Tags.Data(),
		}
		imageInfos = append(imageInfos, imageInfo)
	}
	return ListImageResponse{
		ImageInfoList: imageInfos,
	}
}

// UserChangeImageTagsType godoc
// @Summary 更新镜像的标签
// @Description 更新标签
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/images/tags [POST]
func (mgr *ImagePackMgr) UserChangeImageTags(c *gin.Context) {
	req := &ChangeImageTagsRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate tags data failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	mgr.changeImageTags(c, req.ID, req.Tags)
}

// AdminChangeImageTagsType godoc
// @Summary 管理员更新镜像的标签
// @Description 管理员更新标签
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/admin/images/tags [POST]
func (mgr *ImagePackMgr) AdminChangeImageTags(c *gin.Context) {
	req := &ChangeImageTagsRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate tags data failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	mgr.changeImageTags(c, req.ID, req.Tags)
}

func (mgr *ImagePackMgr) changeImageTags(c *gin.Context, imageID uint, newTags []string) {
	imageQuery := query.Image
	if _, err := imageQuery.WithContext(c).
		Where(imageQuery.ID.Eq(imageID)).
		Update(imageQuery.Tags, datatypes.NewJSONType(newTags)); err != nil {
		logutils.Log.Errorf("update image tags failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "update tags failed", resputil.NotSpecified)
	}
	resputil.Success(c, "")
}
