package image

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/logutils"
	"gorm.io/datatypes"
)

// UserUploadImage godoc
//
//	@Summary		用户上传镜像链接
//	@Description	获取上传镜像的参数，生成变量，调用接口
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body	UploadImageRequest	true	"创建Image entity"
//	@Router			/v1/images/image [POST]
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
		ImageSource: model.ImageUploadType,
	}
	imageQuery := query.Image
	if err := imageQuery.WithContext(c).Create(imageEntity); err != nil {
		logutils.Log.Errorf("create imagepack entity failed, params: %+v", imageEntity)
	}
	resputil.Success(c, "")
}

// UserListImage godoc
//
//	@Summary		用户获取所有镜像数据
//	@Description	返回该用户所有的镜像数据
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/image [GET]
func (mgr *ImagePackMgr) UserListImage(c *gin.Context) {
	var response ListImageResponse
	imageInfoList := mgr.getImages(c)
	response = ListImageResponse{ImageInfoList: imageInfoList}
	resputil.Success(c, response)
}

func (mgr *ImagePackMgr) getImages(c *gin.Context) []*ImageInfo {
	token := util.GetToken(c)
	var err error
	var imageInfoList []*ImageInfo
	imageQuery := query.Image
	// 1. 获取公共镜像	TODO: 移除IsPublic的兼容代码
	var oldPublicImages []*model.Image
	if oldPublicImages, err = imageQuery.WithContext(c).
		Preload(query.Image.User).
		Or(imageQuery.IsPublic).
		Order(imageQuery.CreatedAt.Desc()).
		Find(); err != nil {
		logutils.Log.Errorf("fetch kaniko entity failed, err:%v", err)
	}
	imageInfoList = mgr.processImageListResponse(oldPublicImages, model.Public, imageInfoList)
	// 2. 获取公共镜像（ImageAccountID = 1）
	newPublicImages := mgr.getPublicImages(c)
	imageInfoList = mgr.processImageListResponse(newPublicImages, model.Public, imageInfoList)
	// 3. 获取本账户拥有的镜像
	accountSharedImages := mgr.getAccountSharedImages(c, token.AccountID)
	imageInfoList = mgr.processImageListResponse(accountSharedImages, model.AccountShare, imageInfoList)
	// 4. 获取私有镜像
	var privateImages []*model.Image
	if privateImages, err = imageQuery.WithContext(c).
		Preload(query.Image.User).
		Where(imageQuery.UserID.Eq(token.UserID)).
		Order(imageQuery.CreatedAt.Desc()).
		Find(); err != nil {
		logutils.Log.Errorf("fetch kaniko entity failed, err:%v", err)
	}
	imageInfoList = mgr.processImageListResponse(privateImages, model.Private, imageInfoList)
	// 5. 获取其他用户分享的镜像
	userSharedImages := mgr.getUserSharedImages(c, token.UserID)
	imageInfoList = mgr.processImageListResponse(userSharedImages, model.UserShare, imageInfoList)
	// 6. 去除重复的镜像
	imageInfoList = mgr.deduplicate(imageInfoList)
	// 7. 降序排序
	sort.Slice(imageInfoList, func(i, j int) bool {
		return imageInfoList[i].CreatedAt.After(imageInfoList[j].CreatedAt)
	})
	return imageInfoList
}

