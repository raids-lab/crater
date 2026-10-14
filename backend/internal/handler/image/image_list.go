package image

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

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
	"imageShareStatus": "",
}

var imageListShareStatuses = map[string]struct{}{
	string(model.Public):       {},
	string(model.Private):      {},
	string(model.UserShare):    {},
	string(model.AccountShare): {},
}

func imageListSortClauses(sortValue, visibilityExpression string) []string {
	fields := make(map[string]string, len(imageListSortFields))
	for name, field := range imageListSortFields {
		fields[name] = field
	}
	fields["imageShareStatus"] = visibilityExpression
	return pagination.SortClauses(sortValue, fields, "images.id DESC")
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

func imageVisibilityExpression(userID, accountID uint, isAdmin bool) string {
	publicCondition := fmt.Sprintf(
		"(images.is_public = TRUE OR EXISTS (SELECT 1 FROM image_accounts ia_public "+
			"WHERE ia_public.image_id = images.id AND ia_public.account_id = %d "+
			"AND ia_public.deleted_at IS NULL))",
		model.DefaultAccountID,
	)
	if isAdmin {
		return fmt.Sprintf(
			"CASE WHEN %s THEN '%s' ELSE '%s' END",
			publicCondition,
			model.Public,
			model.Private,
		)
	}
	return fmt.Sprintf(
		"CASE "+
			"WHEN %s THEN '%s' "+
			"WHEN EXISTS (SELECT 1 FROM image_accounts ia_account WHERE ia_account.image_id = images.id "+
			"AND ia_account.account_id = %d AND ia_account.deleted_at IS NULL) THEN '%s' "+
			"WHEN images.user_id = %d THEN '%s' "+
			"WHEN EXISTS (SELECT 1 FROM image_users iu WHERE iu.image_id = images.id "+
			"AND iu.user_id = %d AND iu.deleted_at IS NULL) THEN '%s' "+
			"ELSE NULL END",
		publicCondition,
		model.Public,
		accountID,
		model.AccountShare,
		userID,
		model.Private,
		userID,
		model.UserShare,
	)
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
	token := util.GetToken(c)
	visibilityExpression := imageVisibilityExpression(token.UserID, token.AccountID, isAdmin)

	db := query.GetDB().WithContext(c).
		Model(&model.Image{}).
		Joins("LEFT JOIN users ON users.id = images.user_id").
		Where("images.image_link <> ''")
	if !isAdmin {
		db = db.Where("(" + visibilityExpression + ") IS NOT NULL")
	}
	if len(shareStatuses) > 0 {
		db = db.Where("("+visibilityExpression+") IN ?", shareStatuses)
	}
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where(
			"(LOWER(images.image_link) LIKE ? ESCAPE '\\' OR "+
				"LOWER(COALESCE(images.description, '')) LIKE ? ESCAPE '\\' OR "+
				"LOWER(COALESCE(images.image_pack_name, '')) LIKE ? ESCAPE '\\' OR "+
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
		klog.Errorf("failed to count visible images: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count visible images failed"))
		return
	}
	for _, clause := range imageListSortClauses(request.Sort, visibilityExpression) {
		db = db.Order(clause)
	}
	rows := make([]visibleImageRow, 0, request.PageSize)
	if err := db.Select("images.id, " + visibilityExpression + " AS image_share_status").
		Offset((request.Page - 1) * request.PageSize).
		Limit(request.PageSize).
		Scan(&rows).Error; err != nil {
		klog.Errorf("failed to list visible image ids: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list visible images failed"))
		return
	}

	items, err := mgr.imageItemsForRows(c, rows)
	if err != nil {
		klog.Errorf("failed to load visible image details: %v", err)
		resputil.HandleError(c, err)
		return
	}

	resputil.Success(c, resputil.NewPage(items, total, request.Page, request.PageSize))
}

// UserListImagePage returns images visible to the current user using the shared page protocol.
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
//	@Router		/v1/images/image/page [get]
func (mgr *ImagePackMgr) UserListImagePage(c *gin.Context) {
	mgr.listImagePage(c, false)
}

// AdminListImagePage returns all images using the shared page protocol.
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
//	@Router		/v1/admin/images/image/page [get]
func (mgr *ImagePackMgr) AdminListImagePage(c *gin.Context) {
	mgr.listImagePage(c, true)
}
