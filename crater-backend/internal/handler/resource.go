package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewResourceMgr)
}

type ResourceMgr struct {
	name       string
	kubeClient kubernetes.Interface
}

func NewResourceMgr(conf *RegisterConfig) Manager {
	return &ResourceMgr{
		name:       "resources",
		kubeClient: conf.KubeClient,
	}
}

func (mgr *ResourceMgr) GetName() string { return mgr.name }

func (mgr *ResourceMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ResourceMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.ListResource)
	g.GET("/:id/networks", mgr.GetGPUNetworks)
	g.GET(":id/vgpu", mgr.GetGPUVGPUResources)
}

func (mgr *ResourceMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.POST("/sync", mgr.SyncResource)
	g.PUT("/:id", mgr.UpdateResource) // 注意这里改为新的方法名
	g.DELETE("/:id", mgr.DeleteResource)
	g.POST("/:id/networks", mgr.LinkGPUToRDMA)
	g.GET("/:id/networks", mgr.GetGPUNetworks)
	g.DELETE("/:id/networks/:networkId", mgr.DeleteResourceLink)
	// VGPU management endpoints
	g.POST("/:id/vgpu", mgr.LinkGPUToVGPU)
	g.GET("/:id/vgpu", mgr.GetGPUVGPUResources)
	g.PUT("/:id/vgpu/:vgpuId", mgr.UpdateGPUVGPULink)
	g.DELETE("/:id/vgpu/:vgpuId", mgr.DeleteGPUVGPULink)
}

type (
	ListResourceReq struct {
		// VendorDomain of the resource in parameter (optional)
		WithVendorDomain bool    `form:"withVendorDomain"`
		DomainPrefix     *string `form:"domainPrefix" binding:"omitempty,hostname_rfc1123"`
	}
)

// ListResource godoc
//
//	@Summary		Get a list of resources based on the specified parameters
//	@Description	If the vendorDomain parameter is provided, the API will return a list of resources that match the specified vendor domain.
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			vendorDomain	query		string					false	"Vendor domain of the resource (For example: 'nvidia.com'	)"
//	@Success		200				{object}	resputil.Response[any]	"Success"
//	@Failure		400				{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500				{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/resources [get]
func (mgr *ResourceMgr) ListResource(c *gin.Context) {
	var req ListResourceReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	r := query.Resource
	q := r.WithContext(c).Order(r.Priority.Desc())
	if req.WithVendorDomain {
		// default use nvidia.com
		q = q.Where(r.Type.Eq(string(model.ResourceTypeGPU)))
	}
	resources, err := q.Find()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list resources: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, resources)
}

type (
	quantity struct {
		Total resource.Quantity
		Max   resource.Quantity
	}
)