// AdminListImage godoc
//
//	@Summary		管理员获取所有镜像数据
//	@Description	所有的镜像数据
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/admin/images/image [GET]
func (mgr *ImagePackMgr) AdminListImage(c *gin.Context) {
	var publicImages []*model.Image
	var privateImages []*model.Image
	var publicImageIDs []uint
	var oldPublicImages []uint
	var imageInfoList []*ImageInfo
	var err error
	imageAccountQuery := query.ImageAccount
	imageQuery := query.Image
	if err = imageAccountQuery.WithContext(c).
		Where(imageAccountQuery.AccountID.Eq(model.DefaultAccountID)).
		Pluck(imageAccountQuery.ImageID, &publicImageIDs); err != nil {
		logutils.Log.Errorf("get public account ids failed, err:%v", err)
		resputil.Error(c, "get public account ids failed", resputil.NotSpecified)
		return
	}
	// TODO: 移除IsPublic的兼容代码
	if err = imageQuery.WithContext(c).
		Where(imageQuery.IsPublic).
		Pluck(imageQuery.ID, &oldPublicImages); err != nil {
		logutils.Log.Errorf("get old public account ids failed, err:%v", err)
		resputil.Error(c, "get old public account ids failed", resputil.NotSpecified)
		return
	}
	publicImageIDs = append(publicImageIDs, oldPublicImages...)
	if publicImages, err = imageQuery.WithContext(c).
		Preload(query.Image.User).
		Where(imageQuery.ID.In(publicImageIDs...)).
		Find(); err != nil {
		logutils.Log.Errorf("fetch public image entity failed, err:%v", err)
		resputil.Error(c, "fetch public image entity failed", resputil.NotSpecified)
		return
	}
	imageInfoList = mgr.processImageListResponse(publicImages, model.Public, imageInfoList)
	if privateImages, err = imageQuery.WithContext(c).
		Preload(query.Image.User).
		Where(imageQuery.ID.NotIn(publicImageIDs...)).
		Find(); err != nil {
		logutils.Log.Errorf("fetch public image entity failed, err:%v", err)
		resputil.Error(c, "fetch public image entity failed", resputil.NotSpecified)
		return
	}
	imageInfoList = mgr.processImageListResponse(privateImages, model.Private, imageInfoList)
	sort.Slice(imageInfoList, func(i, j int) bool {
		return imageInfoList[i].CreatedAt.After(imageInfoList[j].CreatedAt)
	})
	response := ListImageResponse{ImageInfoList: imageInfoList}
	resputil.Success(c, response)
}

// ListAvailableImages godoc
//
//	@Summary		用户在运行作业时选择镜像需要调用此接口，来获取可以用的镜像
//	@Description	用userID & jobType 来过滤已完成的镜像
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			type	query	ListAvailableImageRequest	true	"包含了镜像类型"
//	@Router			/v1/images/available [GET]
func (mgr *ImagePackMgr) ListAvailableImages(c *gin.Context) {
	imageList := mgr.getImages(c)
	resp := ListAvailableImageResponse{Images: imageList}
	resputil.Success(c, resp)
}

//nolint:dupl // ignore duplicate code
func (mgr *ImagePackMgr) getUserSharedImages(c *gin.Context, userID uint) []*model.Image {
	sharedImages := []*model.Image{}
	imageShareQuery := query.ImageUser
	imageShares, err := imageShareQuery.WithContext(c).
		Preload(imageShareQuery.Image).
		Preload(imageShareQuery.Image.User).
		Where(imageShareQuery.UserID.Eq(userID)).
		Find()
	if err != nil {
		logutils.Log.Errorf("fetch shared image failed, err:%v", err)
		return sharedImages
	}

	for _, imageShare := range imageShares {
		sharedImages = append(sharedImages, &imageShare.Image)
	}
	return sharedImages
}

func (mgr *ImagePackMgr) getPublicImages(c *gin.Context) []*model.Image {
	sharedImages := []*model.Image{}
	imageShareQuery := query.ImageAccount
	imageShares, err := imageShareQuery.WithContext(c).
		Preload(imageShareQuery.Image).
		Preload(imageShareQuery.Image.User).
		Where(imageShareQuery.AccountID.Eq(model.DefaultAccountID)).
		Find()
	if err != nil {
		logutils.Log.Errorf("fetch shared image failed, err:%v", err)
		return sharedImages
	}
	for _, imageShare := range imageShares {
		sharedImages = append(sharedImages, &imageShare.Image)
	}
	return sharedImages
}

//nolint:dupl // ignore duplicate code
func (mgr *ImagePackMgr) getAccountSharedImages(c *gin.Context, accountID uint) []*model.Image {
	sharedImages := []*model.Image{}
	imageShareQuery := query.ImageAccount
	imageShares, err := imageShareQuery.WithContext(c).
		Preload(imageShareQuery.Image).
		Preload(imageShareQuery.Image.User).
		Where(imageShareQuery.AccountID.Eq(accountID)).
		Find()
	if err != nil {
		logutils.Log.Errorf("fetch shared image failed, err:%v", err)
		return sharedImages
	}
	for _, imageShare := range imageShares {
		sharedImages = append(sharedImages, &imageShare.Image)
	}
	return sharedImages
}

