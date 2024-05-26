package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

type ProjectMgr struct {
	taskController *aitaskctl.TaskController
	client.Client
}

func NewProjectMgr(taskController *aitaskctl.TaskController, cl client.Client) Manager {
	return &ProjectMgr{
		taskController: taskController,
		Client:         cl,
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
	g.DELETE("/:pid", mgr.DeleteProject)
	g.POST("/add/:pid/:uid", mgr.AddUserProject)
	g.POST("/update/:pid/:uid", mgr.UpdateUserProject)
	g.GET("/userIn/:pid", mgr.GetUserInProject)
	g.GET("/userOutOf/:pid", mgr.GetUserOutOfProject)
	g.DELETE("/:pid/:uid", mgr.DeleteUserProject)
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
	token := util.GetToken(c)

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
		PageIndex  *int           `form:"pageIndex" binding:"required"` // 第几页（从0开始）
		PageSize   *int           `form:"pageSize" binding:"required"`  // 每页大小
		IsPersonal *bool          `form:"isPersonal"`                   // 是否为个人项目
		Status     *uint8         `form:"status"`                       // 项目状态（pending, active, inactive）
		NameLike   *string        `form:"nameLike"`                     // 部分匹配项目名称
		OrderCol   *string        `form:"orderCol"`                     // 排序字段
		Order      *payload.Order `form:"order"`                        // 排序方式（升序、降序）
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

	resp := ListAllResp{Rows: projects, Count: count}

	resputil.Success(c, resp)
}

type (
	QuotaReq struct {
		CPU     *int `json:"cpu" binding:"required"`     // decimal
		Memory  *int `json:"memory" binding:"required"`  // giga
		GPU     *int `json:"gpu" binding:"required"`     // decimal
		Storage *int `json:"storage" binding:"required"` // giga
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
	token := util.GetToken(c)

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
		folderPath := fmt.Sprintf("/q-%d", project.ID)
		space := model.Space{
			ProjectID: project.ID,
			Path:      folderPath,
		}
		if err := s.WithContext(c).Create(&space); err != nil {
			return err
		}

		err := mgr.CreateTeamQueue(c, &token, &project, &req, &quota)
		if err != nil {
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

func (mgr *ProjectMgr) CreateTeamQueue(c *gin.Context, token *util.JWTMessage, project *model.Project,
	req *ProjectCreateReq, quota *model.EmbeddedQuota) error {
	// Create a new queue, and set the user as the admin in user_queue
	db := query.Use(query.DB)
	pid := project.ID
	err := db.Transaction(func(tx *query.Query) error {
		q := tx.Queue
		uq := tx.UserQueue

		queue := model.Queue{
			Name:     fmt.Sprintf("q-%d", pid),
			Nickname: req.Name,
			Space:    fmt.Sprintf("/q-%d", pid),
		}
		if err := q.WithContext(c).Create(&queue); err != nil {
			return err
		}

		// Create a user-project relationship without quota limit
		UserQueue := model.UserQueue{
			UserID:     token.UserID,
			QueueID:    queue.ID,
			Role:       model.RoleAdmin, // Set the user as the admin
			AccessMode: model.AccessModeRW,
		}
		if err := uq.WithContext(c).Create(&UserQueue); err != nil {
			return err
		}

		namespace := config.GetConfig().Workspace.Namespace
		labels := map[string]string{
			LabelKeyQueueCreatedBy: token.Username,
		}
		capability := v1.ResourceList{
			v1.ResourceCPU:                    *resource.NewQuantity(int64(quota.CPU), resource.DecimalSI),
			v1.ResourceMemory:                 *resource.NewScaledQuantity(int64(quota.Mem), resource.Giga),
			v1.ResourceName("nvidia.com/gpu"): *resource.NewQuantity(int64(quota.GPU), resource.DecimalSI),
		}
		volcanoQueue := &scheduling.Queue{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("q-%d", pid),
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: scheduling.QueueSpec{
				Capability: capability,
			},
		}

		if err := mgr.Client.Create(c, volcanoQueue); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}
	return nil
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

	capability := &v1.ResourceList{
		v1.ResourceCPU:                    *resource.NewQuantity(int64(*req.CPU), resource.DecimalSI),
		v1.ResourceMemory:                 *resource.NewScaledQuantity(int64(*req.Mem), resource.Giga),
		v1.ResourceName("nvidia.com/gpu"): *resource.NewQuantity(int64(*req.GPU), resource.DecimalSI),
	}

	if err := mgr.UpdateQueueCapcity(c, project, capability); err != nil {
		resputil.Error(c, fmt.Sprintf("update capability failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("update capability to %s", model.ResourceListToJSON(*capability)))
}

func (mgr *ProjectMgr) UpdateQueueCapcity(c *gin.Context, project *model.Project, capability *v1.ResourceList) error {
	pid := project.ID
	queue := &scheduling.Queue{}
	namespace := config.GetConfig().Workspace.Namespace
	err := mgr.Get(c, client.ObjectKey{Name: fmt.Sprintf("q-%d", pid), Namespace: namespace}, queue)
	if err != nil {
		return err
	}
	queue.Spec.Capability = *capability

	if err := mgr.Update(c, queue); err != nil {
		return err
	}

	return nil
}

type DeleteProjectReq struct {
	ID uint `uri:"pid" binding:"required"`
}

type DeleteProjectResp struct {
	Name string `uri:"name" binding:"required"`
}

func (mgr *ProjectMgr) DeleteProject(c *gin.Context) {
	var req DeleteProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate delete parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	db := query.Use(query.DB)

	projectID := req.ID
	var projectName string

	err := db.Transaction(func(tx *query.Query) error {
		p := tx.Project
		up := tx.UserProject
		q := tx.Queue
		uq := tx.UserQueue
		s := tx.Space

		// get project
		project, err := p.WithContext(c).Where(p.ID.Eq(projectID)).First()
		if err != nil {
			return err
		}
		projectName = project.Name
		// get user-project relationship without quota limit
		userProjects, err := up.WithContext(c).Where(up.ProjectID.Eq(projectID)).Find()
		if err != nil {
			return err
		}
		// get queue in db
		queue, err := q.WithContext(c).Where(q.Name.Eq(fmt.Sprintf("q-%d", projectID))).First()
		if err != nil {
			return err
		}

		// get user-project relationship without quota limit
		userQueues, err := uq.WithContext(c).Where(uq.QueueID.Eq(queue.ID)).Find()
		if err != nil {
			return err
		}

		// get space for the project

		spaces, err := s.WithContext(c).Where(s.ProjectID.Eq(projectID)).Find()
		if err != nil {
			return err
		}

		if _, err := s.WithContext(c).Delete(spaces...); err != nil {
			return err
		}

		if _, err := up.WithContext(c).Delete(userProjects...); err != nil {
			return err
		}

		if _, err := p.WithContext(c).Delete(project); err != nil {
			return err
		}

		if _, err := uq.WithContext(c).Delete(userQueues...); err != nil {
			return err
		}

		if _, err := q.WithContext(c).Delete(queue); err != nil {
			return err
		}

		if err := mgr.DeleteQueue(c, fmt.Sprintf("q-%d", projectID)); err != nil {
			return err
		}

		projectID = project.ID

		return nil
	})

	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	} else {
		resputil.Success(c, DeleteProjectResp{Name: projectName})
	}
}

func (mgr *ProjectMgr) DeleteQueue(c *gin.Context, qName string) error {
	queue := &scheduling.Queue{}
	namespace := config.GetConfig().Workspace.Namespace
	err := mgr.Client.Get(c, client.ObjectKey{Name: qName, Namespace: namespace}, queue)
	if err != nil {
		return err
	}

	if queue.Status.Running != 0 {
		return err
	}

	if err := mgr.Client.Delete(c, queue); err != nil {
		return err
	}

	return nil
}

type (
	UserProjectReq struct {
		ProjectID uint `uri:"pid" binding:"required"`
		UserID    uint `uri:"uid" binding:"required"`
	}

	UpdateUserProjectReq struct {
		AccessMode uint `json:"accessmode" binding:"required"`
		Role       uint `json:"role" binding:"required"`
	}
)

func (mgr *ProjectMgr) AddUserProject(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	p := query.Project
	project, err := p.WithContext(c).Where(p.ID.Eq(req.ProjectID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(req.UserID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	up := query.UserProject

	if _, err := up.WithContext(c).Where(up.ProjectID.Eq(req.ProjectID), up.UserID.Eq(req.UserID)).Find(); err != nil {
		resputil.Error(c, fmt.Sprintf("Get UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	} else {
		var reqBody UpdateUserProjectReq
		if err := c.ShouldBindJSON(&reqBody); err != nil {
			resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		userProject := model.UserProject{
			UserID:     req.UserID,
			ProjectID:  req.ProjectID,
			Role:       model.Role(reqBody.Role),
			AccessMode: model.AccessMode(reqBody.AccessMode),
		}

		if err := up.WithContext(c).Create(&userProject); err != nil {
			resputil.Error(c, fmt.Sprintf("create UserProject failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		resputil.Success(c, fmt.Sprintf("Add User %s for %s", user.Name, project.Name))
	}
}

func (mgr *ProjectMgr) UpdateUserProject(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	p := query.Project
	project, err := p.WithContext(c).Where(p.ID.Eq(req.ProjectID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(req.UserID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	up := query.UserProject
	userProject, err := up.WithContext(c).Where(up.ProjectID.Eq(req.ProjectID), up.UserID.Eq(req.UserID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	} else {
		var req UpdateUserProjectReq
		if err := c.ShouldBindJSON(&req); err != nil {
			resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
			return
		}
		userProject.Role = model.Role(req.Role)
		userProject.AccessMode = model.AccessMode(req.AccessMode)
		if _, err := up.WithContext(c).Updates(userProject); err != nil {
			resputil.Error(c, fmt.Sprintf("update UserProject failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		resputil.Success(c, fmt.Sprintf("Update User %s for %s", user.Name, project.Name))
	}
}

type ProjectGetReq struct {
	ID uint `uri:"pid" binding:"required"`
}

type UserProjectGetResp struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Role       uint   `json:"role"`
	AccessMode uint   `json:"accessmode" gorm:"access_mode"`
}

func (mgr *ProjectMgr) GetUserInProject(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	p := query.Project
	project, err := p.WithContext(c).Where(p.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	u := query.User
	up := query.UserProject
	var resp []UserProjectGetResp
	exec := u.WithContext(c).Join(up, up.UserID.EqCol(u.ID)).Where(up.DeletedAt.IsNull())
	exec = exec.Select(u.ID, u.Name, up.Role, up.AccessMode, up.ProjectID)
	if err := exec.Where(up.ProjectID.Eq(project.ID)).Distinct().Scan(&resp); err != nil {
		resputil.Error(c, fmt.Sprintf("Get UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

func (mgr *ProjectMgr) GetUserOutOfProject(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	p := query.Project
	project, err := p.WithContext(c).Where(p.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	u := query.User
	up := query.UserProject
	var resp []UserProjectGetResp
	exec := u.WithContext(c).LeftJoin(up, up.UserID.EqCol(u.ID))
	exec = exec.Where(up.UserID.IsNull()).Or(up.UserID.IsNotNull(), up.ProjectID.Neq(project.ID)).Distinct()
	if err := exec.Scan(&resp); err != nil {
		resputil.Error(c, fmt.Sprintf("Get UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

func (mgr *ProjectMgr) DeleteUserProject(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	p := query.Project
	project, err := p.WithContext(c).Where(p.ID.Eq(req.ProjectID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(req.UserID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	up := query.UserProject
	userProject, err := up.WithContext(c).Where(up.ProjectID.Eq(project.ID), up.UserID.Eq(user.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	if _, err := up.WithContext(c).Delete(userProject); err != nil {
		resputil.Error(c, fmt.Sprintf("delete UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("delete User %s for %s", user.Name, project.Name))
}
