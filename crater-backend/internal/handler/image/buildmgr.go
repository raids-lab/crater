package image

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/logutils"
)

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
	k := query.Kaniko
	kanikos, err = k.WithContext(c).
		Where(k.UserID.Eq(token.UserID)).
		Preload(k.User).
		Order(k.CreatedAt.Desc()).
		Find()
	if err != nil {
		logutils.Log.Errorf("fetch kaniko entity failed, err:%v", err)
	}
	response := mgr.generateKanikoListResponse(kanikos)
	resputil.Success(c, response)
}

// AdminListKaniko godoc
// @Summary 管理员获取相关镜像的功能
// @Description 管理员获取所有镜像制作信息
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/admin/images/kaniko [GET]
func (mgr *ImagePackMgr) AdminListKaniko(c *gin.Context) {
	var kanikos []*model.Kaniko
	var err error
	kanikoQuery := query.Kaniko
	kanikos, err = kanikoQuery.WithContext(c).Preload(kanikoQuery.User).Order(kanikoQuery.CreatedAt.Desc()).Find()
	if err != nil {
		logutils.Log.Errorf("fetch kaniko entity failed, err:%v", err)
	}
	response := mgr.generateKanikoListResponse(kanikos)
	resputil.Success(c, response)
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
	if isSuccess, errorMsg := mgr.deleteKanikoByID(c, false, kanikoID, token.UserID); isSuccess {
		resputil.Success(c, "")
	} else {
		resputil.Error(c, errorMsg, resputil.NotSpecified)
	}
}

// UserDeleteKanikoByIDList godoc
// @Summary 根据IDList删除Kaniko entity
// @Description 遍历列表，根据ID更新Kaniko的状态为Deleted，起到删除的功能
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param IDList body []uint true "删除kaniko的IDList"
// @Router /v1/images/deletekaniko [POST]
func (mgr *ImagePackMgr) UserDeleteKanikoByIDList(c *gin.Context) {
	var err error
	var deleteKanikoListRequest DeleteKanikoByIDListRequest
	if err = c.ShouldBindJSON(&deleteKanikoListRequest); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	flag := mgr.deleteKanikoByIDList(c, true, deleteKanikoListRequest.IDList)
	if flag {
		resputil.Success(c, "")
	} else {
		resputil.Error(c, "failed to delete kaniko", resputil.NotSpecified)
	}
}

// AdminDeleteKanikoByIDList godoc
// @Summary 管理员模式下根据IDList删除Kaniko entity
// @Description 管理员模式下遍历列表，根据ID更新Kaniko的状态为Deleted，起到删除的功能
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param IDList body []uint true "删除kaniko的IDList"
// @Router /v1/admin/images/deletekaniko [POST]
func (mgr *ImagePackMgr) AdminDeleteKanikoByIDList(c *gin.Context) {
	var err error
	var deleteKanikoListRequest DeleteKanikoByIDListRequest
	if err = c.ShouldBindJSON(&deleteKanikoListRequest); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	flag := mgr.deleteKanikoByIDList(c, true, deleteKanikoListRequest.IDList)
	if flag {
		resputil.Success(c, "")
	} else {
		resputil.Error(c, "failed to delete kaniko", resputil.NotSpecified)
	}
}

func (mgr *ImagePackMgr) deleteKanikoByIDList(c *gin.Context, isAdminMode bool, kanikoIDList []uint) (isSuccess bool) {
	flag := true
	userID := util.GetToken(c).UserID
	for _, kanikoID := range kanikoIDList {
		if isSuccess, errorMsg := mgr.deleteKanikoByID(c, isAdminMode, kanikoID, userID); !isSuccess {
			flag = false
			logutils.Log.Errorf("delete kaniko failed, err:%v", errorMsg)
		}
	}
	return flag
}

func (mgr *ImagePackMgr) deleteKanikoByID(c *gin.Context, isAdminMode bool, kanikoID, userID uint) (isSuccess bool, errMsg string) {
	var kaniko *model.Kaniko
	var err error
	var errorMsg string
	flag := true
	k := query.Kaniko
	// 0. specify user & admin query
	kanikoQuery := k.WithContext(c).Preload(k.User)
	if !isAdminMode {
		kanikoQuery = kanikoQuery.Where(k.UserID.Eq(userID))
	}
	// 1. check if kaniko exist and have permission
	if kaniko, err = k.WithContext(c).
		Where(k.ID.Eq(kanikoID)).First(); err != nil {
		errorMsg = fmt.Sprintf("kaniko not exist or have no permission %+v", err)
		logutils.Log.Error(errorMsg)
		return false, errorMsg
	}
	// 2. if kaniko is finished, delete image entity
	if kaniko.Status == model.BuildJobFinished {
		if _, err = query.Image.WithContext(c).Where(query.Image.ImagePackName.Eq(kaniko.ImagePackName)).Delete(); err != nil {
			errorMsg = fmt.Sprintf("delete image entity failed! err:%v", err)
			logutils.Log.Error(errorMsg)
		}
	}
	// 3. delete kaniko entity
	if _, err = kanikoQuery.Delete(kaniko); err != nil {
		errorMsg = fmt.Sprintf("delete kaniko entity failed! err:%v", err)
		logutils.Log.Error(errorMsg)
		return false, errorMsg
	}
	// 4. delete kaniko job
	if err = mgr.imagePacker.DeleteJob(c, kaniko.ImagePackName, UserNameSpace); err != nil {
		errorMsg = fmt.Sprintf("delete kaniko job failed! err:%v", err)
		logutils.Log.Error(errorMsg)
	}
	// 5. if buildkit finished, then delete image from harbor
	if kaniko.Status == model.BuildJobFinished {
		if err = mgr.imageRegistry.DeleteImageFromProject(c, kaniko.ImageLink); err != nil {
			errorMsg = fmt.Sprintf("delete imagepack artifact failed! err:%+v", err)
			logutils.Log.Error(errorMsg)
		}
	}
	return flag, errorMsg
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

// GetImagepackPodName godoc
// @Summary 获取镜像构建Pod名称
// @Description 根据ID获取镜像构建Pod名称和命名空间
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param id query uint true "镜像构建任务ID"
// @Router /v1/images/podname [GET]
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

func (mgr *ImagePackMgr) generateKanikoListResponse(kanikos []*model.Kaniko) ListKanikoResponse {
	kanikoInfos := []KanikoInfo{}
	for i := range kanikos {
		kaniko := kanikos[i]
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
			Tags: kaniko.Tags.Data(),
		}
		kanikoInfos = append(kanikoInfos, kanikoInfo)
	}
	return ListKanikoResponse{
		KanikoInfoList: kanikoInfos,
	}
}
