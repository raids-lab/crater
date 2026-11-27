package image

import (
	"errors"
	"fmt"

	"gorm.io/gen/field"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

// UserRemoveKanikoByID godoc
//
//	@Summary		删除或取消镜像制作任务（批量）
//	@Description	Cancel active builds or delete terminal builds. Omit action to choose by status.
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body	DeleteKanikoByIDListRequest	true	"Build IDs and optional action"
//	@Failure		409	{object}	resputil.Response[any]
//	@Router			/v1/images/remove [POST]
func (mgr *ImagePackMgr) UserRemoveKanikoByID(c *gin.Context) {
	mgr.handleImageBuildRemoval(c, false)
}

// AdminRemoveKanikoByIDList godoc
//
//	@Summary		管理员模式下删除或取消镜像制作任务（批量）
//	@Description	Cancel active builds or delete terminal builds as an administrator. Omit action to choose by status.
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body	DeleteKanikoByIDListRequest	true	"Build IDs and optional action"
//	@Failure		409	{object}	resputil.Response[any]
//	@Router			/v1/admin/images/remove [POST]
func (mgr *ImagePackMgr) AdminRemoveKanikoByIDList(c *gin.Context) {
	mgr.handleImageBuildRemoval(c, true)
}

type imageBuildRemovalAction string

const (
	cancelImageBuild imageBuildRemovalAction = "cancel"
	deleteImageBuild imageBuildRemovalAction = "delete"
)

func (mgr *ImagePackMgr) handleImageBuildRemoval(c *gin.Context, isAdminMode bool) {
	var req DeleteKanikoByIDListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "Provide build IDs and an optional cancel or delete action."))
		return
	}
	if req.Action != "" {
		if err := mgr.applyImageBuildRemoval(c, isAdminMode, req); err != nil {
			resputil.HandleError(c, err)
			return
		}
		resputil.Success(c, "")
		return
	}
	if mgr.removeKanikoByIDList(c, isAdminMode, req.IDList) {
		resputil.Success(c, "")
	} else if isAdminMode {
		resputil.Success(c, "remove kaniko encountered some errors")
	} else {
		resputil.HandleError(c, bizerr.Internal.ServiceError.New("Failed to remove image builds."))
	}
}

func (mgr *ImagePackMgr) applyImageBuildRemoval(c *gin.Context, isAdminMode bool, req DeleteKanikoByIDListRequest) error {
	if len(req.IDList) == 0 {
		return bizerr.BadRequest.ParameterError.New("Provide at least one image build ID.")
	}
	userID := util.GetToken(c).UserID
	builds := make([]*model.Kaniko, 0, len(req.IDList))
	seen := make(map[uint]struct{}, len(req.IDList))
	for _, id := range req.IDList {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		build, err := findKanikoForRemoval(c, isAdminMode, id, userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return bizerr.NotFound.DataBaseNotFound.New("Image build not found.")
		}
		if err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "Unable to read image build.")
		}
		if err := validateImageBuildRemoval(req.Action, build.Status); err != nil {
			return err
		}
		builds = append(builds, build)
	}
	for _, build := range builds {
		if req.Action == cancelImageBuild {
			if err := mgr.imagePacker.DeleteJob(c, build.ImagePackName, UserNameSpace); err != nil {
				return bizerr.Internal.K8sServiceError.Wrap(err, "Unable to cancel image build.")
			}
			continue
		}
		if ok, msg := mgr.deleteKaniko(c, build); !ok || msg != "" {
			return bizerr.Internal.ServiceError.New("Unable to complete image build deletion.")
		}
	}
	return nil
}

func validateImageBuildRemoval(action imageBuildRemovalAction, status model.BuildStatus) error {
	switch action {
	case cancelImageBuild:
		if isTerminalImageBuild(status) {
			return bizerr.Conflict.ResourceStatusError.New("The image build has already ended and cannot be canceled.")
		}
	case deleteImageBuild:
		if !isTerminalImageBuild(status) {
			return bizerr.Conflict.ResourceStatusError.New("Cancel the active image build before deleting it.")
		}
	default:
		return bizerr.BadRequest.InvalidRequest.New("Image build action must be cancel or delete.")
	}
	return nil
}

func isTerminalImageBuild(status model.BuildStatus) bool {
	return status == model.BuildJobFinished || status == model.BuildJobFailed || status == model.BuildJobCanceled
}

