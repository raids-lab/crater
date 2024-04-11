package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type ProjectMgr struct {
	taskController *aitaskctl.TaskController
}

func NewProjectMgr(taskController *aitaskctl.TaskController) Manager {
	return &ProjectMgr{
		taskController: taskController,
	}
}

func (mgr *ProjectMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ProjectMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.ListAllForUser)     // 获取该用户的所有项目
	g.POST("", mgr.CreateTeamProject) // 创建团队项目（个人项目在用户注册时创建）
}

func (mgr *ProjectMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListAllForAdmin) // 获取所有项目
	g.PUT("/:name/quotas", mgr.UpdateQuota)
}

// ListAllForUser godoc
// @Summary 获取用户的所有项目
// @Description 连接用户项目表和项目表，获取用户的所有项目的摘要信息
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[[]payload.ProjectResp] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/projects [get]
func (mgr *ProjectMgr) ListAllForUser(c *gin.Context) { // Check if the user exists
	token, _ := util.GetToken(c)

	p := query.Project
	up := query.UserProject

	// Get all projects for the user
	var projects []payload.ProjectResp
	err := up.WithContext(c).Where(up.UserID.Eq(token.UserID)).Select(p.ID, p.Name, up.Role, p.IsPersonal, p.Status).
		Join(p, p.ID.EqCol(up.ProjectID)).Order(p.ID.Desc()).Scan(&projects)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, projects)
}

type (
	ListAllReq struct {
		// 分页参数
		PageIndex *int `form:"page_index" binding:"required"`
		PageSize  *int `form:"page_size" binding:"required"`
		// 筛选、排序参数
		IsPersonal *bool          `form:"is_personal"`
		Status     *uint8         `form:"status"` // Status is a uint8 type
		NameLike   *string        `form:"name_like"`
		OrderCol   *string        `form:"order_col"`
		Order      *payload.Order `form:"order"`
	}

	// Swagger 不支持范型嵌套，定义别名
	ListAllResp payload.ListResp[*model.Project]
)

// ListAllForAdmin godoc
// @Summary 获取所有项目
// @Description 获取所有项目的摘要信息，支持筛选条件、分页和排序
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query ListAllReq true "分页参数"
// @Success 200 {object} resputil.Response[ListAllResp] "项目列表"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/projects [get]
func (mgr *ProjectMgr) ListAllForAdmin(c *gin.Context) {
	var req ListAllReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.InvalidRequest)
		return
	}

	p := query.Project
	pi := p.WithContext(c)

	// Filter
	if req.IsPersonal != nil {
		pi = pi.Where(p.IsPersonal.Is(*req.IsPersonal))
	}
	if req.Status != nil {
		pi = pi.Where(p.Status.Eq(*req.Status))
	}
	if req.NameLike != nil {
		pi = pi.Where(p.Name.Like("%" + *req.NameLike + "%"))
	}

	// Order
	if req.OrderCol != nil && req.Order != nil {
		orderCol, ok := p.GetFieldByName(*req.OrderCol)
		if ok {
			if *req.Order == payload.Asc {
				pi = pi.Order(orderCol.Asc())
			} else {
				pi = pi.Order(orderCol.Desc())
			}
		}
	}

	// Limit and offset
	projects, count, err := pi.FindByPage((*req.PageIndex)*(*req.PageSize), *req.PageSize)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resp := ListAllResp{Projects: projects, Count: count}

	resputil.Success(c, resp)
}

type (
	QuotaReq struct {
		CPU     *int `json:"cpu" binding:"required"`
		Memory  *int `json:"memory" binding:"required"`
		GPU     *int `json:"gpu" binding:"required"`
		Storage *int `json:"storage" binding:"required"`
	}

	ProjectCreateReq struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description" binding:"required"`
		Quota       QuotaReq `json:"quota" binding:"required"`
	}

	ProjectCreateResp struct {
		ID uint `json:"id"`
	}
)

