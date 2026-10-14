package handler

import (
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

const adminAccountDefaultSort = "name"

type adminAccountPageQuery struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=200"`
	Search   string `form:"search"`
	Sort     string `form:"sort"`
}

var adminAccountSortFields = map[string]string{
	"name":      "name",
	"nickname":  "nickname",
	"createdAt": "created_at",
	"expiredAt": "expired_at",
}

func validateAdminAccountSort(sortValue string) error {
	return pagination.ValidateSort(sortValue, adminAccountSortFields)
}

func adminAccountSortClauses(sortValue string) []string {
	return pagination.SortClauses(sortValue, adminAccountSortFields, "id DESC")
}

func bindAdminAccountPageQuery(c *gin.Context) (adminAccountPageQuery, error) {
	var request adminAccountPageQuery
	if err := pagination.PrepareBoundPageQuery(
		c.ShouldBindQuery(&request), "invalid account list query", &request.Search,
		request.Page, request.PageSize, &request.Sort, adminAccountDefaultSort, adminAccountSortFields,
	); err != nil {
		return adminAccountPageQuery{}, err
	}
	return request, nil
}

// GetAdminAccountPage returns administrator-visible accounts using the shared page protocol.
//
//	@Summary	管理员获取账户分页
//	@Description	按统一分页协议获取账户列表，搜索、排序和分页在数据库中执行
//	@Tags		Project
//	@Produce	json
//	@Security	Bearer
//	@Param		page	query	int	false	"Page number"
//	@Param		page_size	query	int	false	"Page size, 1-200"
//	@Param		search	query	string	false	"Search account name or nickname"
//	@Param		sort	query	string	false	"Sort fields"
//	@Success	200	{object}	resputil.Response[resputil.Page[ListAllResp]]
//	@Router		/v1/admin/accounts/page [get]
func (mgr *AccountMgr) GetAdminAccountPage(c *gin.Context) {
	request, err := bindAdminAccountPageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	db := query.Account.WithContext(c).UnderlyingDB().Model(&model.Account{})
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where("LOWER(name) LIKE ? ESCAPE '\\' OR LOWER(nickname) LIKE ? ESCAPE '\\'", pattern, pattern)
	}
	for _, clause := range adminAccountSortClauses(request.Sort) {
		db = db.Order(clause)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		klog.Errorf("failed to count accounts: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count accounts failed"))
		return
	}

	accounts := make([]*model.Account, 0, request.PageSize)
	if err := db.Offset((request.Page - 1) * request.PageSize).Limit(request.PageSize).Find(&accounts).Error; err != nil {
		klog.Errorf("failed to list accounts: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list accounts failed"))
		return
	}

	items := make([]ListAllResp, 0, len(accounts))
	for _, account := range accounts {
		items = append(items, buildListAllResp(account))
	}
	resputil.Success(c, resputil.NewPage(items, total, request.Page, request.PageSize))
}
