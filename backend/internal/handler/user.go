package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/utils"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewUserMgr)
}

type UserMgr struct {
	name           string
	billingService *service.BillingService
}

func NewUserMgr(conf *RegisterConfig) Manager {
	return &UserMgr{
		name:           "users",
		billingService: conf.BillingService,
	}
}

func (mgr *UserMgr) GetName() string { return mgr.name }

func (mgr *UserMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *UserMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/:name", mgr.GetUser) // 新增获取单个用户的接口
	g.GET("/email/verified", mgr.CheckIfEmailVerified)
}

func (mgr *UserMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListUser)
	g.GET("/billing/summary", mgr.ListUserBillingSummary)
	g.GET("/baseinfo", mgr.ListUserBaseInfo)
	g.DELETE("/:name", mgr.DeleteUser)
	g.PUT("/:name/role", mgr.UpdateRole)
	g.PUT("/:name/attributes", mgr.UpdateUserAttributesByAdmin)
	g.POST("/:name/billing/extra-balance", mgr.AdjustUserExtraBalance)
	g.GET("/:name/billing/accounts", mgr.GetUserBillingAccounts)
}

type UserResp struct {
	ID           uint                                    `json:"id"`           // 用户ID
	Name         string                                  `json:"name"`         // 用户名称
	Role         model.Role                              `json:"role"`         // 用户角色
	Status       model.Status                            `json:"status"`       // 用户状态
	ExtraBalance float64                                 `json:"extraBalance"` // 用户额外点数
	Attributes   datatypes.JSONType[model.UserAttribute] `json:"attributes"`   // 用户额外属性
}

type UserDetailResp struct {
	ID           uint         `json:"id"`           // 用户ID
	Name         string       `json:"name"`         // 用户名称
	Nickname     string       `json:"nickname"`     // 用户昵称
	Role         model.Role   `json:"role"`         // 用户角色
	Status       model.Status `json:"status"`       // 用户状态
	CreatedAt    time.Time    `json:"createdAt"`    // 创建时间
	Teacher      *string      `json:"teacher"`      // 导师
	Group        *string      `json:"group"`        // 课题组
	Avatar       *string      `json:"avatar"`       // 头像
	ExtraBalance float64      `json:"extraBalance"` // 用户额外点数
}

type UserBillingAccountResp struct {
	AccountID                   uint       `json:"accountId"`
	AccountName                 string     `json:"accountName"`
	AccountNickname             string     `json:"accountNickname"`
	PeriodFreeBalance           float64    `json:"periodFreeBalance"`
	ExtraBalance                float64    `json:"extraBalance"`
	TotalAvailable              float64    `json:"totalAvailable"`
	LastIssuedAt                *time.Time `json:"lastIssuedAt"`
	NextIssueAt                 *time.Time `json:"nextIssueAt"`
	EffectiveIssueAmount        float64    `json:"effectiveIssueAmount"`
	EffectiveIssuePeriodMinutes int        `json:"effectiveIssuePeriodMinutes"`
}

type UserBillingSummaryResp struct {
	UserID           uint    `json:"userId"`
	Username         string  `json:"username"`
	ExtraBalance     float64 `json:"extraBalance"`
	PeriodFreeTotal  float64 `json:"periodFreeTotal"`
	TotalIssueAmount float64 `json:"totalIssueAmount"`
	TotalAvailable   float64 `json:"totalAvailable"`
}

func (mgr *UserMgr) isBillingFeatureEnabled(c *gin.Context) bool {
	return mgr.billingService != nil && mgr.billingService.IsFeatureEnabled(c.Request.Context())
}

type UpdateRoleReq struct {
	Role model.Role `json:"role" binding:"required"`
}

type UserNameReq struct {
	Name string `uri:"name" binding:"required"`
}

type UserBaseInfoResp struct {
	Name     string `json:"name"`     // 用户名称
	Nickname string `json:"nickname"` // 用户昵称
	Space    string `json:"space"`
}

type AdjustUserExtraBalanceReq struct {
	Delta  service.BillingAmountInput `json:"delta" binding:"required"`
	Reason string                     `json:"reason"`
}

type AdjustUserExtraBalanceResp struct {
	UserID        uint    `json:"userId"`
	Username      string  `json:"username"`
	BeforeBalance float64 `json:"beforeBalance"`
	Delta         float64 `json:"delta"`
	AfterBalance  float64 `json:"afterBalance"`
}

// DeleteUser godoc
//
//	@Summary		删除用户
//	@Description	删除用户
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string						true	"username"
//	@Success		200		{object}	resputil.Response[string]	"删除成功"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/admin/users/{name} [delete]
func (mgr *UserMgr) DeleteUser(c *gin.Context) {
	name := c.Param("name")
	u := query.User
	_, err := u.WithContext(c).Where(u.Name.Eq(name)).Delete()

	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete user failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	// TODO: delete resource
	klog.Infof("delete user success, username: %s", name)
	resputil.Success(c, "")
}

