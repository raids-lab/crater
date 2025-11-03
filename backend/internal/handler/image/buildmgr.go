package image

import (
	"fmt"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

// UserListKaniko godoc
//
//	@Summary		用户获取镜像构建信息
//	@Description	返回该用户所有的镜像构建数据
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/images/kaniko [GET]
func (mgr *ImagePackMgr) UserListKaniko(c *gin.Context) {
	var kanikos []*model.Kaniko
	var err error
	token := util.GetToken(c)
	k := query.Kaniko
	kanikos, err = k.WithContext(c).
		Where(k.UserID.Eq(token.UserID)).
		Preload(k.User).
		Order(k.CreatedAt.Desc()).
		Find()
	if err != nil {
		klog.Errorf("fetch kaniko entity failed, err:%v", err)
	}
	response := mgr.generateKanikoListResponse(kanikos)
	resputil.Success(c, response)
}

// AdminListKaniko godoc
//
//	@Summary		管理员获取相关镜像的功能
//	@Description	管理员获取所有镜像制作信息
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Router			/v1/admin/images/kaniko [GET]
func (mgr *ImagePackMgr) AdminListKaniko(c *gin.Context) {
	var kanikos []*model.Kaniko
	var err error
	kanikoQuery := query.Kaniko
	kanikos, err = kanikoQuery.WithContext(c).Preload(kanikoQuery.User).Order(kanikoQuery.CreatedAt.Desc()).Find()
	if err != nil {
		klog.Errorf("fetch kaniko entity failed, err:%v", err)
	}
	response := mgr.generateKanikoListResponse(kanikos)
	resputil.Success(c, response)
}

// UserRemoveKanikoByID godoc
//
//	@Summary		删除或取消镜像制作任务（批量）
//	@Description	根据任务状态智能处理：若任务状态为 Finished/Failed/Canceled 则执行删除操作；否则执行取消操作
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			idList	body	[]uint	true	"镜像构建任务ID列表"
//	@Router			/v1/images/remove [POST]
func (mgr *ImagePackMgr) UserRemoveKanikoByID(c *gin.Context) {
	var err error
	var removeKanikoListRequest DeleteKanikoByIDListRequest
	if err = c.ShouldBindJSON(&removeKanikoListRequest); err != nil {
		msg := fmt.Sprintf("validate remove parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}
	flag := mgr.removeKanikoByIDList(c, false, removeKanikoListRequest.IDList)
	if flag {
		resputil.Success(c, "")
	} else {
		resputil.Error(c, "failed to remove kaniko", resputil.NotSpecified)
	}
}

// AdminRemoveKanikoByIDList godoc
//
//	@Summary		管理员模式下删除或取消镜像制作任务（批量）
//	@Description	管理员模式下根据任务状态智能处理：若任务状态为 Finished/Failed/Canceled 则执行删除操作；否则执行取消操作
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			idList	body	[]uint	true	"镜像构建任务ID列表"
//	@Router			/v1/admin/images/remove [POST]
func (mgr *ImagePackMgr) AdminRemoveKanikoByIDList(c *gin.Context) {
	var err error
	var removeKanikoListRequest DeleteKanikoByIDListRequest
	if err = c.ShouldBindJSON(&removeKanikoListRequest); err != nil {
		msg := fmt.Sprintf("validate remove parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}
	flag := mgr.removeKanikoByIDList(c, true, removeKanikoListRequest.IDList)
	if flag {
		resputil.Success(c, "")
	} else {
		resputil.Success(c, "remove kaniko encountered some errors")
	}
}

func (mgr *ImagePackMgr) removeKanikoByIDList(c *gin.Context, isAdminMode bool, kanikoIDList []uint) (isSuccess bool) {
	flag := true
	userID := util.GetToken(c).UserID
	for _, kanikoID := range kanikoIDList {
		if isSuccess, errorMsg := mgr.removeKanikoByID(c, isAdminMode, kanikoID, userID); !isSuccess {
			flag = false
			klog.Errorf("remove kaniko failed, err:%v", errorMsg)
		}
	}
	return flag
}

// removeKanikoByID 统一的删除/取消接口：根据任务状态自动判断执行删除还是取消
func (mgr *ImagePackMgr) removeKanikoByID(c *gin.Context, isAdminMode bool, kanikoID, userID uint) (isSuccess bool, msg string) {
	k := query.Kaniko

	// 构造查询并进行权限过滤
	kanikoQuery := k.WithContext(c)
	if !isAdminMode {
		kanikoQuery = kanikoQuery.Where(k.UserID.Eq(userID))
	}

	// 获取 Kaniko 记录
	kaniko, err := kanikoQuery.Where(k.ID.Eq(kanikoID)).First()
	if err != nil || kaniko == nil {
		msg := fmt.Sprintf("kaniko not exist or have no permission %+v", err)
		klog.Error(msg)
		return false, msg
	}

	// 根据状态判断执行删除还是取消
	if kaniko.Status == model.BuildJobFinished ||
		kaniko.Status == model.BuildJobFailed ||
		kaniko.Status == model.BuildJobCanceled {
		// 任务已结束，执行删除操作
		return mgr.deleteKanikoByID(c, isAdminMode, kanikoID, userID)
	}

	// 任务进行中，执行取消操作
	// 删除对应的 Job
	if err := mgr.imagePacker.DeleteJob(c, kaniko.ImagePackName, UserNameSpace); err != nil {
		klog.Errorf("delete kaniko job failed! err:%v", err)
	}

	return true, ""
}

func (mgr *ImagePackMgr) deleteKanikoByID(c *gin.Context, isAdminMode bool, kanikoID, userID uint) (isSuccess bool, errMsg string) {
	var kaniko *model.Kaniko
	var err error
	var errorMsg string
	flag := true
	k := query.Kaniko
	// 0. specify user & admin query
	kanikoQuery := k.WithContext(c)
	if !isAdminMode {
		kanikoQuery = kanikoQuery.Where(k.UserID.Eq(userID))
	}
	// 1. check if kaniko exist and have permission
	if kaniko, err = kanikoQuery.Where(k.ID.Eq(kanikoID)).First(); err != nil {
		errorMsg = fmt.Sprintf("kaniko not exist or have no permission %+v", err)
		klog.Error(errorMsg)
		return false, errorMsg
	}
	// 2. if kaniko is finished, delete image entity
	if kaniko.Status == model.BuildJobFinished {
		if _, err = query.Image.WithContext(c).Where(query.Image.ImagePackName.Eq(kaniko.ImagePackName)).Delete(); err != nil {
			errorMsg = fmt.Sprintf("delete image entity failed! err:%v", err)
			klog.Error(errorMsg)
		}
	}
	// 3. delete kaniko entity from database
	if _, err = k.WithContext(c).Where(k.ID.Eq(kanikoID)).Delete(); err != nil {
		errorMsg = fmt.Sprintf("delete kaniko entity failed! err:%v", err)
		klog.Error(errorMsg)
		return false, errorMsg
	}
	// 4. if buildkit finished, then delete image from harbor
	if kaniko.Status == model.BuildJobFinished {
		if err = mgr.imageRegistry.DeleteImageFromProject(c, kaniko.ImageLink); err != nil {
			errorMsg = fmt.Sprintf("delete imagepack artifact failed! err:%+v", err)
			klog.Error(errorMsg)
		}
	}
	return flag, errorMsg
}

// GetKanikoByImagePackName godoc
//
//	@Summary		获取imagepack的详细信息
//	@Description	获取imagepackname，搜索到imagepack
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	query	string	true	"获取ImagePack的name"
//	@Router			/v1/images/getbyname [GET]
func (mgr *ImagePackMgr) GetKanikoByImagePackName(c *gin.Context) {
	kanikoQuery := query.Kaniko
	var req GetKanikoRequest
	var err error
	if err = c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate image get parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}
	var kaniko *model.Kaniko
	if kaniko, err = kanikoQuery.WithContext(c).
		Where(kanikoQuery.ImagePackName.Eq(req.ImagePackName)).
		First(); err != nil {
		msg := fmt.Sprintf("fetch kaniko by name failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}

	podName, podNameSpace := mgr.getPodName(c, kaniko.ID)
	getKanikoResponse := GetKanikoResponse{
		ID:            kaniko.ID,
		ImageLink:     kaniko.ImageLink,
		Status:        kaniko.Status,
		BuildSource:   kaniko.BuildSource,
		CreatedAt:     kaniko.CreatedAt,
		ImagePackName: kaniko.ImagePackName,
		Description:   *kaniko.Description,
		Dockerfile:    *kaniko.Dockerfile,
		PodName:       podName,
		PodNameSpace:  podNameSpace,
	}
	resputil.Success(c, getKanikoResponse)
}

// GetKanikoTemplateByImagePackName godoc
//
//	@Summary		获取imagepack的模板信息
//	@Description	获取imagepackname，搜索到imagepack的模板信息
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	query	string	true	"获取ImagePack的name"
//	@Router			/v1/images/get [GET]
func (mgr *ImagePackMgr) GetKanikoTemplateByImagePackName(c *gin.Context) {
	kanikoQuery := query.Kaniko
	var req GetKanikoRequest
	var err error
	if err = c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate image get parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}
	var kaniko *model.Kaniko
	if kaniko, err = kanikoQuery.WithContext(c).
		Where(kanikoQuery.ImagePackName.Eq(req.ImagePackName)).
		First(); err != nil {
		msg := fmt.Sprintf("fetch kaniko by name failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}

	resputil.Success(c, kaniko.Template)
}

// GetImagepackPodName godoc
//
//	@Summary		获取镜像构建Pod名称
//	@Description	根据ID获取镜像构建Pod名称和命名空间
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	query	uint	true	"镜像构建任务ID"
//	@Router			/v1/images/podname [GET]
func (mgr *ImagePackMgr) GetImagepackPodName(c *gin.Context) {
	var req GetKanikoPodRequest
	var err error
	if err = c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate image get parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
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
		resputil.BadRequestError(c, msg)
		return "", UserNameSpace
	}
	var pod *corev1.Pod
	pod, err = mgr.imagepackClient.GetImagePackPod(c, kaniko.ImagePackName, UserNameSpace)
	if err != nil || pod == nil {
		return "", UserNameSpace
	}
	return pod.Name, UserNameSpace
}

func (mgr *ImagePackMgr) generateKanikoListResponse(kanikos []*model.Kaniko) ListKanikoResponse {
	kanikoInfos := []KanikoInfo{}
	for i := range kanikos {
		kaniko := kanikos[i]
		archs := kaniko.Archs.Data()
		// TODO: remove temporary fix for empty archs
		if kaniko.Archs.Data() == nil {
			archs = []string{"linux/amd64"}
		}
		kanikoInfo := KanikoInfo{
			ID:          kaniko.ID,
			ImageLink:   kaniko.ImageLink,
			Status:      kaniko.Status,
			CreatedAt:   kaniko.CreatedAt,
			Size:        kaniko.Size,
			Description: *kaniko.Description,
			UserInfo: model.UserInfo{
				Username: kaniko.User.Name,
				Nickname: kaniko.User.Nickname,
			},
			Tags:          kaniko.Tags.Data(),
			Archs:         archs,
			ImagePackName: kaniko.ImagePackName,
			BuildSource:   kaniko.BuildSource,
		}
		kanikoInfos = append(kanikoInfos, kanikoInfo)
	}
	return ListKanikoResponse{
		KanikoInfoList: kanikoInfos,
	}
}
