package handler

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/samber/lo"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewAccountMgr)
}

type AccountMgr struct {
	name   string
	client client.Client
}

func NewAccountMgr(conf RegisterConfig) Manager {
	return &AccountMgr{
		name:   "accounts",
		client: conf.Client,
	}
}

func (mgr *AccountMgr) GetName() string { return mgr.name }

func (mgr *AccountMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AccountMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.ListForUser) // 获取当前用户可访问的账户
}

func (mgr *AccountMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListForAdmin)
	g.POST("", mgr.CreateAccount)
	g.GET(":aid", mgr.GetAccountByID)
	g.GET(":aid/quota", mgr.GetQuota)
	g.PUT(":aid", mgr.UpdateAccount)
	g.DELETE(":aid", mgr.DeleteAccount)
	g.POST("add/:aid/:uid", mgr.AddUserProject)
	g.POST("update/:aid/:uid", mgr.UpdateUserProject)
	g.GET("userIn/:aid", mgr.GetUserInProject)
	g.GET("userOutOf/:aid", mgr.GetUserOutOfProject)
	g.DELETE(":aid/:uid", mgr.DeleteUserProject)
}

type (
	AccountResp struct {
		Name       string           `json:"name"`
		Nickname   string           `json:"nickname"`
		Role       model.Role       `json:"role"`
		AccessMode model.AccessMode `json:"access"`
		ExpiredAt  *time.Time       `json:"expiredAt"`
	}
)

// ListForUser godoc
// @Summary 获取用户的所有账户
// @Description 连接用户账户表和账户表，获取用户的所有账户的摘要信息
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[[]AccountResp] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/projects [get]
func (mgr *AccountMgr) ListForUser(c *gin.Context) {
	token := util.GetToken(c)

	a := query.Account
	ua := query.UserAccount

	// Get all projects for the user
	var projects []AccountResp
	err := ua.WithContext(c).Where(ua.UserID.Eq(token.UserID)).Select(a.Name, a.Nickname, ua.Role, ua.AccessMode, a.ExpiredAt).
		Join(a, a.ID.EqCol(ua.AccountID)).Order(a.ID.Desc()).Scan(&projects)
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
		NameLike  *string        `form:"nameLike"`                     // 部分匹配账户名称
		OrderCol  *string        `form:"orderCol"`                     // 排序字段
		Order     *payload.Order `form:"order"`                        // 排序方式（升序、降序）
	}

	// Swagger 不支持范型嵌套，定义别名
	ListAllResp struct {
		ID        uint             `json:"id"`
		Name      string           `json:"name"`
		Nickname  string           `json:"nickname"`
		Space     string           `json:"space"`
		Quota     model.QueueQuota `json:"quota"`
		ExpiredAt *time.Time       `json:"expiredAt"`
	}
)

// ListForAdmin godoc
// @Summary 获取所有账户
// @Description 获取所有账户的摘要信息，支持筛选条件、分页和排序
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query ListAllReq true "分页参数"
// @Success 200 {object} resputil.Response[any] "账户列表"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/projects [get]
func (mgr *AccountMgr) ListForAdmin(c *gin.Context) {
	q := query.Account

	queues, err := q.WithContext(c).Order(q.ID.Asc()).Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	lists := make([]ListAllResp, len(queues))
	for i := range queues {
		queue := queues[i]
		lists[i] = ListAllResp{
			ID:        queue.ID,
			Name:      queue.Name,
			Nickname:  queue.Nickname,
			Space:     queue.Space,
			Quota:     queue.Quota.Data(),
			ExpiredAt: queue.ExpiredAt,
		}
	}

	resputil.Success(c, lists)
}

type AccountIDReq struct {
	ID uint `uri:"aid" binding:"required"`
}

