package image

import (
	"errors"
	"fmt"

	"gorm.io/gen/field"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

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
//	@Description	按名称获取镜像构建详情；也兼容通过 id 查询，列表页优先使用 id
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	query	string	false	"兼容：获取ImagePack的name"
//	@Param			id	query	uint	false	"镜像构建记录 ID"
//	@Failure		400	{object}	resputil.Response[any]	"Legacy parameter or lookup failure"
//	@Router			/v1/images/getbyname [GET]
func (mgr *ImagePackMgr) GetKanikoByImagePackName(c *gin.Context) {
	var req GetKanikoRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate image get parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}
	if req.ImagePackName == "" && req.ID == 0 {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.New("either name or id is required"))
		return
	}
	kaniko, err := mgr.findCurrentUserKaniko(c, req.ImagePackName, req.ID)
	if err != nil {
		// Keep the CLI's legacy 400/40001 classification without exposing database causes.
		klog.Errorf("fetch kaniko by name failed: %v", err)
		msg := "fetch kaniko by name failed"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			msg += ", err record not found"
		}
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, msg))
		return
	}

	resputil.Success(c, mgr.buildKanikoDetailResponse(c, kaniko))
}

// GetKanikoByID returns the current user's image build detail by its stable database ID.
//
//	@Summary		获取当前用户的镜像构建详情
//	@Description	通过镜像构建记录 ID 获取详情；按名称查询保留在 getbyname 作为兼容接口
//	@Tags			ImagePack
//	@Produce		json
//	@Security		Bearer
//	@Param			id	query	uint	true	"镜像构建记录 ID"
//	@Success		200	{object}	resputil.Response[GetKanikoResponse]	"Image build details"
//	@Failure		400	{object}	resputil.Response[any]	"Invalid request parameter"
//	@Failure		404	{object}	resputil.Response[any]	"Image build not found or inaccessible"
//	@Router			/v1/images/getbyid [GET]
func (mgr *ImagePackMgr) GetKanikoByID(c *gin.Context) {
	var req GetKanikoPodRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid image build id"))
		return
	}
	kaniko, err := mgr.findCurrentUserKaniko(c, "", req.ID)
	if err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.Wrap(err, "image build not found"))
		return
	}
	resputil.Success(c, mgr.buildKanikoDetailResponse(c, kaniko))
}

// AdminGetKanikoByID returns an image build detail by its stable database ID.
//
//	@Summary		管理员获取镜像构建详情
//	@Tags			ImagePack
//	@Produce		json
//	@Security		Bearer
//	@Param			id	query	uint	true	"镜像构建记录 ID"
//	@Success		200	{object}	resputil.Response[GetKanikoResponse]	"Image build details"
//	@Failure		400	{object}	resputil.Response[any]	"Invalid request parameter"
//	@Failure		404	{object}	resputil.Response[any]	"Image build not found"
//	@Router			/v1/admin/images/getbyid [GET]
func (mgr *ImagePackMgr) AdminGetKanikoByID(c *gin.Context) {
	var req GetKanikoPodRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid image build id"))
		return
	}
	kaniko, err := query.Kaniko.WithContext(c).Where(query.Kaniko.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.Wrap(err, "image build not found"))
		return
	}
	resputil.Success(c, mgr.buildKanikoDetailResponse(c, kaniko))
}