// SyncResource godoc
//
//	@Summary		Get allocatable resources from the Kubernetes cluster and update the database
//	@Description	This API will get the allocatable resources from the Kubernetes cluster
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"Success"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/resources/sync [post]
func (mgr *ResourceMgr) SyncResource(c *gin.Context) {
	nodes, err := mgr.kubeClient.CoreV1().Nodes().List(c, metav1.ListOptions{})
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list nodes: %v", err), resputil.NotSpecified)
		return
	}

	// Create a map to store the resource quantities
	reourceQuantities := make(map[string]quantity)

	// Iterate over each node to get capacities: .status.allocatable
	for i := range nodes.Items {
		node := &nodes.Items[i]
		// Get the allocatable resources of the node
		resources := node.Status.Allocatable
		for key, value := range resources {
			// Get the label value
			resourceName := key.String()
			// Get the number of quantities
			// Add the quantity to the map
			if q, ok := reourceQuantities[resourceName]; ok {
				q.Total.Add(value)
				if value.Cmp(q.Max) > 0 {
					q.Max = value
				}
				reourceQuantities[resourceName] = q
			} else {
				reourceQuantities[resourceName] = quantity{
					Total: value,
					Max:   value,
				}
			}
		}
	}

	// Update the database with the latest information
	r := query.Resource
	for resourceName, quantity := range reourceQuantities {
		info, ierr := r.WithContext(c).Where(r.ResourceName.Eq(resourceName)).
			Updates(map[string]any{
				"amount":            quantity.Total.Value(),
				"amount_single_max": quantity.Max.Value(),
			})
		if ierr != nil {
			resputil.Error(c, fmt.Sprintf("failed to update resource: %v", ierr), resputil.NotSpecified)
			return
		}
		if info.RowsAffected == 0 {
			// if resourceName like "nvidia.com/gpu",
			// then vendorDomain = "nvidia.com", resourceType = "gpu", label = "gpu"

			// 1. try to split resourceName by "/"
			// 2. if split result length is 2, then vendorDomain = split[0], resourceType = split[1]
			// 3. if split result length is 1, then vendorDomain = "", resourceType = split[0]
			// 4. label = resourceType
			split := strings.Split(resourceName, "/")
			var resourceType, label string
			vendorDomain := new(string)
			if len(split) == 2 {
				*vendorDomain = split[0]
				resourceType = split[1]
				label = resourceType
			} else {
				vendorDomain = nil
				resourceType = split[0]
				label = resourceType
			}

			newResource := model.Resource{
				ResourceName:    resourceName,
				VendorDomain:    vendorDomain,
				ResourceType:    resourceType,
				Amount:          quantity.Total.Value(),
				AmountSingleMax: quantity.Max.Value(),
				Format:          string(quantity.Max.Format),
				Label:           label,
			}

			if createErr := r.WithContext(c).Create(&newResource); createErr != nil {
				resputil.Error(c, fmt.Sprintf("failed to create resource: %v", createErr), resputil.NotSpecified)
				return
			}
		}
	}

	// Delete resources that no longer exist in the cluster
	allResources, err := r.WithContext(c).Find()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list all resources: %v", err), resputil.NotSpecified)
		return
	}

	for _, resource := range allResources {
		if _, exists := reourceQuantities[resource.ResourceName]; !exists {
			// This resource doesn't exist in the cluster anymore, permanently delete it from database
			_, err := r.WithContext(c).Where(r.ID.Eq(resource.ID)).Unscoped().Delete()
			if err != nil {
				resputil.Error(c, fmt.Sprintf("failed to delete resource: %v", err), resputil.NotSpecified)
				return
			}
		}
	}

	resputil.Success(c, nil)
}

type (
	UpdateResourceReq struct {
		Label *string                   `json:"label" binding:"omitempty"`
		Type  *model.CraterResourceType `json:"type" binding:"omitempty"`
	}
	ResourcePathReq struct {
		ID uint `uri:"id" binding:"required"`
	}
)

// UpdateResource godoc
//
//	@Summary		Update a resource's attributes
//	@Description	This API will update the label or type of a resource based on the specified ID.
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id			path		uint					true	"Resource ID"
//	@Param			resource	body		UpdateResourceReq		true	"Resource attributes to update"
//	@Success		200			{object}	resputil.Response[any]	"Success"
//	@Failure		400			{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500			{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/resources/{id} [put]
func (mgr *ResourceMgr) UpdateResource(c *gin.Context) {
	var req UpdateResourceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	var param ResourcePathReq
	if err := c.ShouldBindUri(&param); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	r := query.Resource
	updates := make(map[string]any)

	if req.Label != nil {
		updates["label"] = *req.Label
	}

	if req.Type != nil {
		updates["type"] = *req.Type
	}

	if len(updates) == 0 {
		resputil.BadRequestError(c, "no fields to update")
		return
	}

	_, err := r.WithContext(c).Where(r.ID.Eq(param.ID)).Updates(updates)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to update resource: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nil)
}

// DeleteResource godoc
//
//	@Summary		Delete a resource
//	@Description	This API will delete a resource based on the specified ID.
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		uint					true	"Resource ID"
//	@Success		200	{object}	resputil.Response[any]	"Success"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/resources/{id} [delete]
func (mgr *ResourceMgr) DeleteResource(c *gin.Context) {
	var param ResourcePathReq
	if err := c.ShouldBindUri(&param); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	r := query.Resource
	_, err := r.WithContext(c).Where(r.ID.Eq(param.ID)).Delete()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to delete resource: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nil)
}

type GetGPUNetworksReq struct {
	GPUID uint `uri:"gpuId" binding:"required"`
}

