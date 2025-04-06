package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/logutils"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewUserMgr)
}

type UserMgr struct {
	name string
}

func NewUserMgr(_ *RegisterConfig) Manager {
	return &UserMgr{
		name: "users",
	}
}

func (mgr *UserMgr) GetName() string { return mgr.name }

func (mgr *UserMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *UserMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/:name", mgr.GetUser) // 新增获取单个用户的接口
}

func (mgr *UserMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListUser)
	g.DELETE("/:name", mgr.DeleteUser)
	g.PUT("/:name/role", mgr.UpdateRole)
}

type UserResp struct {
	ID         uint                                    `json:"id"`         // 用户ID
	Name       string                                  `json:"name"`       // 用户名称
	Role       model.Role                              `json:"role"`       // 用户角色
	Status     model.Status                            `json:"status"`     // 用户状态
	Attributes datatypes.JSONType[model.UserAttribute] `json:"attributes"` // 用户额外属性
}

type UserDetailResp struct {
	ID        uint         `json:"id"`        // 用户ID
	Name      string       `json:"name"`      // 用户名称
	Nickname  string       `json:"nickname"`  // 用户昵称
	Role      model.Role   `json:"role"`      // 用户角色
	Status    model.Status `json:"status"`    // 用户状态
	CreatedAt time.Time    `json:"createdAt"` // 创建时间
	Teacher   *string      `json:"teacher"`   // 导师
	Group     *string      `json:"group"`     // 课题组
	Avatar    *string      `json:"avatar"`    // 头像
}

type UpdateRoleReq struct {
	Role model.Role `json:"role" binding:"required"`
}

type UserNameReq struct {
	Name string `uri:"name" binding:"required"`
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
// @Success 200 {object} resputil.Response[any] "成功获取用户信息"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/users [get]
func (mgr *UserMgr) ListUser(c *gin.Context) {
	var users []UserResp
	u := query.User
	err := u.WithContext(c).
		Select(u.ID, u.Name, u.Role, u.Status, u.Attributes).
		Order(u.ID.Desc()).
		Scan(&users)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list users failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	logutils.Log.Infof("list users success, count: %d", len(users))
	resputil.Success(c, users)
}

// GetUser godoc
// @Summary 获取单个用户信息
// @Description 获取指定用户的详细信息
// @Tags User
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "username"
// @Success 200 {object} resputil.Response[UserDetailResp] "成功获取用户信息"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/users/{name} [get]
func (mgr *UserMgr) GetUser(c *gin.Context) {
	name := c.Param("name")
	u := query.User
	user, err := u.WithContext(c).
		Where(u.Name.Eq(name)).
		First()

	if err != nil {
		resputil.Error(c, fmt.Sprintf("get user failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	// 创建用户详情响应对象
	userResp := UserDetailResp{
		ID:        user.ID,
		Name:      user.Name,
		Nickname:  user.Nickname,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
	}

	// 从 Attributes 中获取需要的字段
	data := user.Attributes.Data()
	userResp.Teacher = data.Teacher
	userResp.Group = data.Group
	userResp.Avatar = data.Avatar

	logutils.Log.Infof("get user success, username: %s", name)
	resputil.Success(c, userResp)
}

// UpdateRole godoc
// @Summary 更新角色
// @Description 更新角色
// @Tags User
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path UserNameReq true "username"
// @Param data body UpdateRoleReq true "role"
// @Success 200 {object} resputil.Response[string] "更新角色成功"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/users/{name}/role [put]
func (mgr *UserMgr) UpdateRole(c *gin.Context) {
	var req UpdateRoleReq
	var nameReq UserNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	if err := c.ShouldBindUri(&nameReq); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	name := nameReq.Name
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