// ListUser godoc
//
//	@Summary		列出用户信息
//	@Description	列出用户信息（包含私人配额）
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"成功获取用户信息"
//	@Failure		400	{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500	{object}	resputil.Response[any]	"其他错误"
//	@Router			/v1/admin/users [get]
func (mgr *UserMgr) ListUser(c *gin.Context) {
	u := query.User
	users, err := u.WithContext(c).
		Select(u.ID, u.Name, u.Role, u.Status, u.ExtraBalance, u.Attributes).
		Order(u.ID.Desc()).
		Find()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list users failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	resp := make([]UserResp, 0, len(users))
	for i := range users {
		extraBalance := 0.0
		if mgr.isBillingFeatureEnabled(c) {
			extraBalance = service.ToDisplayPoints(users[i].ExtraBalance)
		}
		resp = append(resp, UserResp{
			ID:           users[i].ID,
			Name:         users[i].Name,
			Role:         users[i].Role,
			Status:       users[i].Status,
			ExtraBalance: extraBalance,
			Attributes:   users[i].Attributes,
		})
	}
	klog.Infof("list users success, count: %d", len(resp))
	resputil.Success(c, resp)
}

func (mgr *UserMgr) ListUserBillingSummary(c *gin.Context) {
	if !mgr.isBillingFeatureEnabled(c) {
		resputil.Success(c, []UserBillingSummaryResp{})
		return
	}

	u := query.User
	ua := query.UserAccount
	a := query.Account

	users, err := u.WithContext(c).
		Where(u.DeletedAt.IsNull()).
		Order(u.ID.Desc()).
		Select(u.ID, u.Name, u.ExtraBalance).
		Find()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list user billing summary failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	userAccounts, err := ua.WithContext(c).
		Where(ua.DeletedAt.IsNull()).
		Select(ua.UserID, ua.AccountID, ua.BillingIssueAmountOverride, ua.PeriodFreeBalance).
		Find()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list user billing summary failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	accountIDs := make([]uint, 0, len(userAccounts))
	for i := range userAccounts {
		accountIDs = append(accountIDs, userAccounts[i].AccountID)
	}

	accountsByID := make(map[uint]*model.Account, len(accountIDs))
	if len(accountIDs) > 0 {
		accounts, err := a.WithContext(c).
			Where(a.ID.In(accountIDs...), a.DeletedAt.IsNull()).
			Find()
		if err != nil {
			resputil.Error(c, fmt.Sprintf("list user billing summary failed, detail: %v", err), resputil.NotSpecified)
			return
		}
		for i := range accounts {
			account := accounts[i]
			accountsByID[account.ID] = account
		}
	}

	periodFreeByUserID := make(map[uint]int64, len(userAccounts))
	totalIssueByUserID := make(map[uint]int64, len(userAccounts))
	for i := range userAccounts {
		userAccount := userAccounts[i]
		periodFreeByUserID[userAccount.UserID] += userAccount.PeriodFreeBalance
		if account := accountsByID[userAccount.AccountID]; account != nil {
			issueAmount, _ := mgr.billingService.ResolveEffectiveIssueConfigForUserAccount(
				c.Request.Context(),
				userAccount,
				account,
			)
			totalIssueByUserID[userAccount.UserID] += issueAmount
		}
	}

	resp := make([]UserBillingSummaryResp, 0, len(users))
	for i := range users {
		user := users[i]
		periodFreeTotal := periodFreeByUserID[user.ID]
		resp = append(resp, UserBillingSummaryResp{
			UserID:           user.ID,
			Username:         user.Name,
			ExtraBalance:     service.ToDisplayPoints(user.ExtraBalance),
			PeriodFreeTotal:  service.ToDisplayPoints(periodFreeTotal),
			TotalIssueAmount: service.ToDisplayPoints(totalIssueByUserID[user.ID]),
			TotalAvailable:   service.ToDisplayPoints(user.ExtraBalance + periodFreeTotal),
		})
	}
	resputil.Success(c, resp)
}

// ListUserBaseInfo godoc
//
//	@Summary		列出用户基本信息
//	@Description	列出用户信息，姓名，昵称，用户空间
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"成功获取用户信息"
//	@Failure		400	{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500	{object}	resputil.Response[any]	"其他错误"
//	@Router			/v1/admin/users/baseinfo [get]
func (mgr *UserMgr) ListUserBaseInfo(c *gin.Context) {
	var users []UserBaseInfoResp
	u := query.User
	err := u.WithContext(c).
		Select(u.Name, u.Space, u.Nickname).
		Scan(&users)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list users failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	klog.Infof("list users success, count: %d", len(users))
	resputil.Success(c, users)
}

