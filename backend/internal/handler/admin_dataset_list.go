package handler

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

type adminDatasetListPageQuery struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=200"`
	Search   string `form:"search"`
	Sort     string `form:"sort"`
}

var adminDatasetTypes = map[string]struct{}{
	string(model.DataTypeDataset):   {},
	string(model.DataTypeModel):     {},
	string(model.DataTypeShareFile): {},
}

var adminDatasetSortFields = map[string]string{
	"name":       "name",
	"type":       "type",
	"createdAt":  "created_at",
	"updatedAt":  "updated_at",
	"mountCount": "mount_count",
}

func bindAdminDatasetListPageQuery(c *gin.Context) (adminDatasetListPageQuery, []string, error) {
	var request adminDatasetListPageQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		return adminDatasetListPageQuery{}, nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid admin dataset list query")
	}
	var err error
	request.Search, err = pagination.NormalizePageQuery(request.Search, request.Page, request.PageSize)
	if err != nil {
		return adminDatasetListPageQuery{}, nil, err
	}

	types := c.QueryArray("type")
	seenTypes := make(map[string]struct{}, len(types))
	validTypes := make([]string, 0, len(types))
	for _, value := range types {
		if _, ok := adminDatasetTypes[value]; !ok {
			return adminDatasetListPageQuery{}, nil, bizerr.BadRequest.ParameterError.New(fmt.Sprintf("unsupported dataset type %q", value))
		}
		if _, seen := seenTypes[value]; seen {
			continue
		}
		seenTypes[value] = struct{}{}
		validTypes = append(validTypes, value)
	}

	request.Sort, err = pagination.DefaultAndValidateSort(request.Sort, "-updatedAt", adminDatasetSortFields)
	if err != nil {
		return adminDatasetListPageQuery{}, nil, err
	}
	return request, validTypes, nil
}

func adminDatasetSortClauses(sortValue string) []string {
	return pagination.SortClauses(sortValue, adminDatasetSortFields, "id DESC")
}

// GetAllDatasetPage returns all datasets visible to administrators in the shared page shape.
// The legacy /alldataset endpoint remains available for existing clients.
//
//	@Summary	管理员分页获取所有数据资源
//	@Description	按类型、搜索和排序条件过滤后分页返回数据资源
//	@Tags		Dataset
//	@Produce	json
//	@Security	Bearer
//	@Param		page		query	int		false	"Page number"
//	@Param		page_size	query	int		false	"Page size, 1-200"
//	@Param		search		query	string	false	"Search name or description"
//	@Param		type		query	string	false	"Filter by type; repeatable"
//	@Param		sort		query	string	false	"Sort fields"
//	@Success	200	{object}	resputil.Response[resputil.Page[DatasetResp]]
//	@Failure	400	{object}	resputil.Response[any]
//	@Failure	500	{object}	resputil.Response[any]
//	@Router		/v1/admin/dataset/alldataset/page [get]
func (mgr *DatasetMgr) GetAllDatasetPage(c *gin.Context) {
	request, types, err := bindAdminDatasetListPageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	db := query.Dataset.WithContext(c).UnderlyingDB().Model(&model.Dataset{})
	if len(types) > 0 {
		db = db.Where("type IN ?", types)
	}
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where("(LOWER(name) LIKE ? ESCAPE '\\' OR LOWER(describe) LIKE ? ESCAPE '\\')", pattern, pattern)
	}
	for _, clause := range adminDatasetSortClauses(request.Sort) {
		db = db.Order(clause)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		klog.Errorf("failed to count admin datasets: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count admin datasets failed"))
		return
	}

	datasets := make([]*model.Dataset, 0)
	if err := db.Preload("User").Offset((request.Page - 1) * request.PageSize).Limit(request.PageSize).Find(&datasets).Error; err != nil {
		klog.Errorf("failed to list admin datasets: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list admin datasets failed"))
		return
	}

	result, err := convertDatasetBatch(c, datasets)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "enrich admin datasets failed"))
		return
	}
	resputil.Success(c, resputil.NewPage(result, total, request.Page, request.PageSize))
}
