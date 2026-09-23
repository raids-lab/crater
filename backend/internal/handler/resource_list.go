package handler

import (
	"fmt"

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

type resourcePageQuery struct {
	Page             int     `form:"page,default=1" binding:"min=1"`
	PageSize         int     `form:"page_size,default=10" binding:"min=1,max=200"`
	Search           string  `form:"search"`
	Sort             string  `form:"sort"`
	WithVendorDomain bool    `form:"withVendorDomain"`
	DomainPrefix     *string `form:"domainPrefix" binding:"omitempty,hostname_rfc1123"`
}

var resourcePageSortFields = map[string]string{
	"name":            "resource_name",
	"type":            "type",
	"amount":          "amount",
	"amountSingleMax": "amount_single_max",
	"format":          "format",
	"priority":        "priority",
	"label":           "label",
}

var resourcePageTypes = map[string]struct{}{
	string(model.ResourceTypeGPU):  {},
	string(model.ResourceTypeRDMA): {},
	string(model.ResourceTypeVGPU): {},
}

func resourcePageSortClauses(sortValue string) []string {
	return pagination.SortClauses(sortValue, resourcePageSortFields, "id ASC")
}

func bindResourcePageQuery(c *gin.Context) (resourcePageQuery, []string, error) {
	var request resourcePageQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		return resourcePageQuery{}, nil, bizerr.BadRequest.ParameterError.Wrap(
			err,
			"invalid resource list query",
		)
	}
	var err error
	request.Search, err = pagination.NormalizePageQuery(request.Search, request.Page, request.PageSize)
	if err != nil {
		return resourcePageQuery{}, nil, err
	}
	request.Sort, err = pagination.DefaultAndValidateSort(request.Sort, "name", resourcePageSortFields)
	if err != nil {
		return resourcePageQuery{}, nil, err
	}

	resourceTypes := c.QueryArray("type")
	for _, resourceType := range resourceTypes {
		if _, ok := resourcePageTypes[resourceType]; !ok {
			return resourcePageQuery{}, nil, bizerr.BadRequest.ParameterError.New(
				fmt.Sprintf("unsupported resource type %q", resourceType),
			)
		}
	}
	return request, resourceTypes, nil
}

func bindResourcePriceIDs(c *gin.Context) ([]uint, error) {
	return pagination.ParsePositiveUintValues(c.QueryArray("resource_id"), "resource_id")
}

// ListResourcePage returns resources using the shared page protocol.
//
//	@Summary	分页获取集群资源
//	@Tags		Resource
//	@Produce	json
//	@Security	Bearer
//	@Param		page				query	int		false	"Page number"
//	@Param		page_size			query	int		false	"Page size, 1-200"
//	@Param		search				query	string	false	"Search resource name, label, or vendor"
//	@Param		type				query	string	false	"Resource type filter; repeatable"
//	@Param		sort				query	string	false	"Sort fields"
//	@Param		withVendorDomain	query	bool	false	"Only resources with GPU type"
//	@Param		domainPrefix		query	string	false	"Vendor domain prefix"
//	@Success	200					{object}	resputil.Response[resputil.Page[ResourceResp]]
//	@Router		/v1/resources/page [get]
func (mgr *ResourceMgr) ListResourcePage(c *gin.Context) {
	mgr.listResourcePageWithDB(c, query.GetDB())
}

func (mgr *ResourceMgr) listResourcePageWithDB(c *gin.Context, baseDB *gorm.DB) {
	request, resourceTypes, err := bindResourcePageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	db := baseDB.WithContext(c).Model(&model.Resource{})
	if request.WithVendorDomain {
		db = db.Where("type = ?", model.ResourceTypeGPU)
	}
	if request.DomainPrefix != nil {
		db = db.Where("vendor_domain = ?", *request.DomainPrefix)
	}
	if len(resourceTypes) > 0 {
		db = db.Where("type IN ?", resourceTypes)
	}
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where(
			"(LOWER(resource_name) LIKE ? ESCAPE '\\' OR LOWER(label) LIKE ? ESCAPE '\\' OR "+
				"LOWER(COALESCE(vendor_domain, '')) LIKE ? ESCAPE '\\' OR "+
				"LOWER(resource_type) LIKE ? ESCAPE '\\')",
			pattern,
			pattern,
			pattern,
			pattern,
		)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		klog.Errorf("failed to count resources: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count resources failed"))
		return
	}
	for _, clause := range resourcePageSortClauses(request.Sort) {
		db = db.Order(clause)
	}

	resources := make([]*model.Resource, 0, request.PageSize)
	if err := db.Offset((request.Page - 1) * request.PageSize).
		Limit(request.PageSize).
		Find(&resources).Error; err != nil {
		klog.Errorf("failed to list resources: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list resources failed"))
		return
	}
	items := make([]ResourceResp, 0, len(resources))
	for _, resource := range resources {
		items = append(items, buildResourceResp(resource))
	}
	resputil.Success(c, resputil.NewPage(items, total, request.Page, request.PageSize))
}