func (mgr *ImagePackMgr) buildKanikoDetailResponse(c *gin.Context, kaniko *model.Kaniko) GetKanikoResponse {
	// The database record has already been loaded by the detail handler. Avoid
	// querying it a second time: a concurrent cleanup could otherwise produce a
	// misleading "fetch kaniko by name" error while the detail itself is valid.
	podName, podNameSpace, nodeName := mgr.getPodNameByImagePackName(c, kaniko.ImagePackName)
	description := ""
	if kaniko.Description != nil {
		description = *kaniko.Description
	}
	dockerfile := ""
	if kaniko.Dockerfile != nil {
		dockerfile = *kaniko.Dockerfile
	}
	return GetKanikoResponse{
		ID:            kaniko.ID,
		ImageLink:     kaniko.ImageLink,
		Status:        kaniko.Status,
		BuildSource:   kaniko.BuildSource,
		CreatedAt:     kaniko.CreatedAt,
		ImagePackName: kaniko.ImagePackName,
		Description:   description,
		Dockerfile:    dockerfile,
		PodName:       podName,
		PodNameSpace:  podNameSpace,
		NodeName:      nodeName,
	}
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
	var req GetKanikoRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate image get parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}
	kaniko, err := mgr.findCurrentUserKaniko(c, req.ImagePackName, 0)
	if err != nil {
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
	if _, err = mgr.findCurrentUserKaniko(c, "", req.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("permission denied to view this image build"))
			return
		}
		klog.Errorf("fetch kaniko pod failed, id: %d, err: %v", req.ID, err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "fetch kaniko pod failed"))
		return
	}
	podName, podNameSpace, nodeName := mgr.getPodName(c, req.ID)
	resp := GetKanikoPodResponse{
		PodName:      podName,
		PodNameSpace: podNameSpace,
		NodeName:     nodeName,
	}
	resputil.Success(c, resp)
}

func (mgr *ImagePackMgr) findCurrentUserKaniko(c *gin.Context, imagePackName string, id uint) (*model.Kaniko, error) {
	kanikoQuery := query.Kaniko
	q := kanikoQuery.WithContext(c).Where(kanikoQuery.UserID.Eq(util.GetToken(c).UserID))
	if imagePackName != "" {
		q = q.Where(kanikoQuery.ImagePackName.Eq(imagePackName))
	} else {
		q = q.Where(kanikoQuery.ID.Eq(id))
	}
	return q.First()
}

func (mgr *ImagePackMgr) getPodName(c *gin.Context, kanikoID uint) (name, ns, nodeName string) {
	var kaniko *model.Kaniko
	var err error
	kanikoQuery := query.Kaniko
	if kaniko, err = kanikoQuery.WithContext(c).
		Where(kanikoQuery.ID.Eq(kanikoID)).
		First(); err != nil {
		msg := fmt.Sprintf("fetch kaniko record by id failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return "", UserNameSpace, ""
	}
	return mgr.getPodNameByImagePackName(c, kaniko.ImagePackName)
}

func (mgr *ImagePackMgr) getPodNameByImagePackName(c *gin.Context, imagePackName string) (name, ns, nodeName string) {
	if imagePackName == "" {
		return "", UserNameSpace, ""
	}
	var pod *corev1.Pod
	pod, err := mgr.imagepackClient.GetImagePackPod(c, imagePackName, UserNameSpace)
	if err != nil || pod == nil {
		return "", UserNameSpace, ""
	}
	return pod.Name, UserNameSpace, pod.Spec.NodeName
}

func (mgr *ImagePackMgr) generateKanikoInfos(kanikos []*model.Kaniko) []KanikoInfo {
	kanikoInfos := []KanikoInfo{}
	for i := range kanikos {
		kaniko := kanikos[i]
		archs := kaniko.Archs.Data()
		// TODO: remove temporary fix for empty archs
		if kaniko.Archs.Data() == nil {
			archs = []string{"linux/amd64"}
		}
		description := ""
		if kaniko.Description != nil {
			description = *kaniko.Description
		}
		kanikoInfo := KanikoInfo{
			ID:          kaniko.ID,
			ImageLink:   kaniko.ImageLink,
			Status:      kaniko.Status,
			CreatedAt:   kaniko.CreatedAt,
			Size:        kaniko.Size,
			Description: description,
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
	return kanikoInfos
}

type kanikoListPageQuery struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=200"`
	Search   string `form:"search"`
	Sort     string `form:"sort"`
}

var kanikoListSortFields = map[string]string{
	"image":       "kanikos.image_link",
	"userInfo":    "users.name",
	"createdAt":   "kanikos.created_at",
	"status":      "kanikos.status",
	"size":        "kanikos.size",
	"buildSource": "kanikos.build_source",
}

var kanikoListStatuses = map[string]struct{}{
	string(model.BuildJobInitial):  {},
	string(model.BuildJobPending):  {},
	string(model.BuildJobRunning):  {},
	string(model.BuildJobFinished): {},
	string(model.BuildJobFailed):   {},
	string(model.BuildJobCanceled): {},
}

func kanikoListSortClauses(sortValue string) []string {
	return pagination.SortClauses(sortValue, kanikoListSortFields, "kanikos.id DESC")
}

func bindKanikoListPageQuery(c *gin.Context) (kanikoListPageQuery, []string, error) {
	var request kanikoListPageQuery
	if err := pagination.PrepareBoundPageQuery(
		c.ShouldBindQuery(&request), "invalid image build list query", &request.Search,
		request.Page, request.PageSize, &request.Sort, "-createdAt", kanikoListSortFields,
	); err != nil {
		return kanikoListPageQuery{}, nil, err
	}

	statuses := c.QueryArray("status")
	if err := pagination.ValidateAllowedValues(statuses, kanikoListStatuses, "image build status"); err != nil {
		return kanikoListPageQuery{}, nil, err
	}
	return request, statuses, nil
}

func (mgr *ImagePackMgr) listKanikoPage(c *gin.Context, userID *uint) {
	mgr.listKanikoPageWithDB(c, query.GetDB(), userID)
}

func (mgr *ImagePackMgr) listKanikoPageWithDB(c *gin.Context, baseDB *gorm.DB, userID *uint) {
	request, statuses, err := bindKanikoListPageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	q := query.Use(baseDB)
	k, u := q.Kaniko, q.User
	db := k.WithContext(c).LeftJoin(u, k.UserID.EqCol(u.ID)).Preload(k.User)
	if userID != nil {
		db = db.Where(k.UserID.Eq(*userID))
	}
	if len(statuses) > 0 {
		db = db.Where(k.Status.In(statuses...))
	}
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where(field.Or(k.ImageLink.Lower().Like(pattern), k.ImagePackName.Lower().Like(pattern),
			k.Description.Lower().Like(pattern), u.Name.Lower().Like(pattern), u.Nickname.Lower().Like(pattern)))
	}
	offset, limit := pagination.Bounds(c.Request.URL.Query(), request.Page, request.PageSize)
	kanikos, total, err := db.Order(pagination.SortFields(request.Sort, kanikoListSortFields, "kanikos.id DESC")...).
		FindByPage(offset, limit)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list image builds failed"))
		return
	}
	items := mgr.generateKanikoInfos(kanikos)
	if !pagination.Requested(c.Request.URL.Query()) {
		resputil.Success(c, ListKanikoResponse{KanikoInfoList: items})
		return
	}
	resputil.Success(c, resputil.NewPage(items, total, request.Page, request.PageSize))
}