// GetGPUNetworks godoc
//
//	@Summary		Get all RDMA resources linked to a GPU resource
//	@Description	This API will return all RDMA resources linked to the specified GPU resource.
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			gpuId	path		uint					true	"GPU Resource ID"
//	@Success		200		{object}	resputil.Response[any]	"Success"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/resources/gpu/{gpuId}/networks [get]
func (mgr *ResourceMgr) GetGPUNetworks(c *gin.Context) {
	var req ResourcePathReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Success(c, []model.Resource{})
		return
	}

	// 验证 GPU 资源存在并且类型是 GPU
	r := query.Resource
	gpuResource, err := r.WithContext(c).Where(r.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.Success(c, []model.Resource{})
		return
	}

	if gpuResource.Type == nil || *gpuResource.Type != model.ResourceTypeGPU {
		resputil.Success(c, []model.Resource{})
		return
	}

	// 获取与该 GPU 关联的所有 RDMA 资源
	rn := query.ResourceNetwork
	networkLinks, err := rn.WithContext(c).Where(rn.ResourceID.Eq(req.ID)).Find()
	if err != nil {
		resputil.Success(c, []model.Resource{})
		return
	}

	if len(networkLinks) == 0 {
		resputil.Success(c, []model.Resource{})
		return
	}

	// 提取 RDMA IDs
	var rdmaIDs []uint
	for _, link := range networkLinks {
		rdmaIDs = append(rdmaIDs, link.NetworkID)
	}

	// 通过 IDs 获取 RDMA 资源详情
	rdmaResources, err := r.WithContext(c).Where(r.ID.In(rdmaIDs...)).Find()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to retrieve RDMA resources: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, rdmaResources)
}

type LinkResourceReq struct {
	RDMAID uint `json:"rdmaId" binding:"required"`
}

// LinkGPUToRDMA godoc
//
//	@Summary		Link a GPU resource to an RDMA resource
//	@Description	This API will create a relationship between a GPU resource and an RDMA resource.
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			linkRequest	body		LinkResourceReq			true	"GPU and RDMA IDs to link"
//	@Success		200			{object}	resputil.Response[any]	"Success"
//	@Failure		400			{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500			{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/resources/link [post]
func (mgr *ResourceMgr) LinkGPUToRDMA(c *gin.Context) {
	var pathReq ResourcePathReq
	if err := c.ShouldBindUri(&pathReq); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}
	var req LinkResourceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	// 验证 GPU 资源存在并且类型是 GPU
	r := query.Resource
	gpuResource, err := r.WithContext(c).Where(r.ID.Eq(pathReq.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to find GPU resource: %v", err), resputil.NotSpecified)
		return
	}

	if gpuResource.Type == nil || *gpuResource.Type != model.ResourceTypeGPU {
		resputil.BadRequestError(c, "specified resource is not a GPU")
		return
	}

	// 验证 RDMA 资源存在并且类型是 RDMA
	rdmaResource, err := r.WithContext(c).Where(r.ID.Eq(req.RDMAID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to find RDMA resource: %v", err), resputil.NotSpecified)
		return
	}

	if rdmaResource.Type == nil || *rdmaResource.Type != model.ResourceTypeRDMA {
		resputil.BadRequestError(c, "specified resource is not an RDMA")
		return
	}

	// 创建关联关系
	rn := query.ResourceNetwork
	network := &model.ResourceNetwork{
		ResourceID: gpuResource.ID,
		NetworkID:  rdmaResource.ID,
	}

	err = rn.WithContext(c).Create(network)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create resource network relationship: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, nil)
}

type DeleteResourceLinkReq struct {
	ID        uint `uri:"id" binding:"required"`
	NetworkID uint `uri:"networkId" binding:"required"`
}

