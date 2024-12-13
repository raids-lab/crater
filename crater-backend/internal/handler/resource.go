package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
}

func (mgr *ResourceMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.POST("/sync", mgr.SyncResource)
	g.PUT("/:id", mgr.UpdateLabel)
}

type (
	ListResourceReq struct {
		// VendorDomain of the resource in parameter (optional)
		WithVendorDomain bool `form:"withVendorDomain"`
	}
)

// ListResource godoc
// @Summary Get a list of resources based on the specified parameters
// @Description If the vendorDomain parameter is provided, the API will return a list of resources that match the specified vendor domain.
// @Tags Resource
// @Accept json
// @Produce json
// @Security Bearer
// @Param vendorDomain query string false "Vendor domain of the resource (For example: 'nvidia.com'	)"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/resources [get]
func (mgr *ResourceMgr) ListResource(c *gin.Context) {
	var req ListResourceReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	r := query.Resource
	q := r.WithContext(c).Order(r.Priority.Desc())
	if req.WithVendorDomain {
		q = q.Where(r.VendorDomain.IsNotNull())
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
// @Summary Get allocatable resources from the Kubernetes cluster and update the database
// @Description This API will get the allocatable resources from the Kubernetes cluster and update the database with the latest information.
// @Tags Resource
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/resources/sync [post]
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
		info, err := r.WithContext(c).Where(r.ResourceName.Eq(resourceName)).
			Updates(map[string]any{
				"amount":            quantity.Total.Value(),
				"amount_single_max": quantity.Max.Value(),
			})
		if err != nil {
			resputil.Error(c, fmt.Sprintf("failed to update resource: %v", err), resputil.NotSpecified)
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
			err := r.WithContext(c).Create(&newResource)
			if err != nil {
				resputil.Error(c, fmt.Sprintf("failed to create resource: %v", err), resputil.NotSpecified)
				return
			}
		}
	}

	resputil.Success(c, nil)
}

type (
	UpdateResoueceReq struct {
		Label string `json:"label" binding:"required"`
	}
	UpdateResouecePathReq struct {
		ID uint `uri:"id" binding:"required"`
	}
)

// UpdateLabel godoc
// @Summary Update the label of a resource
// @Description This API will update the label of a resource based on the specified ID.
// @Tags Resource
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path uint true "Resource ID"
// @Param label body UpdateResoueceReq true "Resource label"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/resources/{id} [put]
func (mgr *ResourceMgr) UpdateLabel(c *gin.Context) {
	var req UpdateResoueceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	var param UpdateResouecePathReq
	if err := c.ShouldBindUri(&param); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("failed to bind request: %v", err))
		return
	}

	r := query.Resource
	_, err := r.WithContext(c).Where(r.ID.Eq(param.ID)).Update(r.Label, req.Label)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to update resource: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nil)
}
