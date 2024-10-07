package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

type AccountMgr struct {
	name string
	client.Client
}

func NewAccountMgr(cl client.Client) Manager {
	return &AccountMgr{
		name:   "projects",
		Client: cl,
	}
}

func (mgr *AccountMgr) GetName() string {
	return mgr.name
}

func (mgr *AccountMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AccountMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.ListAllForUser) // 获取该用户的所有项目
	g.POST("", mgr.CreateAccount) // 创建团队项目（个人项目在用户注册时创建）
}

func (mgr *AccountMgr) RegisterAdmin(g *gin.RouterGroup) {
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
func (mgr *AccountMgr) ListAllForUser(c *gin.Context) { // Check if the user exists
	token := util.GetToken(c)

	q := query.Queue
	uq := query.UserQueue

	// Get all projects for the user
	var projects []payload.ProjectResp
	err := uq.WithContext(c).Where(uq.UserID.Eq(token.UserID)).Select(q.ID, q.Name, uq.Role, uq.AccessMode).
		Join(q, q.ID.EqCol(uq.QueueID)).Order(q.ID.Desc()).Scan(&projects)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, projects)
}

type (
	ListAllReq struct {
		PageIndex *int           `form:"pageIndex" binding:"required"` // 第几页（从0开始）
		PageSize  *int           `form:"pageSize" binding:"required"`  // 每页大小
		NameLike  *string        `form:"nameLike"`                     // 部分匹配项目名称
		OrderCol  *string        `form:"orderCol"`                     // 排序字段
		Order     *payload.Order `form:"order"`                        // 排序方式（升序、降序）
	}

	// Swagger 不支持范型嵌套，定义别名
	ListAllResp struct {
		*model.Queue
		Guaranteed v1.ResourceList `json:"guaranteed"`
		Deserved   v1.ResourceList `json:"deserved"`
		Capacity   v1.ResourceList `json:"capacity"`
	}
)

// ListAllForAdmin godoc
// @Summary 获取所有项目
// @Description 获取所有项目的摘要信息，支持筛选条件、分页和排序
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query ListAllReq true "分页参数"
// @Success 200 {object} resputil.Response[any] "项目列表"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/projects [get]
func (mgr *AccountMgr) ListAllForAdmin(c *gin.Context) {
	var req ListAllReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.InvalidRequest)
		return
	}

	q := query.Queue
	qi := q.WithContext(c)

	// Filter
	if req.NameLike != nil {
		qi = qi.Where(q.Name.Like("%" + *req.NameLike + "%"))
	}

	// Order
	if req.OrderCol != nil && req.Order != nil {
		orderCol, ok := q.GetFieldByName(*req.OrderCol)
		if ok {
			if *req.Order == payload.Asc {
				qi = qi.Order(orderCol.Asc())
			} else {
				qi = qi.Order(orderCol.Desc())
			}
		}
	}

	// Limit and offset
	queues, count, err := qi.FindByPage((*req.PageIndex)*(*req.PageSize), *req.PageSize)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	var lists []ListAllResp
	for _, q := range queues {
		lists = append(lists,
			ListAllResp{
				Queue:      q,
				Guaranteed: q.Quota.Data().Guaranteed,
				Deserved:   q.Quota.Data().Deserved,
				Capacity:   q.Quota.Data().Capability,
			},
		)
	}

	resp := payload.ListResp[ListAllResp]{Rows: lists, Count: count}

	resputil.Success(c, resp)
}

type (
	ProjectCreateReq struct {
		Name           string          `json:"name" binding:"required"`
		Guaranteed     v1.ResourceList `json:"guaranteed" binding:"required"`
		Deserved       v1.ResourceList `json:"deserved" binding:"required"`
		Capability     v1.ResourceList `json:"capacity" binding:"required"`
		WithoutVolcano bool            `json:"withoutVolcano"`
	}

	ProjectCreateResp struct {
		ID uint `json:"id"`
	}
)