func (mgr *ImagePackMgr) deduplicate(imageInfoList []*ImageInfo) []*ImageInfo {
	seen := make(map[uint]struct{})
	result := []*ImageInfo{}
	for _, img := range imageInfoList {
		if _, exists := seen[img.ID]; !exists {
			seen[img.ID] = struct{}{}
			result = append(result, img)
		}
	}
	return result
}

// DeleteImageByID godoc
//
//	@Summary		根据ID删除Image
//	@Description	根据ID更新Image的状态为Deleted，起到删除的功能
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			ID	body	uint	true	"删除镜像的ID"
//	@Router			/v1/images/image/{id} [DELETE]
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
//
//	@Summary		用户模式根据ID列表删除Image
//	@Description	用户模式根据ID列表的ID更新Image的状态为Deleted，起到删除的功能
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			ID	body	[]uint	true	"删除镜像的ID"
//	@Router			/v1/images/deleteimage [POST]
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
//
//	@Summary		管理员模式根据ID列表删除Image
//	@Description	管理员模式根据ID列表的ID更新Image的状态为Deleted，起到删除的功能
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			ID	body	[]uint	true	"删除镜像的ID"
//	@Router			/v1/amdin/images/deleteimage [POST]
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

// AdminUpdateImagePublicStatus godoc
//
//	@Summary		管理员模式下更新镜像的公共或私有状态
//	@Description	传入uint参数
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			req	body	ChangeImagePublicStatusRequest	true	"更新镜像的ID"
//	@Router			/v1/images/change [POST]
func (mgr *ImagePackMgr) AdminUpdateImagePublicStatus(c *gin.Context) {
	req := &ChangeImagePublicStatusRequest{}
	var err error
	if err = c.ShouldBindUri(req); err != nil {
		logutils.Log.Errorf("validate update image public status params failed, err %v", err)
		resputil.Error(c, "validate params", resputil.NotSpecified)
		return
	}
	imageAccountQuery := query.ImageAccount
	imageQuery := query.Image
	if _, err = imageAccountQuery.WithContext(c).
		Where(imageAccountQuery.ImageID.Eq(req.ID)).
		Where(imageAccountQuery.AccountID.Eq(model.DefaultAccountID)).
		First(); err != nil {
		imageAccountEntity := &model.ImageAccount{
			ImageID:   req.ID,
			AccountID: model.DefaultAccountID,
		}
		if err = imageAccountQuery.WithContext(c).Create(imageAccountEntity); err != nil {
			logutils.Log.Errorf("create image share entity failed, err %v", err)
			resputil.Error(c, fmt.Sprintf("%+v", err), resputil.NotSpecified)
			return
		}
		// TODO: 移除IsPublic的兼容代码
		_, _ = imageQuery.WithContext(c).
			Where(imageQuery.ID.Eq(req.ID)).
			Update(imageQuery.IsPublic, true)
	} else {
		if _, err = imageAccountQuery.WithContext(c).
			Where(imageAccountQuery.ImageID.Eq(req.ID)).
			Where(imageAccountQuery.AccountID.Eq(model.DefaultAccountID)).
			Delete(); err != nil {
			logutils.Log.Errorf("delete image share entity failed, err %v", err)
			resputil.Error(c, fmt.Sprintf("%+v", err), resputil.NotSpecified)
			return
		}
		// TODO: 移除IsPublic的兼容代码
		_, _ = imageQuery.WithContext(c).
			Where(imageQuery.ID.Eq(req.ID)).
			Update(imageQuery.IsPublic, false)
	}
	resputil.Success(c, "")
}

// UserChangeImageDescription godoc
//
//	@Summary		更新镜像的描述
//	@Description	更新描述
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/description [POST]
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
//
//	@Summary		管理员模式下更新镜像的描述
//	@Description	更新描述
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/admin/images/description [POST]
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
//
//	@Summary		更新镜像的任务类型
//	@Description	更新任务类型
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/type [POST]
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
//
//	@Summary		管理员更新镜像的任务类型
//	@Description	更新任务类型
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/admin/images/type [POST]
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