// UserListKaniko returns the current user's image build records using the shared page protocol.
//
//	@Summary	分页获取当前用户的镜像制作记录
//	@Description	Supplying page or page_size returns a page; omitting both preserves the complete legacy kanikoList wrapper.
//	@Tags		ImagePack
//	@Produce	json
//	@Security	Bearer
//	@Param		page		query	int		false	"Page number"
//	@Param		page_size	query	int		false	"Page size, 1-200"
//	@Param		search		query	string	false	"Search image, description, or creator"
//	@Param		status		query	string	false	"Filter by status; repeatable"
//	@Param		sort		query	string	false	"Sort fields"
//	@Success	200			{object}	resputil.Response[resputil.Page[KanikoInfo]]
//	@Router		/v1/images/kaniko [get]
func (mgr *ImagePackMgr) UserListKaniko(c *gin.Context) {
	userID := util.GetToken(c).UserID
	mgr.listKanikoPage(c, &userID)
}

// AdminListKaniko returns all image build records using the shared page protocol.
//
//	@Summary	分页获取全部镜像制作记录
//	@Description	Supplying page or page_size returns a page; omitting both preserves the complete legacy kanikoList wrapper.
//	@Tags		ImagePack
//	@Produce	json
//	@Security	Bearer
//	@Param		page		query	int		false	"Page number"
//	@Param		page_size	query	int		false	"Page size, 1-200"
//	@Param		search		query	string	false	"Search image, description, or creator"
//	@Param		status		query	string	false	"Filter by status; repeatable"
//	@Param		sort		query	string	false	"Sort fields"
//	@Success	200			{object}	resputil.Response[resputil.Page[KanikoInfo]]
//	@Router		/v1/admin/images/kaniko [get]
func (mgr *ImagePackMgr) AdminListKaniko(c *gin.Context) {
	mgr.listKanikoPage(c, nil)
}