// CreateAccount godoc
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
func (mgr *AccountMgr) CreateAccount(c *gin.Context) {
	token := util.GetToken(c)

	var req ProjectCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}

	// Create a new project, and set the user as the admin in user_project
	db := query.Use(query.GetDB())

	var queueID uint

	err := db.Transaction(func(tx *query.Query) error {
		q := tx.Queue
		uq := tx.UserQueue

		// Create a queue Queue
		queue := model.Queue{
			Nickname: req.Name,
		}
		if err := q.WithContext(c).Create(&queue); err != nil {
			return err
		}

		// Create a user-project relationship without quota limit
		userQueue := model.UserQueue{
			UserID:     token.UserID,
			QueueID:    queue.ID,
			Role:       model.RoleAdmin, // Set the user as the admin
			AccessMode: model.AccessModeRW,
		}
		if err := uq.WithContext(c).Create(&userQueue); err != nil {
			return err
		}

		// Create a space for the project, folder path is generated by uuid
		queue.Name = fmt.Sprintf("q-%d", queue.ID)
		queue.Space = fmt.Sprintf("/q-%d", queue.ID)
		queue.Quota = datatypes.NewJSONType(model.QueueQuota{
			Guaranteed: req.Guaranteed,
			Deserved:   req.Deserved,
			Capability: req.Capability,
		})
		if _, err := q.WithContext(c).Where(q.ID.Eq(queue.ID)).Updates(&queue); err != nil {
			return err
		}

		queueID = queue.ID

		if req.WithoutVolcano {
			return nil
		}

		// Create a queue in Volcano
		err := mgr.CreateVolcanoQueue(c, &token, &queue, &req)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	} else {
		resputil.Success(c, ProjectCreateResp{ID: queueID})
	}
}

const (
	LabelKeyQueueCreatedBy = "crater.raids.io/queue-created-by"
)

func (mgr *AccountMgr) CreateVolcanoQueue(c *gin.Context, token *util.JWTMessage, queue *model.Queue,
	req *ProjectCreateReq) error {
	// Create a new queue, and set the user as the admin in user_queue
	namespace := config.GetConfig().Workspace.Namespace
	labels := map[string]string{
		LabelKeyQueueCreatedBy: token.Username,
	}

	volcanoQueue := &scheduling.Queue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      queue.Name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: scheduling.QueueSpec{
			Guarantee:  scheduling.Guarantee{Resource: req.Guaranteed},
			Capability: req.Capability,
			Deserved:   req.Deserved,
		},
	}

	if err := mgr.Client.Create(c, volcanoQueue); err != nil {
		return err
	}

	return nil
}