func findKanikoForRemoval(c *gin.Context, isAdminMode bool, kanikoID, userID uint) (*model.Kaniko, error) {
	k := query.Kaniko
	q := k.WithContext(c).Where(k.ID.Eq(kanikoID))
	if !isAdminMode {
		q = q.Where(k.UserID.Eq(userID))
	}
	return q.First()
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
	kaniko, err := findKanikoForRemoval(c, isAdminMode, kanikoID, userID)
	if err != nil || kaniko == nil {
		msg := fmt.Sprintf("kaniko not exist or have no permission %+v", err)
		klog.Error(msg)
		return false, msg
	}

	// 根据状态判断执行删除还是取消
	if isTerminalImageBuild(kaniko.Status) {
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
	kaniko, err := findKanikoForRemoval(c, isAdminMode, kanikoID, userID)
	if err != nil {
		errorMsg := fmt.Sprintf("kaniko not exist or have no permission %+v", err)
		klog.Error(errorMsg)
		return false, errorMsg
	}
	return mgr.deleteKaniko(c, kaniko)
}

func (mgr *ImagePackMgr) deleteKaniko(c *gin.Context, kaniko *model.Kaniko) (isSuccess bool, errMsg string) {
	var errorMsg string
	k := query.Kaniko
	// 2. if kaniko is finished, delete image entity
	if kaniko.Status == model.BuildJobFinished {
		if _, err := query.Image.WithContext(c).Where(query.Image.ImagePackName.Eq(kaniko.ImagePackName)).Delete(); err != nil {
			errorMsg = fmt.Sprintf("delete image entity failed! err:%v", err)
			klog.Error(errorMsg)
		}
	}
	// 3. delete kaniko entity from database
	if _, err := k.WithContext(c).Where(k.ID.Eq(kaniko.ID)).Delete(); err != nil {
		errorMsg = fmt.Sprintf("delete kaniko entity failed! err:%v", err)
		klog.Error(errorMsg)
		return false, errorMsg
	}
	// 4. if buildkit finished, then delete image from harbor
	if kaniko.Status == model.BuildJobFinished {
		if err := mgr.imageRegistry.DeleteImageFromProject(c, kaniko.ImageLink); err != nil {
			errorMsg = fmt.Sprintf("delete imagepack artifact failed! err:%+v", err)
			klog.Error(errorMsg)
		}
	}
	return true, errorMsg
}

// UserGetKanikoByImagePackName godoc
//
// @Summary Get a visible image build
// @Description Owners receive full build details. Public and shared image readers receive configuration without pod or node information.
// @Tags ImagePack
// @Produce json
// @Security Bearer
// @Param name query string true "Image build name"
// @Success 200 {object} resputil.Response[GetKanikoResponse]
// @Failure 400,403,404,500 {object} resputil.Response[any]
// @Router /v1/images/getbyname [get]
func (mgr *ImagePackMgr) UserGetKanikoByImagePackName(c *gin.Context) {
	mgr.getKanikoByImagePackName(c, false)
}

// AdminGetKanikoByImagePackName godoc
//
// @Summary Get any image build as an administrator
// @Description Returns full build configuration and pod information through the admin route.
// @Tags ImagePack
// @Produce json
// @Security Bearer
// @Param name query string true "Image build name"
// @Success 200 {object} resputil.Response[GetKanikoResponse]
// @Failure 400,403,404,500 {object} resputil.Response[any]
// @Router /v1/admin/images/getbyname [get]
func (mgr *ImagePackMgr) AdminGetKanikoByImagePackName(c *gin.Context) {
	mgr.getKanikoByImagePackName(c, true)
}

func (mgr *ImagePackMgr) getKanikoByImagePackName(c *gin.Context, admin bool) {
	var req GetKanikoRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "Image build name is required"))
		return
	}
	token := util.GetToken(c)
	kaniko, canManage, err := service.NewImageBuildDetailService(query.Q).Get(
		c, req.ImagePackName, token.UserID, token.AccountID, admin,
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	response := GetKanikoResponse{
		ID:               kaniko.ID,
		ImageLink:        kaniko.ImageLink,
		Status:           kaniko.Status,
		BuildSource:      kaniko.BuildSource,
		CreatedAt:        kaniko.CreatedAt,
		ImagePackName:    kaniko.ImagePackName,
		CanManage:        canManage,
		CanViewBuildLogs: canManage,
	}
	if kaniko.Description != nil {
		response.Description = *kaniko.Description
	}
	if kaniko.Dockerfile != nil {
		response.Dockerfile = *kaniko.Dockerfile
	}
	if canManage {
		response.PodName, response.PodNameSpace, response.NodeName = mgr.getBuildPodName(c, kaniko.ImagePackName)
	}
	resputil.Success(c, response)
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
	return mgr.getBuildPodName(c, kaniko.ImagePackName)
}

func (mgr *ImagePackMgr) getBuildPodName(c *gin.Context, imagePackName string) (name, ns, nodeName string) {
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
