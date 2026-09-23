package handler

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

type datasetSharePageQuery struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=200"`
	Search   string `form:"search"`
	Sort     string `form:"sort"`
}

func bindDatasetSharePageQuery(c *gin.Context) (datasetSharePageQuery, error) {
	var request datasetSharePageQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		return datasetSharePageQuery{}, bizerr.BadRequest.ParameterError.Wrap(
			err,
			"invalid dataset share list query",
		)
	}
	request.Search = strings.TrimSpace(request.Search)
	if utf8.RuneCountInString(request.Search) > datasetMaxSearchRunes {
		return datasetSharePageQuery{}, bizerr.BadRequest.ParameterError.New(
			"search accepts at most 128 characters",
		)
	}
	if request.Page > int(^uint(0)>>1)/request.PageSize {
		return datasetSharePageQuery{}, bizerr.BadRequest.ParameterError.New(
			"page is too large for page_size",
		)
	}
	if request.Sort == "" {
		request.Sort = "name"
	}
	if request.Sort != "name" && request.Sort != "-name" {
		return datasetSharePageQuery{}, bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("unsupported sort field %q", strings.TrimPrefix(request.Sort, "-")),
		)
	}
	return request, nil
}

func datasetShareSortClauses(sortValue, nameColumn, idColumn string) []string {
	direction := "ASC"
	if strings.HasPrefix(sortValue, "-") {
		direction = "DESC"
	}
	return []string{nameColumn + " " + direction, idColumn + " ASC"}
}

func getDatasetForSharePage(c *gin.Context, datasetID uint) (*model.Dataset, error) {
	dataset, err := query.Dataset.WithContext(c).Where(query.Dataset.ID.Eq(datasetID)).First()
	if err != nil {
		return nil, bizerr.NotFound.DataBaseNotFound.Wrap(err, "dataset does not exist")
	}
	return dataset, nil
}

// ListUsersInDatasetPage returns users sharing a dataset using the shared page protocol.
//
//	@Summary	分页获取数据共享用户
//	@Tags		Dataset
//	@Produce	json
//	@Security	Bearer
//	@Param		datasetId	path	uint	true	"Dataset ID"
//	@Param		page		query	int		false	"Page number"
//	@Param		page_size	query	int		false	"Page size, 1-200"
//	@Param		search		query	string	false	"Search username or nickname"
//	@Param		sort		query	string	false	"Sort by name or -name"
//	@Success	200			{object}	resputil.Response[resputil.Page[UserOfDatasetResp]]
//	@Router		/v1/dataset/{datasetId}/usersIn/page [get]
func (mgr *DatasetMgr) ListUsersInDatasetPage(c *gin.Context) {
	var uri DatasetGetReq
	if err := c.ShouldBindUri(&uri); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid dataset ID"))
		return
	}
	request, err := bindDatasetSharePageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	dataset, err := getDatasetForSharePage(c, uri.DatasetID)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	db := query.GetDB().WithContext(c).
		Table("users").
		Joins("JOIN user_datasets ON user_datasets.user_id = users.id").
		Where("users.deleted_at IS NULL").
		Where("user_datasets.deleted_at IS NULL").
		Where("user_datasets.dataset_id = ?", dataset.ID)
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where(
			"(LOWER(users.name) LIKE ? ESCAPE '\\' OR LOWER(users.nickname) LIKE ? ESCAPE '\\')",
			pattern,
			pattern,
		)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		klog.Errorf("failed to count dataset user shares: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count dataset user shares failed"))
		return
	}
	for _, clause := range datasetShareSortClauses(request.Sort, "users.name", "users.id") {
		db = db.Order(clause)
	}

	items := make([]UserOfDatasetResp, 0, request.PageSize)
	if err := db.Select(
		"users.id, users.name, users.attributes AS user_info, (users.id = ?) AS is_owner",
		dataset.UserID,
	).
		Offset((request.Page - 1) * request.PageSize).
		Limit(request.PageSize).
		Scan(&items).Error; err != nil {
		klog.Errorf("failed to list dataset user shares: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list dataset user shares failed"))
		return
	}
	resputil.Success(c, resputil.NewPage(items, total, request.Page, request.PageSize))
}

// ListQueuesInDatasetPage returns accounts sharing a dataset using the shared page protocol.
//
//	@Summary	分页获取数据共享账户
//	@Tags		Dataset
//	@Produce	json
//	@Security	Bearer
//	@Param		datasetId	path	uint	true	"Dataset ID"
//	@Param		page		query	int		false	"Page number"
//	@Param		page_size	query	int		false	"Page size, 1-200"
//	@Param		search		query	string	false	"Search account name or nickname"
//	@Param		sort		query	string	false	"Sort by name or -name"
//	@Success	200			{object}	resputil.Response[resputil.Page[QueueDatasetGetResp]]
//	@Router		/v1/dataset/{datasetId}/queuesIn/page [get]
func (mgr *DatasetMgr) ListQueuesInDatasetPage(c *gin.Context) {
	var uri DatasetGetReq
	if err := c.ShouldBindUri(&uri); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid dataset ID"))
		return
	}
	request, err := bindDatasetSharePageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	if _, err := getDatasetForSharePage(c, uri.DatasetID); err != nil {
		resputil.HandleError(c, err)
		return
	}

	db := query.GetDB().WithContext(c).
		Table("accounts").
		Joins("JOIN account_datasets ON account_datasets.account_id = accounts.id").
		Where("accounts.deleted_at IS NULL").
		Where("account_datasets.deleted_at IS NULL").
		Where("account_datasets.dataset_id = ?", uri.DatasetID)
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where(
			"(LOWER(accounts.name) LIKE ? ESCAPE '\\' OR LOWER(accounts.nickname) LIKE ? ESCAPE '\\')",
			pattern,
			pattern,
		)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		klog.Errorf("failed to count dataset account shares: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count dataset account shares failed"))
		return
	}
	for _, clause := range datasetShareSortClauses(request.Sort, "accounts.nickname", "accounts.id") {
		db = db.Order(clause)
	}

	items := make([]QueueDatasetGetResp, 0, request.PageSize)
	if err := db.Select("accounts.id, accounts.nickname AS name").
		Offset((request.Page - 1) * request.PageSize).
		Limit(request.PageSize).
		Scan(&items).Error; err != nil {
		klog.Errorf("failed to list dataset account shares: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list dataset account shares failed"))
		return
	}
	resputil.Success(c, resputil.NewPage(items, total, request.Page, request.PageSize))
}
