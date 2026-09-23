package image

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

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

	db := baseDB.WithContext(c).
		Model(&model.Kaniko{}).
		Joins("LEFT JOIN users ON users.id = kanikos.user_id")
	if userID != nil {
		db = db.Where("kanikos.user_id = ?", *userID)
	}
	if len(statuses) > 0 {
		db = db.Where("kanikos.status IN ?", statuses)
	}
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where(
			"(LOWER(kanikos.image_link) LIKE ? ESCAPE '\\' OR "+
				"LOWER(kanikos.image_pack_name) LIKE ? ESCAPE '\\' OR "+
				"LOWER(COALESCE(kanikos.description, '')) LIKE ? ESCAPE '\\' OR "+
				"LOWER(users.name) LIKE ? ESCAPE '\\' OR LOWER(users.nickname) LIKE ? ESCAPE '\\')",
			pattern,
			pattern,
			pattern,
			pattern,
			pattern,
		)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		klog.Errorf("failed to count image build records: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count image build records failed"))
		return
	}
	for _, clause := range kanikoListSortClauses(request.Sort) {
		db = db.Order(clause)
	}
	kanikos := make([]*model.Kaniko, 0, request.PageSize)
	if err := db.Preload("User").
		Offset((request.Page - 1) * request.PageSize).
		Limit(request.PageSize).
		Find(&kanikos).Error; err != nil {
		klog.Errorf("failed to list image build records: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list image build records failed"))
		return
	}

	resputil.Success(c, resputil.NewPage(mgr.generateKanikoInfos(kanikos), total, request.Page, request.PageSize))
}

// UserListKanikoPage returns the current user's image build records using the shared page protocol.
//
//	@Summary	分页获取当前用户的镜像制作记录
//	@Tags		ImagePack
//	@Produce	json
//	@Security	Bearer
//	@Param		page		query	int		false	"Page number"
//	@Param		page_size	query	int		false	"Page size, 1-200"
//	@Param		search		query	string	false	"Search image, description, or creator"
//	@Param		status		query	string	false	"Filter by status; repeatable"
//	@Param		sort		query	string	false	"Sort fields"
//	@Success	200			{object}	resputil.Response[resputil.Page[KanikoInfo]]
//	@Router		/v1/images/kaniko/page [get]
func (mgr *ImagePackMgr) UserListKanikoPage(c *gin.Context) {
	userID := util.GetToken(c).UserID
	mgr.listKanikoPage(c, &userID)
}

// AdminListKanikoPage returns all image build records using the shared page protocol.
//
//	@Summary	分页获取全部镜像制作记录
//	@Tags		ImagePack
//	@Produce	json
//	@Security	Bearer
//	@Param		page		query	int		false	"Page number"
//	@Param		page_size	query	int		false	"Page size, 1-200"
//	@Param		search		query	string	false	"Search image, description, or creator"
//	@Param		status		query	string	false	"Filter by status; repeatable"
//	@Param		sort		query	string	false	"Sort fields"
//	@Success	200			{object}	resputil.Response[resputil.Page[KanikoInfo]]
//	@Router		/v1/admin/images/kaniko/page [get]
func (mgr *ImagePackMgr) AdminListKanikoPage(c *gin.Context) {
	mgr.listKanikoPage(c, nil)
}