type UpdateQuotaReq struct {
	Guaranteed     v1.ResourceList `json:"guaranteed" binding:"required"`
	Deserved       v1.ResourceList `json:"deserved" binding:"required"`
	Capability     v1.ResourceList `json:"capacity" binding:"required"`
	WithoutVolcano bool            `json:"withoutVolcano"`
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
func (mgr *AccountMgr) UpdateQuota(c *gin.Context) {
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
	q := query.Queue
	queue, err := q.WithContext(c).Where(q.Nickname.Eq(name)).First()

	if err != nil {
		resputil.Error(c, fmt.Sprintf("find project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	// update db
	queue.Quota = datatypes.NewJSONType(model.QueueQuota{
		Guaranteed: req.Guaranteed,
		Deserved:   req.Deserved,
		Capability: req.Capability,
	})
	if _, err := q.WithContext(c).Where(q.ID.Eq(queue.ID)).Updates(queue); err != nil {
		resputil.Error(c, fmt.Sprintf("update project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	// update queue
	if req.WithoutVolcano {
		resputil.Success(c, fmt.Sprintf("update capability of %s", name))
		return
	}

	if err := mgr.updateVolcanoQueue(c, queue, &req); err != nil {
		resputil.Error(c, fmt.Sprintf("update capability failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("update capability of %s", name))
}

func (mgr *AccountMgr) updateVolcanoQueue(c *gin.Context, queue *model.Queue, req *UpdateQuotaReq) error {
	vcQueue := &scheduling.Queue{}
	namespace := config.GetConfig().Workspace.Namespace
	err := mgr.Get(c, client.ObjectKey{Name: queue.Name, Namespace: namespace}, vcQueue)
	if err != nil {
		return err
	}

	vcQueue.Spec.Guarantee = scheduling.Guarantee{Resource: req.Guaranteed}
	vcQueue.Spec.Deserved = req.Deserved
	vcQueue.Spec.Capability = req.Capability

	if err := mgr.Update(c, vcQueue); err != nil {
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

// / DeleteProject godoc
// @Summary 删除项目
// @Description 删除项目record和队列crd
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param pid path DeleteProjectReq true "pid"
// @Success 200 {object} resputil.Response[DeleteProjectResp] "删除的队列名"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/{pid} [delete]
func (mgr *AccountMgr) DeleteProject(c *gin.Context) {
	var req DeleteProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate delete parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	db := query.Use(query.GetDB())

	queueID := req.ID
	var queueName string

	err := db.Transaction(func(tx *query.Query) error {
		q := tx.Queue
		uq := tx.UserQueue

		// get queue in db
		queue, err := q.WithContext(c).Where(q.ID.Eq(queueID)).First()
		if err != nil {
			return err
		}
		queueName = queue.Nickname

		// get user-queues relationship without quota limit
		userQueues, err := uq.WithContext(c).Where(uq.QueueID.Eq(queue.ID)).Find()
		if err != nil {
			return err
		}

		if _, err := uq.WithContext(c).Delete(userQueues...); err != nil {
			return err
		}

		if _, err := q.WithContext(c).Delete(queue); err != nil {
			return err
		}

		if err := mgr.DeleteQueue(c, queue.Name); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	} else {
		resputil.Success(c, DeleteProjectResp{Name: queueName})
	}
}

func (mgr *AccountMgr) DeleteQueue(c *gin.Context, qName string) error {
	queue := &scheduling.Queue{}
	namespace := config.GetConfig().Workspace.Namespace
	err := mgr.Client.Get(c, client.ObjectKey{Name: qName, Namespace: namespace}, queue)
	if err != nil {
		return err
	}

	if queue.Status.Running != 0 {
		return fmt.Errorf("queue still have running pod")
	}

	if err := mgr.Client.Delete(c, queue); err != nil {
		return err
	}

	return nil
}

type (
	UserProjectReq struct {
		QueueID uint `uri:"pid" binding:"required"`
		UserID  uint `uri:"uid" binding:"required"`
	}

	UpdateUserProjectReq struct {
		AccessMode string `json:"accessmode" binding:"required"`
		Role       string `json:"role" binding:"required"`
	}
)

// / AddUserProject godoc
// @Summary 向项目中添加用户
// @Description 创建一个userproject
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param uid path uint true "uid"
// @Param pid path uint true "pid"
// @Param req body UpdateUserProjectReq true "权限角色"
// @Success 200 {object} resputil.Response[any] "返回添加的用户名和队列名"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/add/{pid}/{uid} [post]
func (mgr *AccountMgr) AddUserProject(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	q := query.Queue
	queue, err := q.WithContext(c).Where(q.ID.Eq(req.QueueID)).First()
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
	uq := query.UserQueue

	if _, err := uq.WithContext(c).Where(uq.QueueID.Eq(req.QueueID), uq.UserID.Eq(req.UserID)).Find(); err != nil {
		resputil.Error(c, fmt.Sprintf("Get UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	} else {
		var reqBody UpdateUserProjectReq
		if err = c.ShouldBindJSON(&reqBody); err != nil {
			resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		var role, access uint64

		if role, err = strconv.ParseUint(reqBody.Role, 10, 64); err != nil {
			resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		if access, err = strconv.ParseUint(reqBody.AccessMode, 10, 64); err != nil {
			resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		userQueue := model.UserQueue{
			UserID:  req.UserID,
			QueueID: req.QueueID,
			//nolint:gosec // TODO(wanggz): refactor this
			Role: model.Role(role),
			//nolint:gosec // TODO(wanggz): refactor this
			AccessMode: model.AccessMode(access),
		}

		if err := uq.WithContext(c).Create(&userQueue); err != nil {
			resputil.Error(c, fmt.Sprintf("create UserProject failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		resputil.Success(c, fmt.Sprintf("Add User %s for %s", user.Name, queue.Nickname))
	}
}

// / UpdateUserProject godoc
// @Summary 更新项目用户
// @Description 创建一个userQueue条目
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param uid path uint true "uid"
// @Param pid path uint true "pid"
// @Param req body UpdateUserProjectReq true "权限角色"
// @Success 200 {object} resputil.Response[any] "返回添加的用户名和队列名"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/update/{pid}/{uid} [post]
func (mgr *AccountMgr) UpdateUserProject(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	q := query.Queue
	queue, err := q.WithContext(c).Where(q.ID.Eq(req.QueueID)).First()
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
	uq := query.UserQueue
	userQueue, err := uq.WithContext(c).Where(uq.QueueID.Eq(req.QueueID), uq.UserID.Eq(req.UserID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	} else {
		var req UpdateUserProjectReq
		if err = c.ShouldBindJSON(&req); err != nil {
			resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		var role, access uint64

		if role, err = strconv.ParseUint(req.Role, 10, 64); err != nil {
			resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		if access, err = strconv.ParseUint(req.AccessMode, 10, 64); err != nil {
			resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		//nolint:gosec // TODO(wanggz): refactor this
		userQueue.Role = model.Role(role)
		//nolint:gosec // TODO(wanggz): refactor this
		userQueue.AccessMode = model.AccessMode(access)
		if _, err := uq.WithContext(c).Updates(userQueue); err != nil {
			resputil.Error(c, fmt.Sprintf("update UserProject failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		resputil.Success(c, fmt.Sprintf("Update User %s for %s", user.Name, queue.Nickname))
	}
}

type ProjectGetReq struct {
	ID uint `uri:"pid" binding:"required"`
}

type UserProjectGetResp struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	AccessMode string `json:"accessmode" gorm:"access_mode"`
}

// / GetUserInProject godoc
// @Summary 获取项目下的用户
// @Description sql查询-join
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param pid path uint true "pid"
// @Success 200 {object} resputil.Response[UserProjectGetResp[]] "userQueue条目"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/userIn/{pid} [post]
func (mgr *AccountMgr) GetUserInProject(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	q := query.Queue
	queue, err := q.WithContext(c).Where(q.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	u := query.User
	uq := query.UserQueue
	var resp []UserProjectGetResp
	exec := u.WithContext(c).Join(uq, uq.UserID.EqCol(u.ID)).Where(uq.DeletedAt.IsNull())
	exec = exec.Select(u.ID, u.Name, uq.Role, uq.AccessMode, uq.QueueID)
	if err := exec.Where(uq.QueueID.Eq(queue.ID)).Distinct().Scan(&resp); err != nil {
		resputil.Error(c, fmt.Sprintf("Get UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

// / GetUserOutOfProject godoc
// @Summary 获取项目外的用户
// @Description sql查询-subquery
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param pid path uint true "pid"
// @Success 200 {object} resputil.Response[UserProjectGetResp[]] "userQueue条目"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/userOutOf/{pid} [post]
func (mgr *AccountMgr) GetUserOutOfProject(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	q := query.Queue
	queue, err := q.WithContext(c).Where(q.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	u := query.User
	uq := query.UserQueue
	var uids []uint

	if err := uq.WithContext(c).Select(uq.UserID).Where(uq.QueueID.Eq(queue.ID)).Scan(&uids); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to scan user IDs: %v", err), resputil.NotSpecified)
		return
	}

	var resp []UserProjectGetResp
	exec := u.WithContext(c).Where(u.ID.NotIn(uids...)).Distinct()
	if err := exec.Scan(&resp); err != nil {
		resputil.Error(c, fmt.Sprintf("Get UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

// / DeleteUserProject godoc
// @Summary 删除项目用户
// @Description 删除对应userQueue条目
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param uid path uint true "uid"
// @Param pid path uint true "pid"
// @Success 200 {object} resputil.Response[any] "返回添加的用户名和队列名"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/update/{pid}/{uid} [delete]
func (mgr *AccountMgr) DeleteUserProject(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	q := query.Queue
	queue, err := q.WithContext(c).Where(q.ID.Eq(req.QueueID)).First()
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
	uq := query.UserQueue
	userQueue, err := uq.WithContext(c).Where(uq.QueueID.Eq(queue.ID), uq.UserID.Eq(user.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	if _, err := uq.WithContext(c).Delete(userQueue); err != nil {
		resputil.Error(c, fmt.Sprintf("delete UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("delete User %s for %s", user.Name, queue.Nickname))
}