// DeleteResourceLink godoc
//
//	@Summary		Delete the link between a GPU resource and an RDMA resource
//	@Description	This API will delete the link between a GPU resource and an RDMA resource based on the specified IDs.
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id			path		uint					true	"GPU Resource ID"
//	@Param			networkId	path		uint					true	"RDMA Resource ID"
//	@Success		200			{object}	resputil.Response[any]	"Success"
//	@Failure		400			{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500			{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/resources/{id}/networks/{networkId} [delete]
func (mgr *ResourceMgr) DeleteResourceLink(c *gin.Context) {
	var req DeleteResourceLinkReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	// 验证 GPU 资源存在并且类型是 GPU
	r := query.Resource
	gpuResource, err := r.WithContext(c).Where(r.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to find GPU resource: %v", err), resputil.NotSpecified)
		return
	}

	if gpuResource.Type == nil || *gpuResource.Type != model.ResourceTypeGPU {
		resputil.BadRequestError(c, "specified resource is not a GPU")
		return
	}

	rn := query.ResourceNetwork
	_, err = rn.WithContext(c).Where(rn.ResourceID.Eq(req.ID), rn.NetworkID.Eq(req.NetworkID)).Delete()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to delete resource network relationship: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, nil)
}

// VGPU management APIs

type LinkGPUToVGPUReq struct {
	VGPUResourceID uint    `json:"vgpuResourceId" binding:"required"`
	Min            *int    `json:"min,omitempty" binding:"omitempty"`
	Max            *int    `json:"max,omitempty" binding:"omitempty"`
	Description    *string `json:"description,omitempty"`
}

// LinkGPUToVGPU godoc
//
//	@Summary		Link a GPU resource to a VGPU resource
//	@Description	This API will create a one-to-one relationship between a GPU resource and a VGPU resource.
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id			path		uint					true	"GPU Resource ID"
//	@Param			vgpuRequest	body		LinkGPUToVGPUReq		true	"VGPU resource configuration"
//	@Success		200			{object}	resputil.Response[any]	"Success"
//	@Failure		400			{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500			{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/resources/{id}/vgpu [post]
func (mgr *ResourceMgr) LinkGPUToVGPU(c *gin.Context) {
	var pathReq ResourcePathReq
	if err := c.ShouldBindUri(&pathReq); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	var req LinkGPUToVGPUReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	// Validate range
	if req.Min != nil && req.Max != nil {
		if *req.Min > *req.Max {
			resputil.BadRequestError(c, "min cannot be greater than max")
			return
		}
	}

	// Verify GPU resource exists and is of type GPU
	r := query.Resource
	gpuResource, err := r.WithContext(c).Where(r.ID.Eq(pathReq.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to find GPU resource: %v", err), resputil.NotSpecified)
		return
	}

	if gpuResource.Type == nil || *gpuResource.Type != model.ResourceTypeGPU {
		resputil.BadRequestError(c, "specified resource is not a GPU")
		return
	}

	// Verify VGPU resource exists
	vgpuResource, err := r.WithContext(c).Where(r.ID.Eq(req.VGPUResourceID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to find VGPU resource: %v", err), resputil.NotSpecified)
		return
	}
	if vgpuResource.Type == nil || *vgpuResource.Type != model.ResourceTypeVGPU {
		resputil.BadRequestError(c, "specified resource is not a VGPU resource")
		return
	}

	// Check if relationship already exists
	rv := query.ResourceVGPU
	existing, _ := rv.WithContext(c).Where(rv.GPUResourceID.Eq(pathReq.ID), rv.VGPUResourceID.Eq(req.VGPUResourceID)).First()
	if existing != nil {
		resputil.BadRequestError(c, "relationship already exists between these resources")
		return
	}

	// Create VGPU relationship
	vgpuLink := &model.ResourceVGPU{
		GPUResourceID:  gpuResource.ID,
		VGPUResourceID: req.VGPUResourceID,
		Min:            req.Min,
		Max:            req.Max,
		Description:    req.Description,
	}

	err = rv.WithContext(c).Create(vgpuLink)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create GPU-VGPU relationship: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, vgpuLink)
}

// GetGPUVGPUResources godoc
//
//	@Summary		Get all VGPU resources linked to a GPU resource
//	@Description	This API will return all VGPU resources linked to the specified GPU resource.
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		uint					true	"GPU Resource ID"
//	@Success		200	{object}	resputil.Response[any]	"Success"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/resources/{id}/vgpu [get]
func (mgr *ResourceMgr) GetGPUVGPUResources(c *gin.Context) {
	var req ResourcePathReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Success(c, []model.ResourceVGPU{})
		return
	}

	// Verify GPU resource exists and is of type GPU
	r := query.Resource
	gpuResource, err := r.WithContext(c).Where(r.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.Success(c, []model.ResourceVGPU{})
		return
	}

	if gpuResource.Type == nil || *gpuResource.Type != model.ResourceTypeGPU {
		resputil.Success(c, []model.ResourceVGPU{})
		return
	}

	// Get all VGPU links for this GPU
	rv := query.ResourceVGPU
	vgpuLinks, err := rv.WithContext(c).
		Preload(rv.VGPUResource).
		Where(rv.GPUResourceID.Eq(req.ID)).
		Find()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to retrieve VGPU resources: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, vgpuLinks)
}

