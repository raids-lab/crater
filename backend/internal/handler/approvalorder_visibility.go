package handler

import (
	"maps"
	"slices"

	"github.com/gin-gonic/gin"
	"gorm.io/gen/field"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/internal/util"
)

type ApprovalOrderNameReq struct {
	Name string `uri:"name" binding:"required"` // 工单名称
}

// swagger
//
//	@Summary		通过名称获取审批工单
//	@Description	按名称查询当前用户可访问的作业或数据集的审批工单
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name path string true "工单名称"
//	@Param			type query string false "关联资源类型" Enums(job, dataset)
//	@Success		200 {object} resputil.Response[[]ApprovalOrderResp] "成功返回工单列表"
//	@Failure		400 {object} resputil.Response[any] "请求参数错误"
//	@Failure		500 {object} resputil.Response[any] "服务器错误"
//	@Router			/v1/approvalorder/name/{name} [get]
func (mgr *ApprovalOrderMgr) GetApprovalOrderByName(c *gin.Context) {
	var nameReq ApprovalOrderNameReq
	if err := c.ShouldBindUri(&nameReq); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid approval order name"))
		return
	}
	var filter struct {
		Type string `form:"type" binding:"omitempty,oneof=job dataset"`
	}
	if err := c.ShouldBindQuery(&filter); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid approval order type"))
		return
	}
	token := util.GetToken(c)
	if token.UserID == 0 {
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("cannot get user id"))
		return
	}

	ao := query.ApprovalOrder
	ordersByName := ao.WithContext(c).Where(ao.Name.Eq(nameReq.Name))
	datasetOrder := ao.Type.Eq(string(model.ApprovalOrderTypeDataset))
	visibility := datasetOrder
	if filter.Type != string(model.ApprovalOrderTypeDataset) {
		// Older job orders stored a placeholder type ID. The job name is the unique key.
		visibleJobs := vcjobservice.VisibleJobByName(c, nameReq.Name, token).Select(query.Job.JobName)
		visibility = field.And(
			ao.Type.Eq(string(model.ApprovalOrderTypeJob)),
			ordersByName.Columns(ao.Name).In(visibleJobs),
		)
		if filter.Type == "" {
			visibility = field.Or(visibility, datasetOrder)
		}
	}
	orders, err := ordersByName.Where(visibility).
		Preload(ao.Creator).
		Preload(ao.Reviewer).
		Order(ao.CreatedAt.Desc(), ao.ID.Desc()).
		Find()
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to get approval orders"))
		return
	}

	datasetIDs, err := approvalOrderVisibleDatasetIDs(c, token, orders)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to check dataset access"))
		return
	}
	result := make([]ApprovalOrderResp, 0, len(orders))
	for _, order := range orders {
		if order.Type == model.ApprovalOrderTypeDataset {
			if _, visible := datasetIDs[order.Content.Data().ApprovalOrderTypeID]; !visible {
				continue
			}
		}
		result = append(result, convertToApprovalOrderResp(order))
	}
	resputil.Success(c, result)
}

func approvalOrderVisibleDatasetIDs(c *gin.Context, token util.JWTMessage, orders []*model.ApprovalOrder) (map[uint]struct{}, error) {
	candidates := make(map[uint]struct{})
	for _, order := range orders {
		if order.Type == model.ApprovalOrderTypeDataset && order.Content.Data().ApprovalOrderTypeID != 0 {
			candidates[order.Content.Data().ApprovalOrderTypeID] = struct{}{}
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	d := query.Dataset
	datasets := d.WithContext(c)
	if token.RolePlatform != model.RoleAdmin {
		datasets = sharedDatasets(c, token)
	}
	var ids []uint
	if err := datasets.Where(d.ID.In(slices.Collect(maps.Keys(candidates))...)).Pluck(d.ID, &ids); err != nil {
		return nil, err
	}
	visible := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		visible[id] = struct{}{}
	}
	return visible, nil
}