func (mgr *ImagePackMgr) processImageListResponse(
	images []*model.Image,
	status model.ImageShareType,
	imageInfoList []*ImageInfo,
) []*ImageInfo {
	imageInfos := []*ImageInfo{}
	for i := range images {
		image := images[i]
		imageInfo := &ImageInfo{
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
			Tags:             image.Tags.Data(),
			ImageBuildSource: image.ImageSource,
			ImagePackName:    image.ImagePackName,
			ImageShareStatus: status,
		}
		imageInfos = append(imageInfos, imageInfo)
	}
	imageInfoList = append(imageInfoList, imageInfos...)
	return imageInfoList
}

// UserChangeImageTagsType godoc
//
//	@Summary		更新镜像的标签
//	@Description	更新标签
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/tags [POST]
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
//
//	@Summary		管理员更新镜像的标签
//	@Description	管理员更新标签
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/admin/images/tags [POST]
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

// UserShareImageWithAccount godoc
//
//	@Summary		分享镜像到账户或用户
//	@Description	普通用户分享镜像到其他账户或用户
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/share [POST]
func (mgr *ImagePackMgr) UserShareImage(c *gin.Context) {
	req := &ShareImageRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate tags data failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	for _, id := range req.IDList {
		if req.Type == "user" {
			if err := mgr.createImageUserEntity(c, req.ImageID, id); err != nil {
				logutils.Log.Errorf("create image share entity failed, err %v", err)
				resputil.Error(c, fmt.Sprintf("%+v", err), resputil.NotSpecified)
				return
			}
		} else {
			if err := mgr.createImageAccountEntity(c, req.ImageID, id); err != nil {
				logutils.Log.Errorf("create image share entity failed, err %v", err)
				resputil.Error(c, fmt.Sprintf("%+v", err), resputil.NotSpecified)
				return
			}
		}
	}
	resputil.Success(c, "")
}

//nolint:dupl // ignore duplicate code
func (mgr *ImagePackMgr) createImageAccountEntity(c *gin.Context, imageID, accountID uint) error {
	accountImageQuery := query.ImageAccount
	accountQuery := query.Account
	if _, err := accountQuery.WithContext(c).Where(accountQuery.ID.Eq(accountID)).First(); err != nil {
		return err
	}
	// check if the image has been shared to this account
	imageShareEntity, _ := accountImageQuery.WithContext(c).
		Where(accountImageQuery.ImageID.Eq(imageID)).
		Where(accountImageQuery.AccountID.Eq(accountID)).
		First()
	if imageShareEntity != nil {
		return fmt.Errorf("image has been shared to this account")
	}
	// create a new image share entity
	imageShareEntity = &model.ImageAccount{
		ImageID:   imageID,
		AccountID: accountID,
	}
	if err := accountImageQuery.WithContext(c).Create(imageShareEntity); err != nil {
		return err
	}
	return nil
}

//nolint:dupl // ignore duplicate code
func (mgr *ImagePackMgr) createImageUserEntity(c *gin.Context, imageID, userID uint) error {
	userImageQuery := query.ImageUser
	userQuery := query.User
	// check if the user exists
	if _, err := userQuery.WithContext(c).Where(userQuery.ID.Eq(userID)).First(); err != nil {
		return err
	}
	// check if the image has been shared to this user
	imageShareEntity, _ := userImageQuery.WithContext(c).
		Where(userImageQuery.ImageID.Eq(imageID)).
		Where(userImageQuery.UserID.Eq(userID)).
		First()
	if imageShareEntity != nil {
		return fmt.Errorf("image has been shared to this user")
	}
	// create a new image share entity
	imageShareEntity = &model.ImageUser{
		ImageID: imageID,
		UserID:  userID,
	}
	if err := userImageQuery.WithContext(c).Create(imageShareEntity); err != nil {
		return err
	}
	return nil
}

// UserCancelShareImageWithAccount godoc
//
//	@Summary		取消分享镜像到账户
//	@Description	普通用户取消分享镜像到其他账户
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/share [DELETE]
func (mgr *ImagePackMgr) UserCancelShareImage(c *gin.Context) {
	req := &CancelShareImageRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate tags data failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	if req.Type == "user" {
		if err := mgr.cancelShareImageWithUser(c, req.ImageID, req.ID); err != nil {
			resputil.Error(c, fmt.Sprintf("%v", err), resputil.NotSpecified)
			return
		}
	} else {
		if err := mgr.cancelShareImageWithAccount(c, req.ImageID, req.ID); err != nil {
			resputil.Error(c, fmt.Sprintf("%v", err), resputil.NotSpecified)
			return
		}
	}
	resputil.Success(c, "")
}