// GetAccountByID godoc
// @Summary 获取指定账户
// @Description 根据账户ID获取账户的信息
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param aid path AccountIDReq true "projectname"
// @Success 200 {object} resputil.Response[any] "账户信息"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/projects/{aid} [get]
func (mgr *AccountMgr) GetAccountByID(c *gin.Context) {
	var uriReq AccountIDReq
	if err := c.ShouldBindUri(&uriReq); err != nil {
		resputil.Error(c, fmt.Sprintf("invalid request, detail: %v", err), resputil.NotSpecified)
		return
	}
	q := query.Account
	queue, err := q.WithContext(c).Where(q.ID.Eq(uriReq.ID)).First()

	if err != nil {
		resputil.Error(c, fmt.Sprintf("find project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resp := ListAllResp{
		ID:        queue.ID,
		Name:      queue.Name,
		Nickname:  queue.Nickname,
		Space:     queue.Space,
		Quota:     queue.Quota.Data(),
		ExpiredAt: queue.ExpiredAt,
	}

	resputil.Success(c, resp)
}

//nolint:gocyclo // TODO(liyilong): delete other duplicated code
func (mgr *AccountMgr) GetQuota(c *gin.Context) {
	var uriReq AccountIDReq
	if err := c.ShouldBindUri(&uriReq); err != nil {
		resputil.Error(c, fmt.Sprintf("invalid request, detail: %v", err), resputil.NotSpecified)
		return
	}

	a := query.Account
	account, err := a.WithContext(c).Where(a.ID.Eq(uriReq.ID)).First()
	if err != nil {
		resputil.Error(c, "Account not found", resputil.NotSpecified)
		return
	}

	queue := scheduling.Queue{}

	if err = mgr.client.Get(c, types.NamespacedName{
		Name:      account.Name,
		Namespace: config.GetConfig().Workspace.Namespace,
	}, &queue); err != nil {
		resputil.Error(c, "Queue not found", resputil.TokenInvalid)
		return
	}

	allocated := queue.Status.Allocated
	guarantee := queue.Spec.Guarantee.Resource
	deserved := queue.Spec.Deserved
	capability := queue.Spec.Capability

	// resources is a map, key is the resource name, value is the resource amount
	resources := make(map[v1.ResourceName]payload.ResourceResp)

	for name, quantity := range allocated {
		if name == v1.ResourceCPU || name == v1.ResourceMemory || strings.Contains(string(name), "/") {
			resources[name] = payload.ResourceResp{
				Label: string(name),
				Allocated: lo.ToPtr(payload.ResourceBase{
					Amount: quantity.Value(),
					Format: string(quantity.Format),
				}),
			}
		}
	}
	for name, quantity := range guarantee {
		if v, ok := resources[name]; ok {
			v.Guarantee = lo.ToPtr(payload.ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}
	for name, quantity := range deserved {
		if v, ok := resources[name]; ok {
			v.Deserved = lo.ToPtr(payload.ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}
	for name, quantity := range capability {
		if v, ok := resources[name]; ok {
			v.Capability = lo.ToPtr(payload.ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}

	// if capability is not set, read max from db
	r := query.Resource
	for name, resource := range resources {
		if resource.Capability == nil {
			resouece, err := r.WithContext(c).Where(r.ResourceName.Eq(string(name))).First()
			if err != nil {
				continue
			}
			resource.Capability = &payload.ResourceBase{
				Amount: resouece.Amount,
				Format: resouece.Format,
			}
			resources[name] = resource
		}
	}

	// map contains cpu, memory, gpus, get them from the map
	cpu := resources[v1.ResourceCPU]
	cpu.Label = "cpu"
	memory := resources[v1.ResourceMemory]
	memory.Label = "mem"
	var gpus []payload.ResourceResp
	for name, resource := range resources {
		if strings.Contains(string(name), "/") {
			// convert nvidia.com/v100 to v100
			split := strings.Split(string(name), "/")
			if len(split) == 2 {
				resourceType := split[1]
				label := resourceType
				resource.Label = label
			}
			gpus = append(gpus, resource)
		}
	}
	sort.Slice(gpus, func(i, j int) bool {
		return gpus[i].Label < gpus[j].Label
	})

	resputil.Success(c, payload.QuotaResp{
		CPU:    cpu,
		Memory: memory,
		GPUs:   gpus,
	})
}

type (
	AccountCreateOrUpdateReq struct {
		Nickname string `json:"name" binding:"required"`
		Quota    struct {
			Guaranteed v1.ResourceList `json:"guaranteed"`
			Deserved   v1.ResourceList `json:"deserved"`
			Capability v1.ResourceList `json:"capability"`
		} `json:"quota"`
		WithoutVolcano bool      `json:"withoutVolcano"`
		ExpiredAt      time.Time `json:"ExpiredAt"`
	}

	ProjectCreateResp struct {
		ID uint `json:"id"`
	}
)

// CreateAccount godoc
// @Summary 创建团队账户
// @Description 从请求中获取账户名称、描述和配额，以当前用户为管理员，创建一个团队账户
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param data body any true "账户信息"
// @Success 200 {object} resputil.Response[ProjectCreateResp] "成功创建账户，返回账户ID"
// @Failure 400 {object} resputil.Response[any]	"请求参数错误"
// @Failure 500 {object} resputil.Response[any]	"账户创建失败，返回错误信息"
// @Router /v1/projects [post]
func (mgr *AccountMgr) CreateAccount(c *gin.Context) {
	token := util.GetToken(c)

	var req AccountCreateOrUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, err.Error(), resputil.InvalidRequest)
		return
	}

	// Create a new project, and set the user as the admin in user_project
	db := query.Use(query.GetDB())

	var queueID uint

	err := db.Transaction(func(tx *query.Query) error {
		q := tx.Account
		uq := tx.UserAccount

		// Create a queue Queue
		queue := model.Account{
			Nickname: req.Nickname,
		}
		if err := q.WithContext(c).Create(&queue); err != nil {
			return err
		}

		// Create a user-project relationship without quota limit
		userQueue := model.UserAccount{
			UserID:     token.UserID,
			AccountID:  queue.ID,
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
			Guaranteed: req.Quota.Guaranteed,
			Deserved:   req.Quota.Deserved,
			Capability: req.Quota.Capability,
		})
		if !req.ExpiredAt.IsZero() {
			queue.ExpiredAt = &req.ExpiredAt
		}
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

func (mgr *AccountMgr) CreateVolcanoQueue(c *gin.Context, token *util.JWTMessage, queue *model.Account,
	req *AccountCreateOrUpdateReq) error {
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
			Guarantee:  scheduling.Guarantee{Resource: req.Quota.Guaranteed},
			Capability: req.Quota.Capability,
			Deserved:   req.Quota.Deserved,
		},
	}

	if err := mgr.client.Create(c, volcanoQueue); err != nil {
		return err
	}

	return nil
}

// UpdateAccount godoc
// @Summary 更新配额
// @Description 更新配额
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param aid path AccountIDReq true "projectname"
// @Param data body any true "更新quota"
// @Success 200 {object} resputil.Response[string] "成功更新配额"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/projects/{aid} [put]
func (mgr *AccountMgr) UpdateAccount(c *gin.Context) {
	var req AccountCreateOrUpdateReq
	var uriReq AccountIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	if err := c.ShouldBindUri(&uriReq); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	q := query.Account
	queue, err := q.WithContext(c).Where(q.ID.Eq(uriReq.ID)).First()

	if err != nil {
		resputil.Error(c, fmt.Sprintf("find project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	// update db
	queue.Quota = datatypes.NewJSONType(model.QueueQuota{
		Guaranteed: req.Quota.Guaranteed,
		Deserved:   req.Quota.Deserved,
		Capability: req.Quota.Capability,
	})
	if !req.ExpiredAt.IsZero() {
		queue.ExpiredAt = &req.ExpiredAt
	}
	queue.Nickname = req.Nickname
	if _, err := q.WithContext(c).Where(q.ID.Eq(queue.ID)).Updates(queue); err != nil {
		resputil.Error(c, fmt.Sprintf("update project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	// update queue
	if req.WithoutVolcano {
		resputil.Success(c, fmt.Sprintf("update capability of %s", queue.Name))
		return
	}

	if err := mgr.updateVolcanoQueue(c, queue, &req); err != nil {
		resputil.Error(c, fmt.Sprintf("update capability failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("update capability of %s", queue.Name))
}

func (mgr *AccountMgr) updateVolcanoQueue(c *gin.Context, queue *model.Account, req *AccountCreateOrUpdateReq) error {
	vcQueue := &scheduling.Queue{}
	namespace := config.GetConfig().Workspace.Namespace
	err := mgr.client.Get(c, client.ObjectKey{Name: queue.Name, Namespace: namespace}, vcQueue)
	if err != nil {
		return err
	}

	vcQueue.Spec.Guarantee = scheduling.Guarantee{Resource: req.Quota.Guaranteed}
	vcQueue.Spec.Deserved = req.Quota.Deserved
	vcQueue.Spec.Capability = req.Quota.Capability

	if err := mgr.client.Update(c, vcQueue); err != nil {
		return err
	}

	return nil
}

type DeleteProjectReq struct {
	ID uint `uri:"aid" binding:"required"`
}

type DeleteProjectResp struct {
	Name string `uri:"name" binding:"required"`
}

// / DeleteAccount godoc
// @Summary 删除账户
// @Description 删除账户record和队列crd
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param aid path DeleteProjectReq true "aid"
// @Success 200 {object} resputil.Response[DeleteProjectResp] "删除的队列名"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/{aid} [delete]
func (mgr *AccountMgr) DeleteAccount(c *gin.Context) {
	var req DeleteProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate delete parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	db := query.Use(query.GetDB())

	queueID := req.ID

	uq := query.UserAccount

	// get user-queues relationship without quota limit

	if userQueues, err := uq.WithContext(c).Where(uq.AccountID.Eq(queueID)).Find(); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	} else if len(userQueues) > 0 {
		resputil.Error(c, "Still have members", resputil.InvalidRequest)
		return
	}

	var queueName string

	err := db.Transaction(func(tx *query.Query) error {
		q := tx.Account
		uq := tx.UserAccount

		// get queue in db
		queue, err := q.WithContext(c).Where(q.ID.Eq(queueID)).First()
		if err != nil {
			return err
		}
		queueName = queue.Nickname

		// get user-queues relationship without quota limit
		userQueues, err := uq.WithContext(c).Where(uq.AccountID.Eq(queue.ID)).Find()
		if err != nil {
			return err
		}

		if len(userQueues) > 0 {
			if _, err := uq.WithContext(c).Delete(userQueues...); err != nil {
				return err
			}
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
	err := mgr.client.Get(c, client.ObjectKey{Name: qName, Namespace: namespace}, queue)
	if err != nil {
		return err
	}

	if queue.Status.Running != 0 {
		return fmt.Errorf("queue still have running pod")
	}

	if err := mgr.client.Delete(c, queue); err != nil {
		return err
	}

	return nil
}

type (
	UserProjectReq struct {
		QueueID uint `uri:"aid" binding:"required"`
		UserID  uint `uri:"uid" binding:"required"`
	}

	UpdateUserProjectReq struct {
		AccessMode string          `json:"accessmode" binding:"required"`
		Role       string          `json:"role" binding:"required"`
		Quota      v1.ResourceList `json:"quota"`
	}
)

// / AddUserProject godoc
// @Summary 向账户中添加用户
// @Description 创建一个userproject
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param uid path uint true "uid"
// @Param aid path uint true "aid"
// @Param req body any true "权限角色"
// @Success 200 {object} resputil.Response[any] "返回添加的用户名和队列名"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/add/{aid}/{uid} [post]
func (mgr *AccountMgr) AddUserProject(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	q := query.Account
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
	uq := query.UserAccount

	if _, err := uq.WithContext(c).Where(uq.AccountID.Eq(req.QueueID), uq.UserID.Eq(req.UserID)).Find(); err != nil {
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

		userQueue := model.UserAccount{
			UserID:    req.UserID,
			AccountID: req.QueueID,
			//nolint:gosec // TODO(wanggz): refactor this
			Role: model.Role(role),
			//nolint:gosec // TODO(wanggz): refactor this
			AccessMode: model.AccessMode(access),
		}
		userQueue.Quota = datatypes.NewJSONType(model.QueueQuota{
			Capability: reqBody.Quota,
		})

		if err := uq.WithContext(c).Create(&userQueue); err != nil {
			resputil.Error(c, fmt.Sprintf("create UserProject failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		resputil.Success(c, fmt.Sprintf("Add User %s for %s", user.Name, queue.Nickname))
	}
}

// / UpdateUserProject godoc
// @Summary 更新账户用户
// @Description 创建一个userQueue条目
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param uid path uint true "uid"
// @Param aid path uint true "aid"
// @Param req body any true "权限角色"
// @Success 200 {object} resputil.Response[any] "返回添加的用户名和队列名"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/update/{aid}/{uid} [post]
func (mgr *AccountMgr) UpdateUserProject(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	q := query.Account
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
	uq := query.UserAccount
	userQueue, err := uq.WithContext(c).Where(uq.AccountID.Eq(req.QueueID), uq.UserID.Eq(req.UserID)).First()
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

		userQueue.Quota = datatypes.NewJSONType(model.QueueQuota{
			Capability: req.Quota,
		})
		if _, err := uq.WithContext(c).Updates(userQueue); err != nil {
			resputil.Error(c, fmt.Sprintf("update UserProject failed, detail: %v", err), resputil.NotSpecified)
			return
		}

		resputil.Success(c, fmt.Sprintf("Update User %s for %s", user.Name, queue.Nickname))
	}
}

type ProjectGetReq struct {
	ID uint `uri:"aid" binding:"required"`
}

type UserProjectGetResp struct {
	ID         uint                                    `json:"id"`
	Name       string                                  `json:"name"`
	Role       string                                  `json:"role"`
	AccessMode string                                  `json:"accessmode" gorm:"access_mode"`
	Attributes datatypes.JSONType[model.UserAttribute] `json:"userInfo"`
	Quota      datatypes.JSONType[model.QueueQuota]    `json:"quota"`
}

// / GetUserInProject godoc
// @Summary 获取账户下的用户
// @Description sql查询-join
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param aid path uint true "aid"
// @Success 200 {object} resputil.Response[any] "userQueue条目"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/userIn/{aid} [get]
func (mgr *AccountMgr) GetUserInProject(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	q := query.Account
	queue, err := q.WithContext(c).Where(q.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	u := query.User
	uq := query.UserAccount

	var resp []UserProjectGetResp
	exec := u.WithContext(c).Join(uq, uq.UserID.EqCol(u.ID)).Where(uq.DeletedAt.IsNull())
	exec = exec.Select(u.ID, u.Name, uq.Role, uq.AccessMode, uq.AccountID, u.Attributes, uq.Quota)
	if err := exec.Where(uq.AccountID.Eq(queue.ID)).Distinct().Scan(&resp); err != nil {
		resputil.Error(c, fmt.Sprintf("Get UserProject failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

// / GetUserOutOfProject godoc
// @Summary 获取账户外的用户
// @Description sql查询-subquery
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param aid path uint true "aid"
// @Success 200 {object} resputil.Response[any] "userQueue条目"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/userOutOf/{aid} [get]
func (mgr *AccountMgr) GetUserOutOfProject(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	q := query.Account
	queue, err := q.WithContext(c).Where(q.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get Project failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	u := query.User
	uq := query.UserAccount
	var uids []uint

	if err := uq.WithContext(c).Select(uq.UserID).Where(uq.AccountID.Eq(queue.ID)).Scan(&uids); err != nil {
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
// @Summary 删除账户用户
// @Description 删除对应userQueue条目
// @Tags Project
// @Accept json
// @Produce json
// @Security Bearer
// @Param uid path uint true "uid"
// @Param aid path uint true "aid"
// @Success 200 {object} resputil.Response[any] "返回添加的用户名和队列名"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/projects/update/{aid}/{uid} [delete]
func (mgr *AccountMgr) DeleteUserProject(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	q := query.Account
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
	uq := query.UserAccount
	userQueue, err := uq.WithContext(c).Where(uq.AccountID.Eq(queue.ID), uq.UserID.Eq(user.ID)).First()
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
