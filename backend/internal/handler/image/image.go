package image

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"gorm.io/datatypes"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/klog/v2"
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
		resputil.BadRequestError(c, msg)
		return
	}
	userQuery := query.User
	user, err := userQuery.WithContext(c).Where(userQuery.ID.Eq(token.UserID)).First()
	if err != nil {
		klog.Errorf("fetch user failed, params: %+v err:%v", req, err)
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
		Archs:       datatypes.NewJSONType(req.Archs),
		ImageSource: model.ImageUploadType,
	}
	imageQuery := query.Image
	if err := imageQuery.WithContext(c).Create(imageEntity); err != nil {
		klog.Errorf("create imagepack entity failed, params: %+v", imageEntity)
	}
	resputil.Success(c, "")
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
	db, visibility := visibleImageQuery(c, false, "")
	rows := make([]visibleImageRow, 0)
	if err := db.Select("images.id, ? AS image_share_status", visibility).
		Order("images.created_at DESC, images.id DESC").Scan(&rows).Error; err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list available images failed"))
		return
	}
	items, err := mgr.imageItemsForRows(c, rows)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, ListAvailableImageResponse{Images: items})
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
		resputil.BadRequestError(c, msg)
		return
	}
	imageID := deleteImageRequest.ID
	imageQuery := query.Image
	if !mgr.requireImageOwner(c, imageID) {
		return
	}
	if _, err = imageQuery.WithContext(c).
		Where(imageQuery.ID.Eq(imageID)).
		Where(imageQuery.UserID.Eq(util.GetToken(c).UserID)).
		Delete(); err != nil {
		klog.Errorf("delete image entity failed! err:%v", err)
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
		resputil.BadRequestError(c, msg)
		return
	}
	for _, imageID := range deleteImageListRequest.IDList {
		if !mgr.requireImageOwner(c, imageID) {
			return
		}
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
		resputil.BadRequestError(c, msg)
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
		klog.Errorf("delete image entity failed! err:%v", err)
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
		klog.Errorf("validate update image public status params failed, err %v", err)
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
			klog.Errorf("create image share entity failed, err %v", err)
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
			klog.Errorf("delete image share entity failed, err %v", err)
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
		klog.Errorf("validate description failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}
	if !mgr.requireImageOwner(c, req.ID) {
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
		klog.Errorf("validate description failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
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
		klog.Errorf("update image description failed, err %v", err)
		resputil.BadRequestError(c, "update description failed")
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
		klog.Errorf("validate task type failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}
	if !mgr.requireImageOwner(c, req.ID) {
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
		klog.Errorf("validate task type failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
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
		klog.Errorf("update image task type failed, err %v", err)
		resputil.BadRequestError(c, "update task type failed")
	}
	resputil.Success(c, "")
}

func (mgr *ImagePackMgr) imageInfoFromModel(image *model.Image, status model.ImageShareType) *ImageInfo {
	archs := image.Archs.Data()
	// TODO: remove temporary fix for empty archs
	if archs == nil {
		archs = []string{"linux/amd64"}
	}
	userInfo := model.UserInfo{}
	if image.User.ID != 0 {
		userInfo.Username = image.User.Name
		userInfo.Nickname = image.User.Nickname
	}
	return &ImageInfo{
		ID:               image.ID,
		ImageLink:        image.ImageLink,
		Description:      image.Description,
		CreatedAt:        image.CreatedAt,
		IsPublic:         image.IsPublic,
		TaskType:         image.TaskType,
		UserInfo:         userInfo,
		Tags:             image.Tags.Data(),
		Archs:            archs,
		ImageBuildSource: image.ImageSource,
		ImagePackName:    image.ImagePackName,
		ImageShareStatus: status,
	}
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
		klog.Errorf("validate tags data failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}
	if !mgr.requireImageOwner(c, req.ID) {
		return
	}
	mgr.changeImageTags(c, false, req.ID, req.Tags)
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
		klog.Errorf("validate tags data failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}
	mgr.changeImageTags(c, true, req.ID, req.Tags)
}

func (mgr *ImagePackMgr) changeImageTags(c *gin.Context, isAdminMode bool, imageID uint, newTags []string) {
	imageQuery := query.Image
	specifiedQuery := imageQuery.WithContext(c)
	if !isAdminMode {
		specifiedQuery = specifiedQuery.Where(imageQuery.UserID.Eq(util.GetToken(c).UserID))
	}
	if _, err := specifiedQuery.Where(imageQuery.ID.Eq(imageID)).
		Update(imageQuery.Tags, datatypes.NewJSONType(newTags)); err != nil {
		klog.Errorf("update image tags failed, err %v", err)
		resputil.BadRequestError(c, "update tags failed")
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
		klog.Errorf("validate tags data failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}
	if !mgr.requireImageOwner(c, req.ImageID) {
		return
	}
	for _, id := range req.IDList {
		if req.Type == "user" {
			if err := mgr.createImageUserEntity(c, req.ImageID, id); err != nil {
				klog.Errorf("create image share entity failed, err %v", err)
				resputil.Error(c, fmt.Sprintf("%+v", err), resputil.NotSpecified)
				return
			}
		} else {
			if err := mgr.createImageAccountEntity(c, req.ImageID, id); err != nil {
				klog.Errorf("create image share entity failed, err %v", err)
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
		klog.Errorf("validate tags data failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}
	if !mgr.requireImageOwner(c, req.ImageID) {
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
	accountQuery := query.Account
	// check if the account exists
	if _, err := accountQuery.WithContext(c).Where(accountQuery.ID.Eq(accountID)).First(); err != nil {
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
		klog.Errorf("validate imageID failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}
	if !mgr.requireImageOwner(c, req.ImageID) {
		return
	}

	grangtedAccounts := []ImageGrantedAccounts{}
	imageAccountQuery := query.ImageAccount
	imageAccounts, err := imageAccountQuery.WithContext(c).
		Preload(imageAccountQuery.Account).
		Where(imageAccountQuery.ImageID.Eq(req.ImageID)).
		Find()
	if err != nil {
		klog.Errorf("fetch image share with account failed, err:%v", err)
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
		klog.Errorf("fetch image share with user failed, err:%v", err)
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
		klog.Errorf("validate search failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}
	if !mgr.requireImageOwner(c, req.ImageID) {
		return
	}
	// 1. 查询已分享的AccountID
	sharedAccountIDs := []uint{}
	imageAccountQuery := query.ImageAccount
	if err := imageAccountQuery.WithContext(c).
		Where(imageAccountQuery.ImageID.Eq(req.ImageID)).
		Pluck(imageAccountQuery.AccountID, &sharedAccountIDs); err != nil {
		klog.Errorf("query shared account ids failed, err:%v", err)
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
		klog.Errorf("get user current account failed, err:%v", err)
		resputil.Error(c, "get user current account failed", resputil.NotSpecified)
	}
	// 4. 查询未被分享的Account
	accountQuery := query.Account
	accounts, err := accountQuery.WithContext(c).
		Where(accountQuery.ID.In(accountIDs...)).
		Where(accountQuery.ID.NotIn(sharedAccountIDs...)).
		Find()
	if err != nil {
		klog.Errorf("fetch account failed, err:%v", err)
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
		klog.Errorf("validate search failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}
	if !mgr.requireImageOwner(c, req.ImageID) {
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
		klog.Errorf("fetch user failed, err:%v", err)
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

// UserGetCudaBaseImages godoc
//
//	@Summary		获取所有Cuda基础镜像
//	@Description	获取所有可用的Cuda基础镜像列表
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/cudabaseimage [GET]
func (mgr *ImagePackMgr) UserGetCudaBaseImages(c *gin.Context) {
	// 查询所有 CudaBaseImage 数据
	cudaBaseImageQuery := query.CudaBaseImage
	cudaBaseImages, err := cudaBaseImageQuery.WithContext(c).Find()
	if err != nil {
		klog.Errorf("fetch cuda base images failed, err:%v", err)
		resputil.Error(c, "fetch cuda base images failed", resputil.NotSpecified)
		return
	}

	// 转换为值类型切片
	cudaBaseImageList := []CudaBaseImage{}
	for _, img := range cudaBaseImages {
		cudaBaseImageList = append(cudaBaseImageList, CudaBaseImage{
			ID:         img.ID,
			Label:      img.Label,
			ImageLabel: img.ImageLabel,
			Value:      img.Value,
		})
	}

	// 构造响应
	resp := CudaBaseImagesResponse{
		CudaBaseImages: cudaBaseImageList,
	}
	resputil.Success(c, resp)
}

// AdminAddCudaBaseImage godoc
//
//	@Summary		添加CUDA基础镜像
//	@Description	添加新的CUDA基础镜像到系统中
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body	CudaBaseImageCreateRequest	true	"CUDA基础镜像信息"
//	@Router			/v1/admin/images/cudabaseimage [POST]
func (mgr *ImagePackMgr) AdminAddCudaBaseImage(c *gin.Context) {
	req := &CudaBaseImageCreateRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		klog.Errorf("validate cuda base image create request failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}

	// 检查 ImageLabel 是否已存在（由于有唯一索引）
	cudaBaseImageQuery := query.CudaBaseImage
	existingImage, _ := cudaBaseImageQuery.WithContext(c).
		Where(cudaBaseImageQuery.ImageLabel.Eq(req.ImageLabel)).
		First()
	if existingImage != nil {
		klog.Errorf("cuda base image with imageLabel %s already exists", req.ImageLabel)
		resputil.Error(c, "image label already exists", resputil.NotSpecified)
		return
	}

	// 创建新的 CudaBaseImage
	newImage := &model.CudaBaseImage{
		Label:      req.Label,
		ImageLabel: req.ImageLabel,
		Value:      req.Value,
	}

	if err := cudaBaseImageQuery.WithContext(c).Create(newImage); err != nil {
		klog.Errorf("create cuda base image failed, err:%v", err)
		resputil.Error(c, "create cuda base image failed", resputil.NotSpecified)
		return
	}

	klog.Infof("cuda base image created successfully, id: %d, imageLabel: %s", newImage.ID, newImage.ImageLabel)
	resputil.Success(c, "cuda base image created successfully")
}

// AdminDeleteCudaBaseImage godoc
//
//	@Summary		删除CUDA基础镜像
//	@Description	根据ID删除指定的CUDA基础镜像
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path	uint	true	"CUDA基础镜像ID"
//	@Router			/v1/admin/images/cudabaseimage/{id} [DELETE]
func (mgr *ImagePackMgr) AdminDeleteCudaBaseImage(c *gin.Context) {
	req := &CudaBaseImageDeleteRequest{}
	if err := c.ShouldBindUri(req); err != nil {
		klog.Errorf("validate cuda base image delete request failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}

	// 查询要删除的镜像是否存在
	cudaBaseImageQuery := query.CudaBaseImage
	existingImage, err := cudaBaseImageQuery.WithContext(c).
		Where(cudaBaseImageQuery.ID.Eq(req.ID)).
		First()
	if err != nil {
		klog.Errorf("cuda base image with id %d not found, err:%v", req.ID, err)
		resputil.Error(c, "cuda base image not found", resputil.NotSpecified)
		return
	}

	// 真正删除镜像数据（硬删除），而不是软删除
	if _, err := cudaBaseImageQuery.WithContext(c).
		Where(cudaBaseImageQuery.ID.Eq(req.ID)).
		Unscoped().Delete(); err != nil {
		klog.Errorf("delete cuda base image failed, id: %d, err:%v", req.ID, err)
		resputil.Error(c, "delete cuda base image failed", resputil.NotSpecified)
		return
	}

	klog.Infof("cuda base image deleted successfully, id: %d, imageLabel: %s", existingImage.ID, existingImage.ImageLabel)
	resputil.Success(c, "cuda base image deleted successfully")
}

// UserUpdateImageArch godoc
//
//	@Summary		更新镜像架构
//	@Description	根据镜像ID更新镜像的架构列表
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body	UpdateImageArchRequest	true	"更新镜像架构信息"
//	@Router			/v1/images/arch [POST]
func (mgr *ImagePackMgr) UserUpdateImageArch(c *gin.Context) {
	req := &UpdateImageArchRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		klog.Errorf("validate update image arch request failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}
	if !mgr.requireImageOwner(c, req.ID) {
		return
	}

	mgr.updateImageArch(c, false, req.ID, req.Archs)
}

// AdminUpdateImageArch godoc
//
//	@Summary		管理员更新镜像架构
//	@Description	管理员模式下根据镜像ID更新镜像的架构列表
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body	UpdateImageArchRequest	true	"更新镜像架构信息"
//	@Router			/v1/admin/images/arch [POST]
func (mgr *ImagePackMgr) AdminUpdateImageArch(c *gin.Context) {
	req := &UpdateImageArchRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		klog.Errorf("validate update image arch request failed, err %v", err)
		resputil.BadRequestError(c, "validate failed")
		return
	}

	mgr.updateImageArch(c, true, req.ID, req.Archs)
}

func (mgr *ImagePackMgr) updateImageArch(c *gin.Context, isAdminMode bool, imageID uint, newArchs []string) {
	imageQuery := query.Image
	specifiedQuery := imageQuery.WithContext(c).Preload(imageQuery.User)

	if !isAdminMode {
		specifiedQuery = specifiedQuery.Where(imageQuery.UserID.Eq(util.GetToken(c).UserID))
	}

	// 更新镜像的架构
	archsJSON := datatypes.NewJSONType(newArchs)

	if _, err := specifiedQuery.
		Where(imageQuery.ID.Eq(imageID)).
		Update(imageQuery.Archs, archsJSON); err != nil {
		klog.Errorf("update image archs failed, id: %d, err:%v", imageID, err)
		resputil.Error(c, "update image archs failed", resputil.NotSpecified)
		return
	}

	klog.Infof("image archs updated successfully, id: %d, new archs: %v", imageID, newArchs)
	resputil.Success(c, "image archs updated successfully")
}

func (mgr *ImagePackMgr) requireImageOwner(c *gin.Context, imageID uint) bool {
	imageQuery := query.Image
	if _, err := imageQuery.WithContext(c).
		Where(imageQuery.ID.Eq(imageID)).
		Where(imageQuery.UserID.Eq(util.GetToken(c).UserID)).
		First(); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			klog.Warningf("permission denied or image not found, imageID: %d", imageID)
			resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("permission denied to manage this image"))
			return false
		}
		klog.Errorf("check image owner failed, imageID: %d, err: %v", imageID, err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "check image owner failed"))
		return false
	}
	return true
}

type imageListPageQuery struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=200"`
	Search   string `form:"search"`
	Sort     string `form:"sort"`
}

var imageListSortFields = map[string]string{
	"image":            "images.image_link",
	"userInfo":         "users.name",
	"createdAt":        "images.created_at",
	"imageShareStatus": "image_share_status",
}

var imageListShareStatuses = map[string]struct{}{
	string(model.Public):       {},
	string(model.Private):      {},
	string(model.UserShare):    {},
	string(model.AccountShare): {},
}

func imageListSortClauses(sortValue string) []string {
	return pagination.SortClauses(sortValue, imageListSortFields, "images.id DESC")
}

func bindImageListPageQuery(c *gin.Context) (imageListPageQuery, []string, error) {
	var request imageListPageQuery
	if err := pagination.PrepareBoundPageQuery(
		c.ShouldBindQuery(&request), "invalid image list query", &request.Search,
		request.Page, request.PageSize, &request.Sort, "-createdAt", imageListSortFields,
	); err != nil {
		return imageListPageQuery{}, nil, err
	}

	shareStatuses := c.QueryArray("imageShareStatus")
	if err := pagination.ValidateAllowedValues(shareStatuses, imageListShareStatuses, "image visibility"); err != nil {
		return imageListPageQuery{}, nil, err
	}
	return request, shareStatuses, nil
}

func imageVisibilityExpression(ctx context.Context, userID, accountID uint, isAdmin bool) clause.Expr {
	i, ia, iu := query.Image, query.ImageAccount, query.ImageUser
	accountImages := func(account uint) field.Expr {
		subquery := ia.WithContext(ctx).Select(ia.ImageID).Where(ia.AccountID.Eq(account)).UnderlyingDB()
		return field.ContainsSubQuery([]field.Expr{i.ID}, subquery)
	}
	public := field.Or(i.IsPublic.Is(true), accountImages(model.DefaultAccountID))
	if isAdmin {
		return clause.Expr{SQL: "CASE WHEN ? THEN ? ELSE ? END", Vars: []any{public.RawExpr(), string(model.Public), string(model.Private)}}
	}
	userImages := iu.WithContext(ctx).Select(iu.ImageID).Where(iu.UserID.Eq(userID)).UnderlyingDB()
	// Only the derived visibility label needs CASE. All IDs, labels and association
	// subqueries are bound by GORM; there is no formatted or concatenated SQL.
	return clause.Expr{
		SQL: "CASE WHEN ? THEN ? WHEN ? THEN ? WHEN ? THEN ? WHEN ? THEN ? ELSE NULL END",
		Vars: []any{public.RawExpr(), string(model.Public), accountImages(accountID).RawExpr(), string(model.AccountShare),
			i.UserID.Eq(userID).RawExpr(), string(model.Private),
			field.ContainsSubQuery([]field.Expr{i.ID}, userImages).RawExpr(), string(model.UserShare)},
	}
}

type visibleImageRow struct {
	ID               uint                 `gorm:"column:id"`
	ImageShareStatus model.ImageShareType `gorm:"column:image_share_status"`
}

func imageBuildIDsByImagePackName(kanikos []*model.Kaniko) map[string]uint {
	buildIDs := make(map[string]uint, len(kanikos))
	for _, kaniko := range kanikos {
		buildIDs[kaniko.ImagePackName] = kaniko.ID
	}
	return buildIDs
}

func imageBuildIDsForImages(c *gin.Context, images []*model.Image) (map[string]uint, error) {
	imagePackNames := make([]string, 0, len(images))
	for _, image := range images {
		if image.ImageSource == model.ImageCreateType && image.ImagePackName != nil && *image.ImagePackName != "" {
			imagePackNames = append(imagePackNames, *image.ImagePackName)
		}
	}
	if len(imagePackNames) == 0 {
		return map[string]uint{}, nil
	}

	kanikoQuery := query.Kaniko
	kanikos, err := kanikoQuery.WithContext(c).
		Select(kanikoQuery.ID, kanikoQuery.ImagePackName).
		Where(kanikoQuery.ImagePackName.In(imagePackNames...)).
		Find()
	if err != nil {
		return nil, err
	}
	return imageBuildIDsByImagePackName(kanikos), nil
}

func (mgr *ImagePackMgr) imageItemsForRows(c *gin.Context, rows []visibleImageRow) ([]*ImageInfo, error) {
	if len(rows) == 0 {
		return []*ImageInfo{}, nil
	}
	ids := make([]uint, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	imageQuery := query.Image
	images, err := imageQuery.WithContext(c).
		Preload(imageQuery.User).
		Where(imageQuery.ID.In(ids...)).
		Find()
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "load visible images failed")
	}
	imagesByID := make(map[uint]*model.Image, len(images))
	for _, image := range images {
		imagesByID[image.ID] = image
	}
	buildIDsByImagePackName, err := imageBuildIDsForImages(c, images)
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "load image build IDs failed")
	}

	items := make([]*ImageInfo, 0, len(rows))
	for _, row := range rows {
		image := imagesByID[row.ID]
		if image == nil {
			continue
		}
		item := mgr.imageInfoFromModel(image, row.ImageShareStatus)
		if image.ImageSource == model.ImageCreateType && image.ImagePackName != nil {
			if buildID, ok := buildIDsByImagePackName[*image.ImagePackName]; ok {
				item.ImageBuildID = &buildID
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func (mgr *ImagePackMgr) listImagePage(c *gin.Context, isAdmin bool) {
	request, shareStatuses, err := bindImageListPageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	db, visibility := visibleImageQuery(c, isAdmin, request.Search)
	if len(shareStatuses) > 0 {
		db = db.Where(clause.Expr{SQL: "(?) IN ?", Vars: []any{visibility, shareStatuses}})
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count visible images failed"))
		return
	}
	for _, order := range imageListSortClauses(request.Sort) {
		db = db.Order(order)
	}
	rows := make([]visibleImageRow, 0, request.PageSize)
	if err := db.Select("images.id, ? AS image_share_status", visibility).
		Offset((request.Page - 1) * request.PageSize).Limit(request.PageSize).Scan(&rows).Error; err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list visible images failed"))
		return
	}
	items, err := mgr.imageItemsForRows(c, rows)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, resputil.NewPage(items, total, request.Page, request.PageSize))
}

// UserListImage returns images visible to the current user using the shared page protocol.
//
//	@Summary	分页获取当前用户可见镜像
//	@Tags		ImagePack
//	@Produce	json
//	@Security	Bearer
//	@Param		page				query	int		false	"Page number"
//	@Param		page_size			query	int		false	"Page size, 1-200"
//	@Param		search				query	string	false	"Search image, description, or creator"
//	@Param		imageShareStatus	query	string	false	"Filter by visibility; repeatable"
//	@Param		sort				query	string	false	"Sort fields"
//	@Success	200					{object}	resputil.Response[resputil.Page[ImageInfo]]
//	@Router		/v1/images/image [get]
func (mgr *ImagePackMgr) UserListImage(c *gin.Context) {
	mgr.listImagePage(c, false)
}

// AdminListImage returns all images using the shared page protocol.
//
//	@Summary	分页获取全部镜像
//	@Tags		ImagePack
//	@Produce	json
//	@Security	Bearer
//	@Param		page				query	int		false	"Page number"
//	@Param		page_size			query	int		false	"Page size, 1-200"
//	@Param		search				query	string	false	"Search image, description, or creator"
//	@Param		imageShareStatus	query	string	false	"Filter by visibility; repeatable"
//	@Param		sort				query	string	false	"Sort fields"
//	@Success	200					{object}	resputil.Response[resputil.Page[ImageInfo]]
//	@Router		/v1/admin/images/image [get]
func (mgr *ImagePackMgr) AdminListImage(c *gin.Context) {
	mgr.listImagePage(c, true)
}

// visibleImageQuery is shared by management tables and the complete job-image selector.
func visibleImageQuery(c *gin.Context, isAdmin bool, search string) (*gorm.DB, clause.Expr) {
	i, u := query.Image, query.User
	token := util.GetToken(c)
	visibility := imageVisibilityExpression(c, token.UserID, token.AccountID, isAdmin)
	db := i.WithContext(c).LeftJoin(u, i.UserID.EqCol(u.ID)).Where(i.ImageLink.Neq(""))
	if search != "" {
		pattern := util.ContainsPattern(search)
		db = db.Where(field.Or(i.ImageLink.Lower().Like(pattern), i.Description.Lower().Like(pattern),
			i.ImagePackName.Lower().Like(pattern), u.Name.Lower().Like(pattern), u.Nickname.Lower().Like(pattern)))
	}
	result := db.UnderlyingDB()
	if !isAdmin {
		result = result.Where(clause.Expr{SQL: "(?) IS NOT NULL", Vars: []any{visibility}})
	}
	return result, visibility
}