//nolint:dupl // ignore duplicate code
func (mgr *ImagePackMgr) cancelShareImageWithAccount(c *gin.Context, imageID, accountID uint) error {
	accountImageQuery := query.ImageAccount
	userQuery := query.User
	// check if the account exists
	if _, err := userQuery.WithContext(c).Where(userQuery.ID.Eq(accountID)).First(); err != nil {
		return fmt.Errorf("account does not exist: %w", err)
	}
	// check if the image has been shared to this account
	imageAccountEntity, _ := accountImageQuery.WithContext(c).
		Where(accountImageQuery.ImageID.Eq(imageID)).
		Where(accountImageQuery.AccountID.Eq(accountID)).
		First()
	if imageAccountEntity == nil {
		return fmt.Errorf("image hasn't been shared to this account")
	}
	// create a new image share entity
	if _, err := accountImageQuery.WithContext(c).Delete(imageAccountEntity); err != nil {
		return fmt.Errorf("cancel share with account failed: %w", err)
	}
	return nil
}

//nolint:dupl // ignore duplicate code
func (mgr *ImagePackMgr) cancelShareImageWithUser(c *gin.Context, imageID, userID uint) error {
	userImageQuery := query.ImageUser
	userQuery := query.User
	// check if the user exists
	if _, err := userQuery.WithContext(c).Where(userQuery.ID.Eq(userID)).First(); err != nil {
		return fmt.Errorf("user does not exist: %w", err)
	}
	// check if the image has been shared to this user
	imageUserEntity, _ := userImageQuery.WithContext(c).
		Where(userImageQuery.ImageID.Eq(imageID)).
		Where(userImageQuery.UserID.Eq(userID)).
		First()
	if imageUserEntity == nil {
		return fmt.Errorf("image hasn't been shared to this user")
	}
	// create a new image share entity
	if _, err := userImageQuery.WithContext(c).Delete(imageUserEntity); err != nil {
		return fmt.Errorf("cancel share with user failed: %w", err)
	}
	return nil
}

