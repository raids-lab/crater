package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type UserMgr struct {
	taskController *aitaskctl.TaskController
}

func NewUserMgr(taskController *aitaskctl.TaskController) Manager {
	return &UserMgr{
		taskController: taskController,
	}
}

func (mgr *UserMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *UserMgr) RegisterProtected(_ *gin.RouterGroup) {}

func (mgr *UserMgr) RegisterAdmin(users *gin.RouterGroup) {
	users.GET("", mgr.ListUser)
	users.DELETE("/:name", mgr.DeleteUser)
	users.PUT("/:name/role", mgr.UpdateRole)
	users.PUT("/:name/quotas", mgr.UpdateQuota)
}

type UserResp struct {
	ID     uint         `json:"id"`     // 用户ID
	Name   string       `json:"name"`   // 用户名称
	Role   model.Role   `json:"role"`   // 用户角色
	Status model.Status `json:"status"` // 用户状态
	// 私人Quota，包含Job、Node等
	Quota payload.Quota `json:"quota"`
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

type UpdateRoleReq struct {
	Role model.Role `json:"role" binding:"required"`
}

// DeleteUser godoc
// @Summary 删除用户
// @Description 删除用户
// @Tags User
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "username"
// @Success 200 {object} resputil.Response[string] "删除成功"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/users/{name} [delete]
func (mgr *UserMgr) DeleteUser(c *gin.Context) {
	name := c.Param("name")
	u := query.User
	_, err := u.WithContext(c).Where(u.Name.Eq(name)).Delete()

	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete user failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	// TODO: delete resource
	logutils.Log.Infof("delete user success, username: %s", name)
	resputil.Success(c, "")
}

// ListUser godoc
// @Summary 列出用户信息
// @Description 列出用户信息（包含私人配额）
// @Tags User
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[[]UserResp] "成功获取用户信息"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/users [get]
func (mgr *UserMgr) ListUser(c *gin.Context) {
	u := query.User
	p := query.Project
	up := query.UserProject
	q := query.Quota

	userProjects, err := up.WithContext(c).Join(p, p.ID.EqCol(up.ProjectID), p.IsPersonal.Is(true)).Join(u, u.ID.EqCol(up.UserID)).
		Select(u.ID, u.Name, u.Role, u.Status, up.ALL).Find()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list user failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	usersWithQuota := make([]UserResp, 0)
	for i := range userProjects {
		userProject := userProjects[i]
		user, err := u.WithContext(c).Where(u.ID.Eq(userProject.UserID)).First()
		if err != nil {
			resputil.Error(c, fmt.Sprintf("find user failed, detail: %v", err), resputil.NotSpecified)
			return
		}
		userResp := UserResp{
			ID:     user.ID,
			Name:   user.Name,
			Role:   user.Role,
			Status: user.Status, // 用户状态
		}
		var quota payload.Quota
		err = q.WithContext(c).Where(q.ProjectID.Eq(userProject.ProjectID)).Select(q.ALL).Scan(&quota)
		if err == nil {
			userResp.Quota = quota
		}
		usersWithQuota = append(usersWithQuota, userResp)
	}
	logutils.Log.Infof("list users success, taskNum: %d", len(usersWithQuota))
	resputil.Success(c, usersWithQuota)
}

// UpdateQuota godoc
// @Summary 更新配额
// @Description 更新配额
// @Tags User
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "username"
// @Param data body UpdateQuotaReq true "更新quota"
// @Success 200 {object} resputil.Response[string] "成功更新配额"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/users/{name}/quotas [put]
func (mgr *UserMgr) UpdateQuota(c *gin.Context) {
	var req UpdateQuotaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	name := c.Param("name")
	p := query.Project
	u := query.User
	up := query.UserProject
	q := query.Quota
	user, err := u.WithContext(c).Where(u.Name.Eq(name)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get user failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	userProject, err := up.WithContext(c).Join(p, p.ID.EqCol(up.ProjectID), p.IsPersonal.Is(true)).Where(up.UserID.Eq(user.ID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get userProject failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	_, err = q.WithContext(c).Where(q.ProjectID.Eq(userProject.ProjectID)).Updates(map[string]any{
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

	UserProjectID := strconv.FormatUint(uint64(user.ID), 10) + "-" + strconv.FormatUint(uint64(userProject.ProjectID), 10)

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
	mgr.taskController.AddOrUpdateQuotaInfo(UserProjectID, &oldquota)

	logutils.Log.Infof("update quota success, user: %s, quota:%v", name, oldquota.HardQuota)
	resputil.Success(c, "")
}

// UpdateRole godoc
// @Summary 更新角色
// @Description 更新角色
// @Tags User
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "username"
// @Param role body UpdateRoleReq true "role"
// @Success 200 {object} resputil.Response[string] "更新角色成功"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/users/{name}/role [put]
func (mgr *UserMgr) UpdateRole(c *gin.Context) {
	var req UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	name := c.Param("name")
	if req.Role < 1 || req.Role > 3 {
		resputil.Error(c, fmt.Sprintf("role value exceeds the allowed range 1-3,detail: Role is %s,out of range", req.Role),
			resputil.NotSpecified)
		return
	}
	u := query.User
	_, err := u.WithContext(c).Where(u.Name.Eq(name)).Update(u.Role, req.Role)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("update user role failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	logutils.Log.Infof("update user role success, user: %s, role: %v", name, req.Role)

	resputil.Success(c, "")
}