// CreateTeamProject godoc
// @Summary 创建团队项目
// @Description 从请求中获取项目名称、描述和配额，以当前用户为管理员，创建一个团队项目
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param data body ProjectCreateReq true "项目信息"
// @Success 200 {object} resputil.Response[ProjectCreateResp] "成功创建项目，返回项目ID"
// @Failure 400 {object} resputil.Response[any]	"请求参数错误"
// @Failure 500 {object} resputil.Response[any]	"项目创建失败，返回错误信息"
// @Router /v1/projects [post]
func (mgr *ProjectMgr) CreateTeamProject(c *gin.Context) {
	token, _ := util.GetToken(c)

	var req ProjectCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}

	// Create a new project, and set the user as the admin in user_project
	db := query.Use(query.DB)

	var projectID uint

	err := db.Transaction(func(tx *query.Query) error {
		p := tx.Project
		up := tx.UserProject
		s := tx.Space

		// Create a project
		quota := model.QuotaDefault
		quota.CPUReq = *req.Quota.CPU
		quota.CPU = *req.Quota.CPU
		quota.MemReq = *req.Quota.Memory
		quota.Mem = *req.Quota.Memory
		quota.GPUReq = *req.Quota.GPU
		quota.GPU = *req.Quota.GPU
		quota.Storage = *req.Quota.Storage

		project := model.Project{
			Name:          req.Name,
			Description:   &req.Description,
			Namespace:     config.GetConfig().Workspace.Namespace,
			Status:        model.StatusActive, // todo: change to pending
			IsPersonal:    false,
			EmbeddedQuota: quota,
		}
		if err := p.WithContext(c).Create(&project); err != nil {
			return err
		}

		// Create a user-project relationship without quota limit
		userProject := model.UserProject{
			UserID:        token.UserID,
			ProjectID:     project.ID,
			Role:          model.RoleAdmin, // Set the user as the admin
			EmbeddedQuota: model.QuotaUnlimited,
		}
		if err := up.WithContext(c).Create(&userProject); err != nil {
			return err
		}

		// Create a space for the project, folder path is generated by uuid
		uuidStr := uuid.New().String()
		folderPath := fmt.Sprintf("/%s-%d", uuidStr[:8], project.ID)
		space := model.Space{
			ProjectID: project.ID,
			Path:      folderPath,
		}
		if err := s.WithContext(c).Create(&space); err != nil {
			return err
		}

		projectID = project.ID

		return nil
	})

	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	} else {
		resputil.Success(c, ProjectCreateResp{ID: projectID})
	}
}

type UpdateQuotaReq struct {
	JobReq *int `json:"jobReq"`
	Job    *int `json:"job"`

	NodeReq *int `json:"nodeReq"`
	Node    *int `json:"node"`

	CPUReq *int `json:"cpuReq"`
	CPU    *int `json:"cpu"`

	GPUReq *int `json:"gpuReq"`
	GPU    *int `json:"gpu"`

	MemReq *int `json:"memReq"`
	Mem    *int `json:"mem"`

	GPUMemReq *int `json:"gpuMemReq"`
	GPUMem    *int `json:"gpuMem"`

	Storage *int    `json:"storage"`
	Extra   *string `json:"extra"`
}

type ProjectNameReq struct {
	Name string `uri:"name" binding:"required"`
}

// UpdateQuota godoc
// @Summary 更新配额
// @Description 更新配额
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path ProjectNameReq true "projectname"
// @Param data body UpdateQuotaReq true "更新quota"
// @Success 200 {object} resputil.Response[string] "成功更新配额"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/projects/{name}/quotas [put]
func (mgr *ProjectMgr) UpdateQuota(c *gin.Context) {
	var req UpdateQuotaReq
	var nameReq ProjectNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	if err := c.ShouldBindUri(&nameReq); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	name := nameReq.Name
	p := query.Project
	_, err := p.WithContext(c).Where(p.Name.Eq(name)).Updates(map[string]any{
		"job_req":     *req.JobReq,
		"job":         *req.Job,
		"node_req":    *req.NodeReq,
		"node":        *req.Node,
		"cpu_req":     *req.CPUReq,
		"cpu":         *req.CPU,
		"gpu_req":     *req.GPUReq,
		"gpu":         *req.GPU,
		"mem_req":     *req.MemReq,
		"mem":         *req.Mem,
		"gpu_mem_req": *req.GPUMemReq,
		"gpu_mem":     *req.GPUMem,
		"storage":     *req.Storage,
		"extra":       req.Extra,
	})
	if err != nil {
		resputil.Error(c, fmt.Sprintf("update quota failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	project, err := p.WithContext(c).Where(p.Name.Eq(name)).First()

	if err != nil {
		resputil.Error(c, fmt.Sprintf("find project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	ProjectID := strconv.FormatUint(uint64(project.ID), 10)

	q1 := v1.ResourceList{
		v1.ResourceCPU:                    *resource.NewScaledQuantity(int64(*req.CPU), resource.Milli),
		v1.ResourceMemory:                 *resource.NewScaledQuantity(int64(*req.Mem), resource.Giga),
		v1.ResourceName("nvidia.com/gpu"): *resource.NewQuantity(int64(*req.GPU), resource.DecimalSI),
	}
	oldquota := models.Quota{
		UserName:  name,
		NameSpace: config.GetConfig().Workspace.Namespace,
		HardQuota: models.ResourceListToJSON(q1),
	}
	mgr.taskController.AddOrUpdateQuotaInfo(ProjectID, &oldquota)

	logutils.Log.Infof("update quota success, project: %s, quota:%v", name, oldquota.HardQuota)
	resputil.Success(c, "")
}