// GetImageGrantedUserOrAccount godoc
//
//	@Summary		获取镜像分享到的用户或账户
//	@Description	获取镜像分享到的用户或账户
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/grant [GET]
func (mgr *ImagePackMgr) GetImageGrantedUserOrAccount(c *gin.Context) {
	req := &ImageGrantRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		logutils.Log.Errorf("validate imageID failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}

	grangtedAccounts := []ImageGrantedAccounts{}
	imageAccountQuery := query.ImageAccount
	imageAccounts, err := imageAccountQuery.WithContext(c).
		Preload(imageAccountQuery.Account).
		Where(imageAccountQuery.ImageID.Eq(req.ImageID)).
		Find()
	if err != nil {
		logutils.Log.Errorf("fetch image share with account failed, err:%v", err)
		resputil.Error(c, "fetch image share with account failed", resputil.NotSpecified)
		return
	}
	for _, ia := range imageAccounts {
		grangtedAccounts = append(grangtedAccounts, ImageGrantedAccounts{
			ID:   ia.Account.ID,
			Name: ia.Account.Nickname,
		})
	}

	grantedUsers := []ImageGrantedUsers{}
	imageUserQuery := query.ImageUser
	imageUsers, err := imageUserQuery.WithContext(c).
		Preload(imageUserQuery.User).
		Where(imageUserQuery.ImageID.Eq(req.ImageID)).
		Find()
	if err != nil {
		logutils.Log.Errorf("fetch image share with user failed, err:%v", err)
		resputil.Error(c, "fetch image share with user failed", resputil.NotSpecified)
		return
	}
	for _, iu := range imageUsers {
		grantedUsers = append(grantedUsers, ImageGrantedUsers{
			ID:       iu.User.ID,
			Name:     iu.User.Name,
			Nickname: iu.User.Nickname,
		})
	}
	resputil.Success(c, ImageGrantResponse{UserList: grantedUsers, AccountList: grangtedAccounts})
}

// UserSearchNotGrantedAccounts godoc
//
//	@Summary		获取未被分享该镜像的账户
//	@Description	获取未被分享该镜像的账户
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/account [GET]
func (mgr *ImagePackMgr) UserGetImageUngrantedAccounts(c *gin.Context) {
	req := &AccountSearchRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		logutils.Log.Errorf("validate search failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	// 1. 查询已分享的AccountID
	sharedAccountIDs := []uint{}
	imageAccountQuery := query.ImageAccount
	if err := imageAccountQuery.WithContext(c).
		Where(imageAccountQuery.ImageID.Eq(req.ImageID)).
		Pluck(imageAccountQuery.AccountID, &sharedAccountIDs); err != nil {
		logutils.Log.Errorf("query shared account ids failed, err:%v", err)
		resputil.Error(c, "query shared account ids failed", resputil.NotSpecified)
		return
	}
	// 2. 禁止用户分享给默认用户
	sharedAccountIDs = append(sharedAccountIDs, model.DefaultAccountID)
	// 3. 获取用户当前所在的账户
	userAccountQuery := query.UserAccount
	accountIDs := []uint{}
	if err := userAccountQuery.WithContext(c).
		Where(userAccountQuery.UserID.Eq(util.GetToken(c).UserID)).
		Pluck(userAccountQuery.AccountID, &accountIDs); err != nil {
		logutils.Log.Errorf("get user current account failed, err:%v", err)
		resputil.Error(c, "get user current account failed", resputil.NotSpecified)
	}
	// 4. 查询未被分享的Account
	accountQuery := query.Account
	accounts, err := accountQuery.WithContext(c).
		Where(accountQuery.ID.In(accountIDs...)).
		Where(accountQuery.ID.NotIn(sharedAccountIDs...)).
		Find()
	if err != nil {
		logutils.Log.Errorf("fetch account failed, err:%v", err)
		resputil.Error(c, "fetch account failed", resputil.NotSpecified)
		return
	}
	// 5. 生成请求内容
	accountInfos := []ImageGrantedAccounts{}
	for _, account := range accounts {
		accountInfos = append(accountInfos, ImageGrantedAccounts{
			ID:   account.ID,
			Name: account.Nickname,
		})
	}
	resp := ImageGrantResponse{AccountList: accountInfos}
	resputil.Success(c, resp)
}

// UserSearchUngrantedUsers godoc
//
//	@Summary		获取未被分享该镜像的用户（支持名称模糊搜索）
//	@Description	获取未被分享该镜像的用户
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/user [GET]
func (mgr *ImagePackMgr) UserSearchUngrantedUsers(c *gin.Context) {
	req := &UserSearchRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		logutils.Log.Errorf("validate search failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}

	// 1. 查询已分享的用户ID
	sharedUserIDs := []uint{}
	imageUserQuery := query.ImageUser
	if err := imageUserQuery.WithContext(c).
		Where(imageUserQuery.ImageID.Eq(req.ImageID)).
		Pluck(imageUserQuery.UserID, &sharedUserIDs); err != nil {
		resputil.Error(c, "query shared user ids failed", resputil.NotSpecified)
		return
	}
	sharedUserIDs = append(sharedUserIDs, util.GetToken(c).UserID)
	// 2. 查询名称包含该字符串的用户，去除已被分享的用户
	userQuery := query.User
	users, err := userQuery.WithContext(c).
		Where(userQuery.Name.Like(fmt.Sprintf("%%%s%%", req.Name))).
		Or(userQuery.Nickname.Like(fmt.Sprintf("%%%s%%", req.Name))).
		Where(userQuery.ID.NotIn(sharedUserIDs...)).
		Find()
	if err != nil {
		logutils.Log.Errorf("fetch user failed, err:%v", err)
		resputil.Error(c, "fetch user failed", resputil.NotSpecified)
		return
	}
	userInfos := []ImageGrantedUsers{}
	for _, user := range users {
		userInfos = append(userInfos, ImageGrantedUsers{
			ID:       user.ID,
			Name:     user.Name,
			Nickname: user.Nickname,
		})
	}
	resp := ImageGrantResponse{UserList: userInfos}
	resputil.Success(c, resp)
}