// GetUser godoc
//
//	@Summary		获取单个用户信息
//	@Description	获取指定用户的详细信息
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string								true	"username"
//	@Success		200		{object}	resputil.Response[UserDetailResp]	"成功获取用户信息"
//	@Failure		400		{object}	resputil.Response[any]				"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]				"其他错误"
//	@Router			/v1/users/{name} [get]
func (mgr *UserMgr) GetUser(c *gin.Context) {
	name := c.Param("name")
	token := util.GetToken(c)
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
		ID:           user.ID,
		Name:         user.Name,
		Nickname:     user.Nickname,
		Role:         user.Role,
		Status:       user.Status,
		CreatedAt:    user.CreatedAt,
		ExtraBalance: service.ToDisplayPoints(user.ExtraBalance),
	}
	if !mgr.isBillingFeatureEnabled(c) || !canViewUserExtraBalance(token, user) {
		userResp.ExtraBalance = 0
	}

	// 从 Attributes 中获取需要的字段
	data := user.Attributes.Data()
	userResp.Teacher = data.Teacher
	userResp.Group = data.Group
	userResp.Avatar = data.Avatar

	klog.Infof("get user success, username: %s", name)
	resputil.Success(c, userResp)
}

func canViewUserExtraBalance(token util.JWTMessage, user *model.User) bool {
	if user == nil {
		return false
	}
	return token.RolePlatform == model.RoleAdmin || token.UserID == user.ID
}

// UpdateRole godoc
//
//	@Summary		更新角色
//	@Description	更新角色
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		UserNameReq					true	"username"
//	@Param			data	body		UpdateRoleReq				true	"role"
//	@Success		200		{object}	resputil.Response[string]	"更新角色成功"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/admin/users/{name}/role [put]
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

	klog.Infof("update user role success, user: %s, role: %v", name, req.Role)

	resputil.Success(c, "")
}

// CheckIfEmailVerified godoc
//
//	@Summary		检查邮箱是否已验证
//	@Description	检查邮箱是否已验证
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"成功获取用户信息"
//	@Failure		400	{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500	{object}	resputil.Response[any]	"其他错误"
//	@Router			/v1/users/email/verified [get]
func (mgr *UserMgr) CheckIfEmailVerified(c *gin.Context) {
	type Resp struct {
		Verified            bool       `json:"verified"`
		LastEmailVerifiedAt *time.Time `json:"lastEmailVerifiedAt"`
	}

	token := util.GetToken(c)
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get user failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	// 检查邮箱过去半年是否验证过
	verified, last := utils.CheckEmailVerified(user.LastEmailVerifiedAt)
	resputil.Success(c, Resp{
		Verified:            verified,
		LastEmailVerifiedAt: last,
	})
}

// UpdateUserAttributesByAdmin godoc
//
//	@Summary		管理员更新用户属性
//	@Description	管理员更新指定用户的属性
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name		path		string					true	"username"
//	@Param			attributes	body		model.UserAttribute		true	"用户属性"
//	@Success		200			{object}	resputil.Response[any]	"用户属性更新成功"
//	@Failure		400			{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500			{object}	resputil.Response[any]	"其他错误"
//	@Router			/v1/admin/users/{name}/attributes [put]
func (mgr *UserMgr) UpdateUserAttributesByAdmin(c *gin.Context) {
	var nameReq UserNameReq
	if err := c.ShouldBindUri(&nameReq); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("Invalid request body: %v", err))
		return
	}
	name := nameReq.Name

	var attributes model.UserAttribute
	if err := c.ShouldBindJSON(&attributes); err != nil {
		resputil.BadRequestError(c, "Invalid request body")
		return
	}

	u := query.User
	user, err := u.WithContext(c).Where(u.Name.Eq(name)).First()
	if err != nil {
		resputil.Error(c, "User not found", resputil.NotSpecified)
		return
	}

	user.Attributes = datatypes.NewJSONType(attributes)
	user.Nickname = attributes.Nickname
	if err := u.WithContext(c).Save(user); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to update user attributes: %v", err), resputil.NotSpecified)
		return
	}

	klog.Infof("update user attributes success by admin, username: %s", name)
	resputil.Success(c, "用户属性更新成功")
}