type UpdateGPUVGPULinkReq struct {
	VGPUResourceID uint    `json:"vgpuResourceId,omitempty"`
	Min            *int    `json:"min,omitempty"`
	Max            *int    `json:"max,omitempty"`
	Description    *string `json:"description,omitempty"`
}

type VGPUPathReq struct {
	ID     uint `uri:"id" binding:"required"`
	VGPUId uint `uri:"vgpuId" binding:"required"`
}

// UpdateGPUVGPULink godoc
//
//	@Summary		Update a GPU-VGPU resource relationship
//	@Description	This API will update the relationship between a GPU resource and a VGPU resource.
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id			path		uint					true	"GPU Resource ID"
//	@Param			vgpuId		path		uint					true	"VGPU Link ID"
//	@Param			vgpuRequest	body		UpdateGPUVGPULinkReq	true	"VGPU resource configuration to update"
//	@Success		200			{object}	resputil.Response[any]	"Success"
//	@Failure		400			{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500			{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/resources/{id}/vgpu/{vgpuId} [put]
func (mgr *ResourceMgr) UpdateGPUVGPULink(c *gin.Context) {
	var pathReq VGPUPathReq
	if err := c.ShouldBindUri(&pathReq); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	var req UpdateGPUVGPULinkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	// Validate range
	if req.Min != nil && req.Max != nil {
		if *req.Min > *req.Max {
			resputil.BadRequestError(c, "min cannot be greater than max")
			return
		}
	}

	rv := query.ResourceVGPU
	updates := make(map[string]any)

	if req.VGPUResourceID != 0 {
		// Verify the new VGPU resource exists
		r := query.Resource
		vgpuResource, err := r.WithContext(c).Where(r.ID.Eq(req.VGPUResourceID)).First()
		if err != nil {
			resputil.Error(c, fmt.Sprintf("failed to find VGPU resource: %v", err), resputil.NotSpecified)
			return
		}
		if vgpuResource.Type == nil || *vgpuResource.Type != model.ResourceTypeVGPU {
			resputil.BadRequestError(c, "specified resource is not a VGPU resource")
			return
		}
		updates["vgpu_resource_id"] = req.VGPUResourceID
	}
	if req.Min != nil {
		updates["min"] = *req.Min
	}
	if req.Max != nil {
		updates["max"] = *req.Max
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if len(updates) == 0 {
		resputil.BadRequestError(c, "no fields to update")
		return
	}

	_, err := rv.WithContext(c).Where(rv.ID.Eq(pathReq.VGPUId), rv.GPUResourceID.Eq(pathReq.ID)).Updates(updates)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to update GPU-VGPU relationship: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, nil)
}

// DeleteGPUVGPULink godoc
//
//	@Summary		Delete a GPU-VGPU resource relationship
//	@Description	This API will delete the relationship between a GPU resource and a VGPU resource.
//	@Tags			Resource
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path		uint					true	"GPU Resource ID"
//	@Param			vgpuId	path		uint					true	"VGPU Link ID"
//	@Success		200		{object}	resputil.Response[any]	"Success"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/resources/{id}/vgpu/{vgpuId} [delete]
func (mgr *ResourceMgr) DeleteGPUVGPULink(c *gin.Context) {
	var req VGPUPathReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	rv := query.ResourceVGPU
	_, err := rv.WithContext(c).Where(rv.ID.Eq(req.VGPUId), rv.GPUResourceID.Eq(req.ID)).Delete()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to delete GPU-VGPU relationship: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, nil)
}