// AdjustUserExtraBalance godoc
//
//	@Summary		调整用户额外点数
//	@Description	管理员按增量调整用户 extraBalance（可正可负），使用行锁避免并发覆盖
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string									true	"username"
//	@Param			body	body		AdjustUserExtraBalanceReq				true	"增量与原因"
//	@Success		200		{object}	resputil.Response[AdjustUserExtraBalanceResp]	"调整成功"
//	@Failure		400		{object}	resputil.Response[any]					"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]					"其他错误"
//	@Router			/v1/admin/users/{name}/billing/extra-balance [post]
func (mgr *UserMgr) AdjustUserExtraBalance(c *gin.Context) {
	var (
		nameReq UserNameReq
		req     AdjustUserExtraBalanceReq
		resp    AdjustUserExtraBalanceResp
	)
	if err := c.ShouldBindUri(&nameReq); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid uri params: %v", err))
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	err := query.GetDB().WithContext(c).Transaction(func(tx *gorm.DB) error {
		txQuery := query.Use(tx)
		u := txQuery.User
		user, err := u.WithContext(c).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(u.Name.Eq(nameReq.Name), u.DeletedAt.IsNull()).
			First()
		if err != nil {
			return err
		}

		before := user.ExtraBalance
		delta := req.Delta.MicroPoints()
		after := before + delta
		if after < 0 {
			return fmt.Errorf("extraBalance would be negative: before=%d delta=%d", before, delta)
		}
		if _, err := u.WithContext(c).
			Where(u.ID.Eq(user.ID), u.DeletedAt.IsNull()).
			Update(u.ExtraBalance, after); err != nil {
			return err
		}
		resp = AdjustUserExtraBalanceResp{
			UserID:        user.ID,
			Username:      user.Name,
			BeforeBalance: service.ToDisplayPoints(before),
			Delta:         service.ToDisplayPoints(delta),
			AfterBalance:  service.ToDisplayPoints(after),
		}
		return nil
	})
	if err != nil {
		resputil.Error(c, fmt.Sprintf("adjust user extra balance failed: %v", err), resputil.NotSpecified)
		return
	}

	klog.Infof(
		"adjust user extra balance success, user=%s delta=%.2f before=%.2f after=%.2f reason=%s",
		resp.Username,
		resp.Delta,
		resp.BeforeBalance,
		resp.AfterBalance,
		req.Reason,
	)
	resputil.Success(c, resp)
}

func (mgr *UserMgr) GetUserBillingAccounts(c *gin.Context) {
	if !mgr.isBillingFeatureEnabled(c) {
		resputil.Success(c, []UserBillingAccountResp{})
		return
	}
	var nameReq UserNameReq
	if err := c.ShouldBindUri(&nameReq); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid uri params: %v", err))
		return
	}

	u := query.User
	user, err := u.WithContext(c).
		Where(u.Name.Eq(nameReq.Name), u.DeletedAt.IsNull()).
		First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to load user: %v", err), resputil.NotSpecified)
		return
	}

	ua := query.UserAccount
	a := query.Account
	userAccounts, err := ua.WithContext(c).
		Where(ua.UserID.Eq(user.ID), ua.DeletedAt.IsNull()).
		Order(ua.AccountID).
		Find()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to query user billing accounts: %v", err), resputil.NotSpecified)
		return
	}

	accountIDs := make([]uint, 0, len(userAccounts))
	for i := range userAccounts {
		accountIDs = append(accountIDs, userAccounts[i].AccountID)
	}

	accountsByID := make(map[uint]*model.Account, len(accountIDs))
	if len(accountIDs) > 0 {
		accounts, err := a.WithContext(c).
			Where(a.ID.In(accountIDs...), a.DeletedAt.IsNull()).
			Find()
		if err != nil {
			resputil.Error(c, fmt.Sprintf("failed to query user billing accounts: %v", err), resputil.NotSpecified)
			return
		}
		for i := range accounts {
			accountsByID[accounts[i].ID] = accounts[i]
		}
	}

	resp := make([]UserBillingAccountResp, 0, len(userAccounts))
	for i := range userAccounts {
		row := userAccounts[i]
		account := accountsByID[row.AccountID]
		if account == nil {
			continue
		}
		issueAmount, issuePeriod := mgr.billingService.ResolveEffectiveIssueConfigForUserAccount(
			c.Request.Context(),
			row,
			account,
		)
		nextIssueAt := mgr.billingService.ComputeNextIssueAt(account.BillingLastIssuedAt, issuePeriod, time.Now())
		resp = append(resp, UserBillingAccountResp{
			AccountID:                   account.ID,
			AccountName:                 account.Name,
			AccountNickname:             account.Nickname,
			PeriodFreeBalance:           service.ToDisplayPoints(row.PeriodFreeBalance),
			ExtraBalance:                service.ToDisplayPoints(user.ExtraBalance),
			TotalAvailable:              service.ToDisplayPoints(row.PeriodFreeBalance + user.ExtraBalance),
			LastIssuedAt:                account.BillingLastIssuedAt,
			NextIssueAt:                 nextIssueAt,
			EffectiveIssueAmount:        service.ToDisplayPoints(issueAmount),
			EffectiveIssuePeriodMinutes: issuePeriod,
		})
	}

	resputil.Success(c, resp)
}
