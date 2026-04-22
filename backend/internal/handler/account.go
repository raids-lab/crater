package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/vcqueue"
)

const (
	// maxUint8Value is the maximum value for uint8 type
	maxUint8Value = 255
	// httpStatusForbidden is HTTP 403 Forbidden status code
	httpStatusForbidden = 403
)

var (
	errBillingFeatureDisabled        = errors.New("billing feature is disabled")
	errNoBillingConfigFieldsToUpdate = errors.New("no billing config fields to update")
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewAccountMgr)
}

type AccountMgr struct {
	name           string
	client         client.Client
	billingService *service.BillingService
}

func NewAccountMgr(conf *RegisterConfig) Manager {
	return &AccountMgr{
		name:           "accounts",
		client:         conf.Client,
		billingService: conf.BillingService,
	}
}

func (mgr *AccountMgr) GetName() string { return mgr.name }

func (mgr *AccountMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AccountMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.UserListAccounts)              // Get accounts accessible by current user
	g.GET("by-name/:name", mgr.GetAccountByName) // Get account by name (use specific path to avoid conflict with :aid routes)
	mgr.registerUserBillingRoutes(g)
	mgr.registerUserMemberRoutes(g)
}

func (mgr *AccountMgr) registerUserBillingRoutes(g *gin.RouterGroup) {
	g.GET(":aid/billing/config", mgr.UserGetAccountBillingConfig)
	g.PUT(":aid/billing/config", mgr.UserUpdateAccountBillingConfig)
	g.GET(":aid/billing/members", mgr.UserListAccountBillingMembers)
	g.PUT(":aid/billing/members/:uid", mgr.UserUpdateAccountBillingMemberIssueAmount)
	g.POST(":aid/billing/reset", mgr.UserResetAccountBillingBalance)
}

func (mgr *AccountMgr) registerUserMemberRoutes(g *gin.RouterGroup) {
	g.POST(":aid/users/:uid", mgr.UserAddAccountMember)           // Add user to account
	g.POST(":aid/users/:uid/update", mgr.UserUpdateAccountMember) // Update user in account
	g.DELETE(":aid/users/:uid", mgr.UserRemoveAccountMember)      // Remove user from account
	g.GET(":aid/users/out", mgr.UserListUsersOutOfAccount)        // Get users out of account
	g.GET(":aid/users", mgr.UserListAccountMembers)               // Get users in account
	g.PUT(":aid/users/:uid", mgr.UserUpdateAccountMemberPartial)  // Batch update user-account relationship
}

func (mgr *AccountMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.AdminListAccounts)
	g.POST("", mgr.CreateAccount)
	g.GET(":aid", mgr.GetAccountByID)
	g.GET(":aid/quota", mgr.GetQuota)
	g.GET(":aid/billing/config", mgr.GetAccountBillingConfig)
	g.PUT(":aid/billing/config", mgr.UpdateAccountBillingConfig)
	g.GET(":aid/billing/members", mgr.AdminListAccountBillingMembers)
	g.PUT(":aid/billing/members/:uid", mgr.AdminUpdateAccountBillingMemberIssueAmount)
	g.POST(":aid/billing/reset", mgr.AdminResetAccountBillingBalance)
	g.PUT(":aid", mgr.UpdateAccount)
	g.DELETE(":aid", mgr.DeleteAccount)
	g.POST("add/:aid/:uid", mgr.AdminAddAccountMember)
	g.POST("update/:aid/:uid", mgr.AdminUpdateAccountMember)
	g.GET("userIn/:aid", mgr.AdminListAccountMembers)
	g.GET("userOutOf/:aid", mgr.AdminListUsersOutOfAccount)
	g.DELETE(":aid/:uid", mgr.AdminRemoveAccountMember)
	g.PUT("userIn/:aid", mgr.AdminUpdateAccountMemberPartial)
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

// UserListAccounts lists accounts accessible by current user
//
//	@Summary		Get all accounts for user
//	@Description	Join user_account and account tables to get summary info of all user's accounts
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[[]AccountResp]	"成功返回值描述"
//	@Failure		400	{object}	resputil.Response[any]				"请求参数错误"
//	@Failure		500	{object}	resputil.Response[any]				"其他错误"
//	@Router			/v1/accounts [get]
func (mgr *AccountMgr) UserListAccounts(c *gin.Context) {
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

	AccountBillingConfigResp struct {
		IssueAmount                 float64    `json:"issueAmount"`
		IssuePeriodMinutes          int        `json:"issuePeriodMinutes"`
		EffectiveIssueAmount        float64    `json:"effectiveIssueAmount"`
		EffectiveIssuePeriodMinutes int        `json:"effectiveIssuePeriodMinutes"`
		LastIssuedAt                *time.Time `json:"lastIssuedAt"`
	}

	AccountBillingConfigUpdateReq struct {
		IssueAmount        *service.BillingAmountInput `json:"issueAmount"`
		IssuePeriodMinutes *int                        `json:"issuePeriodMinutes" binding:"omitempty,gte=0"`
	}
)

func buildListAllResp(account *model.Account) ListAllResp {
	if account == nil {
		return ListAllResp{}
	}
	return ListAllResp{
		ID:        account.ID,
		Name:      account.Name,
		Nickname:  account.Nickname,
		Space:     account.Space,
		Quota:     account.Quota.Data(),
		ExpiredAt: account.ExpiredAt,
	}
}

func (mgr *AccountMgr) respondAccountLookup(
	c *gin.Context,
	lookup func() (*model.Account, error),
	notFoundMessage string,
) {
	account, err := lookup()
	if err != nil {
		resputil.Error(c, notFoundMessage, resputil.NotSpecified)
		return
	}

	resputil.Success(c, buildListAllResp(account))
}

func (mgr *AccountMgr) isBillingFeatureEnabled(ctx context.Context) bool {
	return mgr.billingService != nil && mgr.billingService.IsFeatureEnabled(ctx)
}

func (mgr *AccountMgr) isBillingUserFacingEnabled(ctx context.Context) bool {
	return mgr.billingService != nil && mgr.billingService.IsUserFacingEnabled(ctx)
}

func (mgr *AccountMgr) getAccountIssueOverrideFlags(ctx context.Context) (amountEnabled, periodEnabled bool) {
	if !mgr.isBillingFeatureEnabled(ctx) {
		return false, false
	}
	return mgr.billingService.IsAccountIssueAmountOverrideEnabled(ctx),
		mgr.billingService.IsAccountIssuePeriodOverrideEnabled(ctx)
}

func (mgr *AccountMgr) requireBillingEnabled(ctx context.Context) error {
	if mgr.billingService == nil || !mgr.isBillingFeatureEnabled(ctx) {
		return errBillingFeatureDisabled
	}
	return nil
}

func (mgr *AccountMgr) requireUserFacingBillingEnabled(ctx context.Context) error {
	if mgr.billingService == nil || !mgr.isBillingUserFacingEnabled(ctx) {
		return errBillingFeatureDisabled
	}
	return nil
}

func respondIfBillingFeatureDisabled(c *gin.Context, err error) bool {
	if !errors.Is(err, errBillingFeatureDisabled) {
		return false
	}
	resputil.Error(c, err.Error(), resputil.BusinessLogicError)
	return true
}

func handleAccountBillingConfigUpdateErr(c *gin.Context, err error) {
	if respondIfBillingFeatureDisabled(c, err) {
		return
	}
	if errors.Is(err, errNoBillingConfigFieldsToUpdate) {
		resputil.BadRequestError(c, err.Error())
		return
	}
	resputil.Error(c, err.Error(), resputil.NotSpecified)
}

func (mgr *AccountMgr) writeAccountBillingConfigUpdateResp(
	c *gin.Context,
	accountID uint,
	req *AccountBillingConfigUpdateReq,
) {
	resp, err := mgr.updateAccountBillingConfigCore(c, accountID, req)
	if err != nil {
		handleAccountBillingConfigUpdateErr(c, err)
		return
	}
	resputil.Success(c, resp)
}

func (mgr *AccountMgr) writeAccountBillingMemberIssueAmountResp(
	c *gin.Context,
	accountID, userID uint,
	req *UpdateAccountBillingMemberIssueAmountReq,
) {
	resp, err := mgr.updateAccountBillingMemberIssueAmount(c, accountID, userID, req)
	if err != nil {
		if respondIfBillingFeatureDisabled(c, err) {
			return
		}
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, resp)
}

// AdminListAccounts lists all accounts (admin API)
//
//	@Summary		Get all accounts
//	@Description	Get summary info of all accounts, supports filtering, pagination and sorting
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			page	query		ListAllReq				true	"分页参数"
//	@Success		200		{object}	resputil.Response[any]	"账户列表"
//	@Failure		400		{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]	"其他错误"
//	@Router			/v1/admin/projects [get]
func (mgr *AccountMgr) AdminListAccounts(c *gin.Context) {
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

func (mgr *AccountMgr) bindAccountID(c *gin.Context) (uint, bool) {
	var uriReq AccountIDReq
	if err := c.ShouldBindUri(&uriReq); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid account ID parameter: %v", err))
		return 0, false
	}
	return uriReq.ID, true
}

func (mgr *AccountMgr) requireUserManagedAccountID(c *gin.Context) (uint, bool) {
	accountID, ok := mgr.bindAccountID(c)
	if !ok {
		return 0, false
	}

	token := util.GetToken(c)
	if err := mgr.checkAccountAdmin(c, token.UserID, accountID); err != nil {
		resputil.HTTPError(c, httpStatusForbidden, "Forbidden: User is not account admin", resputil.NotSpecified)
		return 0, false
	}

	return accountID, true
}

func bindAccountName(c *gin.Context) (string, bool) {
	var uriReq AccountNameReq
	if err := c.ShouldBindUri(&uriReq); err != nil {
		resputil.Error(c, fmt.Sprintf("invalid request, detail: %v", err), resputil.NotSpecified)
		return "", false
	}
	return uriReq.Name, true
}

func (mgr *AccountMgr) buildAccountBillingConfigResp(
	ctx context.Context,
	account *model.Account,
) AccountBillingConfigResp {
	resp := AccountBillingConfigResp{}
	if account == nil {
		return resp
	}
	resp.LastIssuedAt = account.BillingLastIssuedAt
	if !mgr.isBillingFeatureEnabled(ctx) {
		return resp
	}
	if account.BillingIssueAmount != nil {
		resp.IssueAmount = service.ToDisplayPoints(*account.BillingIssueAmount)
	}
	if account.BillingIssuePeriodMinutes != nil {
		resp.IssuePeriodMinutes = *account.BillingIssuePeriodMinutes
	}
	effectiveIssueAmount, effectiveIssuePeriodMinutes := mgr.billingService.ResolveEffectiveIssueConfigForAccount(ctx, account)
	resp.EffectiveIssueAmount = service.ToDisplayPoints(effectiveIssueAmount)
	resp.EffectiveIssuePeriodMinutes = effectiveIssuePeriodMinutes
	return resp
}

func (mgr *AccountMgr) loadAccountByID(c *gin.Context, accountID uint) (*model.Account, error) {
	return query.Account.WithContext(c).Where(query.Account.ID.Eq(accountID)).First()
}

func (mgr *AccountMgr) getAccountBillingConfigResp(
	c *gin.Context,
	accountID uint,
) (AccountBillingConfigResp, error) {
	account, err := mgr.validateAccount(c, accountID)
	if err != nil {
		return AccountBillingConfigResp{}, err
	}

	return mgr.buildAccountBillingConfigResp(c.Request.Context(), account), nil
}

func (mgr *AccountMgr) updateAccountBillingConfigCore(
	c *gin.Context,
	accountID uint,
	req *AccountBillingConfigUpdateReq,
) (AccountBillingConfigResp, error) {
	if err := mgr.requireBillingEnabled(c.Request.Context()); err != nil {
		return AccountBillingConfigResp{}, err
	}

	if _, err := mgr.validateAccount(c, accountID); err != nil {
		return AccountBillingConfigResp{}, err
	}

	updates := map[string]any{}
	amountOverrideEnabled, periodOverrideEnabled := mgr.getAccountIssueOverrideFlags(c.Request.Context())
	if amountOverrideEnabled && req != nil && req.IssueAmount != nil {
		updates["billing_issue_amount"] = req.IssueAmount.MicroPoints()
	}
	if periodOverrideEnabled && req != nil && req.IssuePeriodMinutes != nil {
		updates["billing_issue_period_minutes"] = *req.IssuePeriodMinutes
	}
	if len(updates) == 0 {
		return AccountBillingConfigResp{}, errNoBillingConfigFieldsToUpdate
	}

	if _, err := query.Account.WithContext(c).Where(query.Account.ID.Eq(accountID)).Updates(updates); err != nil {
		return AccountBillingConfigResp{}, fmt.Errorf("failed to update account billing config: %w", err)
	}

	account, err := mgr.loadAccountByID(c, accountID)
	if err != nil {
		return AccountBillingConfigResp{}, fmt.Errorf("failed to reload account billing config: %w", err)
	}

	return mgr.buildAccountBillingConfigResp(c.Request.Context(), account), nil
}

func (mgr *AccountMgr) resetAccountBillingBalanceCore(c *gin.Context, accountID uint) (string, error) {
	if err := mgr.requireBillingEnabled(c.Request.Context()); err != nil {
		return "", err
	}

	account, err := mgr.validateAccount(c, accountID)
	if err != nil {
		return "", err
	}

	if err := mgr.billingService.IssueAccountNow(c.Request.Context(), accountID); err != nil {
		return "", fmt.Errorf("failed to reset billing balance for %s: %w", account.Name, err)
	}

	return fmt.Sprintf("billing balance reset for %s", account.Name), nil
}

func (mgr *AccountMgr) GetAccountBillingConfig(c *gin.Context) {
	accountID, ok := mgr.bindAccountID(c)
	if !ok {
		return
	}

	resp, err := mgr.getAccountBillingConfigResp(c, accountID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

func (mgr *AccountMgr) UpdateAccountBillingConfig(c *gin.Context) {
	accountID, ok := mgr.bindAccountID(c)
	if !ok {
		return
	}

	var req AccountBillingConfigUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	mgr.writeAccountBillingConfigUpdateResp(c, accountID, &req)
}

func (mgr *AccountMgr) UserGetAccountBillingConfig(c *gin.Context) {
	if err := mgr.requireUserFacingBillingEnabled(c.Request.Context()); err != nil {
		handleAccountBillingConfigUpdateErr(c, err)
		return
	}
	accountID, ok := mgr.requireUserManagedAccountID(c)
	if !ok {
		return
	}

	resp, err := mgr.getAccountBillingConfigResp(c, accountID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

func (mgr *AccountMgr) UserUpdateAccountBillingConfig(c *gin.Context) {
	var req AccountBillingConfigUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := mgr.requireUserFacingBillingEnabled(c.Request.Context()); err != nil {
		handleAccountBillingConfigUpdateErr(c, err)
		return
	}

	accountID, ok := mgr.requireUserManagedAccountID(c)
	if !ok {
		return
	}
	mgr.writeAccountBillingConfigUpdateResp(c, accountID, &req)
}

// GetAccountByID godoc
//
//	@Summary		获取指定账户
//	@Description	根据账户ID获取账户的信息
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			aid	path		AccountIDReq			true	"projectname"
//	@Success		200	{object}	resputil.Response[any]	"账户信息"
//	@Failure		400	{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500	{object}	resputil.Response[any]	"其他错误"
//	@Router			/v1/admin/accounts/{aid} [get]
func (mgr *AccountMgr) GetAccountByID(c *gin.Context) {
	accountID, ok := mgr.bindAccountID(c)
	if !ok {
		return
	}
	mgr.respondAccountByID(c, accountID)
}

type AccountNameReq struct {
	Name string `uri:"name" binding:"required"`
}

// GetAccountByName godoc
//
//	@Summary		获取指定账户
//	@Description	根据账户名称获取账户的信息
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		AccountNameReq			true	"projectname"
//	@Success		200		{object}	resputil.Response[any]	"账户信息"
//	@Failure		400		{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]	"其他错误"
//	@Router			/v1/accounts/by-name/{name} [get]
func (mgr *AccountMgr) GetAccountByName(c *gin.Context) {
	accountName, ok := bindAccountName(c)
	if !ok {
		return
	}
	mgr.respondAccountByName(c, accountName)
}

func (mgr *AccountMgr) respondAccountByID(c *gin.Context, accountID uint) {
	q := query.Account
	mgr.respondAccountLookup(
		c,
		func() (*model.Account, error) {
			return q.WithContext(c).Where(q.ID.Eq(accountID)).First()
		},
		fmt.Sprintf("account not found: account ID %d does not exist", accountID),
	)
}

func (mgr *AccountMgr) respondAccountByName(c *gin.Context, accountName string) {
	q := query.Account
	mgr.respondAccountLookup(
		c,
		func() (*model.Account, error) {
			return q.WithContext(c).Where(q.Name.Eq(accountName)).First()
		},
		fmt.Sprintf("account not found: account name '%s' does not exist", accountName),
	)
}

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

	queueName := vcqueue.ResolveAccountQueueName(account.ID, account.Name)
	queue := scheduling.Queue{}

	if err = mgr.client.Get(c, types.NamespacedName{
		Name:      queueName,
		Namespace: config.GetConfig().Namespaces.Job,
	}, &queue); err != nil {
		resputil.Error(c, "Queue not found", resputil.NotSpecified)
		return
	}

	allocated := queue.Status.Allocated
	guarantee := queue.Spec.Guarantee.Resource
	deserved := queue.Spec.Deserved
	capability := queue.Spec.Capability

	resources := newAllocatedQuotaResources(allocated)
	applyQuotaResourceList(resources, guarantee, setQuotaGuarantee)
	applyQuotaResourceList(resources, deserved, setQuotaDeserved)
	applyQuotaResourceList(resources, capability, setQuotaCapability)

	// if capability is not set, read max from db
	r := query.Resource
	for name, resourceResp := range resources {
		if resourceResp.Capability != nil {
			continue
		}
		resourceModel, err := r.WithContext(c).Where(r.ResourceName.Eq(string(name))).First()
		if err != nil {
			continue
		}
		resourceResp.Capability = &payload.ResourceBase{
			Amount: resourceModel.Amount,
			Format: resourceModel.Format,
		}
		resources[name] = resourceResp
	}

	resputil.Success(c, buildQuotaResp(resources))
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
		Admins         []uint    `json:"admins"`
		ExpiredAt      time.Time `json:"ExpiredAt"`
	}

	ProjectCreateResp struct {
		ID uint `json:"id"`
	}
)

// CreateAccount godoc
//
//	@Summary		创建团队账户
//	@Description	从请求中获取账户名称、描述和配额，以当前用户为管理员，创建一个团队账户
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body		any										true	"账户信息"
//	@Success		200		{object}	resputil.Response[ProjectCreateResp]	"成功创建账户，返回账户ID"
//	@Failure		400		{object}	resputil.Response[any]					"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]					"账户创建失败，返回错误信息"
//	@Router			/v1/projects [post]
func (mgr *AccountMgr) CreateAccount(c *gin.Context) {
	token := util.GetToken(c)

	var req AccountCreateOrUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if len(req.Admins) == 0 {
		resputil.Error(c, "Admins is empty", resputil.InvalidRequest)
		return
	}

	queueID, createdQueueNames, createdAdminIDs, err := mgr.createAccountCore(c, token, &req)

	if err != nil {
		mgr.cleanupCreatedQueues(c, createdQueueNames)
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	mgr.issueUserAccountNowForAdmins(c, queueID, createdAdminIDs)
	resputil.Success(c, ProjectCreateResp{ID: queueID})
}

// UpdateAccount godoc
//
//	@Summary		更新配额
//	@Description	更新配额
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			aid		path		AccountIDReq				true	"projectname"
//	@Param			data	body		any							true	"更新quota"
//	@Success		200		{object}	resputil.Response[string]	"成功更新配额"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/admin/projects/{aid} [put]
func (mgr *AccountMgr) UpdateAccount(c *gin.Context) {
	var req AccountCreateOrUpdateReq
	var uriReq AccountIDReq
	var queueName string
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := c.ShouldBindUri(&uriReq); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid account ID parameter: %v", err))
		return
	}
	token := util.GetToken(c)
	err := mgr.updateAccountCore(c, token, uriReq.ID, &req, &queueName)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, fmt.Sprintf("update capability of %s", queueName))
}

// AdminResetAccountBillingBalance godoc
//
//	@Summary		立即重置账户免费额度
//	@Description	立即对指定账户执行一次免费额度发放，不影响 extra 点数
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			aid	path		AccountIDReq				true	"account id"
//	@Success		200	{object}	resputil.Response[string]	"重置成功"
//	@Failure		400	{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500	{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/admin/projects/{aid}/billing/reset [post]
func (mgr *AccountMgr) AdminResetAccountBillingBalance(c *gin.Context) {
	accountID, ok := mgr.bindAccountID(c)
	if !ok {
		return
	}

	resp, err := mgr.resetAccountBillingBalanceCore(c, accountID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

func (mgr *AccountMgr) UserResetAccountBillingBalance(c *gin.Context) {
	if err := mgr.requireUserFacingBillingEnabled(c.Request.Context()); err != nil {
		handleAccountBillingConfigUpdateErr(c, err)
		return
	}
	accountID, ok := mgr.requireUserManagedAccountID(c)
	if !ok {
		return
	}

	resp, err := mgr.resetAccountBillingBalanceCore(c, accountID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

func (mgr *AccountMgr) createAccountCore(
	c *gin.Context,
	token util.JWTMessage,
	req *AccountCreateOrUpdateReq,
) (queueID uint, createdQueueNames []string, createdAdminIDs []uint, err error) {
	db := query.Use(query.GetDB())
	createdQueueNames = make([]string, 0)
	createdAdminIDs = make([]uint, 0)
	err = db.Transaction(func(tx *query.Query) error {
		queue, quota, createErr := mgr.createAccountRecord(c, tx, req)
		if createErr != nil {
			return createErr
		}
		queueID = queue.ID
		adminIDs, queueNames, parentNames, queueQuotas, createErr := mgr.createAccountMembers(c, tx, req, queue, quota)
		if createErr != nil {
			return createErr
		}
		createdAdminIDs = adminIDs
		createdQueueNames, createErr = mgr.createVolcanoQueues(c, token, queueNames, parentNames, queueQuotas)
		return createErr
	})
	return queueID, createdQueueNames, createdAdminIDs, err
}

func (mgr *AccountMgr) createAccountRecord(
	c *gin.Context,
	tx *query.Query,
	req *AccountCreateOrUpdateReq,
) (*model.Account, *model.QueueQuota, error) {
	q := tx.Account
	queue := &model.Account{Nickname: req.Nickname}
	if err := q.WithContext(c).Create(queue); err != nil {
		return nil, nil, err
	}
	quota := &model.QueueQuota{
		Guaranteed: req.Quota.Guaranteed,
		Deserved:   req.Quota.Deserved,
		Capability: req.Quota.Capability,
	}
	queue.Name = fmt.Sprintf("q-%d", queue.ID)
	queue.Space = fmt.Sprintf("q-%d", queue.ID)
	queue.Quota = datatypes.NewJSONType(*quota)
	if !req.ExpiredAt.IsZero() {
		queue.ExpiredAt = &req.ExpiredAt
	}
	if _, err := q.WithContext(c).Where(q.ID.Eq(queue.ID)).Updates(queue); err != nil {
		return nil, nil, err
	}
	return queue, quota, nil
}

func (mgr *AccountMgr) createAccountMembers(
	c *gin.Context,
	tx *query.Query,
	req *AccountCreateOrUpdateReq,
	queue *model.Account,
	quota *model.QueueQuota,
) (adminIDs []uint, queueNames []string, parentNames []*string, queueQuotas []*model.QueueQuota, err error) {
	uq := tx.UserAccount
	adminIDs = make([]uint, 0, len(req.Admins))
	queueNames = make([]string, 0)
	parentNames = make([]*string, 0)
	queueQuotas = make([]*model.QueueQuota, 0)
	logicQueueName := vcqueue.GetAccountLogicQueueName(queue.ID)
	leafQueueName := vcqueue.GetAccountQueueName(queue.Name)
	queueNames = append(queueNames, logicQueueName, leafQueueName)
	parentNames = append(parentNames, nil, &logicQueueName)
	queueQuotas = append(queueQuotas, quota, nil)

	for _, adminID := range req.Admins {
		userQueue := model.UserAccount{
			UserID:     adminID,
			AccountID:  queue.ID,
			Role:       model.RoleAdmin,
			AccessMode: model.AccessModeRW,
		}
		if err := uq.WithContext(c).Create(&userQueue); err != nil {
			return nil, nil, nil, nil, err
		}
		adminIDs = append(adminIDs, adminID)
		if !req.WithoutVolcano {
			queueName := vcqueue.GetUserQueueName(queue.ID, adminID)
			queueNames = append(queueNames, queueName)
			parentNames = append(parentNames, &logicQueueName)
			queueQuotas = append(queueQuotas, nil)
		}
	}
	return adminIDs, queueNames, parentNames, queueQuotas, nil
}

func (mgr *AccountMgr) createVolcanoQueues(
	c *gin.Context,
	token util.JWTMessage,
	queueNames []string,
	parentQueueNames []*string,
	queueQuotas []*model.QueueQuota,
) ([]string, error) {
	created := make([]string, 0, len(queueNames))
	for i := 0; i < len(queueNames); i++ {
		queueName := queueNames[i]
		parentQueueName := parentQueueNames[i]
		quota := queueQuotas[i]
		if err := vcqueue.CreateQueue(c, mgr.client, token, queueName, parentQueueName, quota); err != nil {
			return created, err
		}
		created = append(created, queueName)
	}
	return created, nil
}

func (mgr *AccountMgr) cleanupCreatedQueues(c *gin.Context, queueNames []string) {
	for _, queueName := range queueNames {
		_ = vcqueue.DeleteQueue(c, mgr.client, queueName)
	}
}

func (mgr *AccountMgr) issueUserAccountNowForAdmins(c *gin.Context, accountID uint, adminIDs []uint) {
	if mgr.billingService == nil ||
		!shouldIssueInitialAccountBalance(
			mgr.isBillingFeatureEnabled(c.Request.Context()),
			mgr.billingService.IsActive(c.Request.Context()),
			len(adminIDs),
		) {
		return
	}
	if issueErr := mgr.billingService.IssueAccountNow(c.Request.Context(), accountID); issueErr != nil {
		klog.Warningf(
			"immediate billing issue after account create failed, accountID=%d, err=%v",
			accountID,
			issueErr,
		)
	}
}

func shouldIssueInitialAccountBalance(featureEnabled, active bool, adminCount int) bool {
	return featureEnabled && active && adminCount > 0
}

func (mgr *AccountMgr) updateAccountCore(
	c *gin.Context,
	token util.JWTMessage,
	accountID uint,
	req *AccountCreateOrUpdateReq,
	queueName *string,
) error {
	return query.Q.Transaction(func(tx *query.Query) error {
		queue, err := tx.Account.WithContext(c).Where(tx.Account.ID.Eq(accountID)).First()
		if err != nil {
			resputil.Error(c, fmt.Sprintf("account not found: account ID %d does not exist", accountID), resputil.NotSpecified)
			return err
		}
		*queueName = queue.Name
		if err := mgr.updateAccountRecord(c, tx, queue.ID, req); err != nil {
			return err
		}
		if req.WithoutVolcano {
			return nil
		}
		return mgr.updateAccountVolcanoQueue(c, token, queue.ID, queue.Name, req)
	})
}

func (mgr *AccountMgr) updateAccountRecord(
	c *gin.Context,
	tx *query.Query,
	accountID uint,
	req *AccountCreateOrUpdateReq,
) error {
	updates := map[string]any{
		"nickname": req.Nickname,
		"quota": datatypes.NewJSONType(model.QueueQuota{
			Guaranteed: req.Quota.Guaranteed,
			Deserved:   req.Quota.Deserved,
			Capability: req.Quota.Capability,
		}),
	}
	if !req.ExpiredAt.IsZero() {
		updates["expired_at"] = req.ExpiredAt
	}
	if _, err := tx.Account.WithContext(c).Where(tx.Account.ID.Eq(accountID)).Updates(updates); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to update account %d: %v", accountID, err), resputil.NotSpecified)
		return err
	}
	return nil
}

func (mgr *AccountMgr) updateAccountVolcanoQueue(
	c *gin.Context,
	token util.JWTMessage,
	accountID uint,
	accountName string,
	req *AccountCreateOrUpdateReq,
) error {
	quota := model.QueueQuota{
		Guaranteed: req.Quota.Guaranteed,
		Deserved:   req.Quota.Deserved,
		Capability: req.Quota.Capability,
	}
	if err := vcqueue.EnsureAccountQueueExists(c, mgr.client, token, accountID); err != nil {
		return err
	}
	queueName := vcqueue.ResolveAccountQueueName(accountID, accountName)
	if err := vcqueue.UpdateQueue(c, mgr.client, queueName, quota); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to update Volcano queue for account %d: %v", accountID, err), resputil.NotSpecified)
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
//
//	@Summary		删除账户
//	@Description	删除账户record和队列crd
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			aid	path		DeleteProjectReq						true	"aid"
//	@Success		200	{object}	resputil.Response[DeleteProjectResp]	"删除的队列名"
//	@Failure		400	{object}	resputil.Response[any]					"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]					"Other errors"
//	@Router			/v1/admin/projects/{aid} [delete]
func (mgr *AccountMgr) DeleteAccount(c *gin.Context) {
	var req DeleteProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid account ID parameter: %v", err))
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
		msg := fmt.Sprintf(
			"cannot delete account %d: account still has %d member(s)",
			queueID,
			len(userQueues),
		)
		resputil.Error(c, msg, resputil.InvalidRequest)
		return
	}

	var accountName string

	err := db.Transaction(func(tx *query.Query) error {
		q := tx.Account
		uq := tx.UserAccount

		// get queue in db
		queue, err := q.WithContext(c).Where(q.ID.Eq(queueID)).First()
		if err != nil {
			return err
		}
		accountName = queue.Name
		toDeleteQueues := make([]string, 0)

		// get user-queues relationship without quota limit
		userQueues, err := uq.WithContext(c).Where(uq.AccountID.Eq(queue.ID)).Find()
		if err != nil {
			return err
		}

		if len(userQueues) > 0 {
			for _, uq := range userQueues {
				queueName := vcqueue.GetUserQueueName(queue.ID, uq.UserID)
				toDeleteQueues = append(toDeleteQueues, queueName)
			}
			if _, err := uq.WithContext(c).Delete(userQueues...); err != nil {
				return err
			}
		}

		if _, err := q.WithContext(c).Delete(queue); err != nil {
			return err
		}
		toDeleteQueues = append(toDeleteQueues, vcqueue.GetAccountQueueName(queue.Name), vcqueue.GetAccountLogicQueueName(queue.ID))

		for _, queueName := range toDeleteQueues {
			if err := vcqueue.DeleteQueue(c, mgr.client, queueName); err != nil {
				klog.Errorf("failed to delete Volcano queue %s for account %d: %v", queueName, queue.ID, err)
			}
		}

		return nil
	})

	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	} else {
		resputil.Success(c, DeleteProjectResp{Name: accountName})
	}
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

// AdminAddAccountMember adds user to account (admin API)
//
//	@Summary		Add user to account
//	@Description	Create a user-account relationship
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			uid	path		uint					true	"uid"
//	@Param			aid	path		uint					true	"aid"
//	@Param			req	body		any						true	"Role and access mode"
//	@Success		200	{object}	resputil.Response[any]	"Returns added username and account name"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/projects/add/{aid}/{uid} [post]
//
//nolint:dupl // AdminAddAccountMember and AdminUpdateAccountMember have similar structure but different business logic
func (mgr *AccountMgr) AdminAddAccountMember(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	queue, err := mgr.validateAccount(c, req.QueueID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	user, err := mgr.validateUser(c, req.UserID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	var reqBody UpdateUserProjectReq
	if err = c.ShouldBindJSON(&reqBody); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	role, err := mgr.parseAndValidateRole(reqBody.Role)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	accessMode, err := mgr.parseAndValidateAccessMode(reqBody.AccessMode)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if err := mgr.createUserAccount(c, req.QueueID, req.UserID, role, accessMode, reqBody.Quota); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("Add User %s for %s", user.Name, queue.Nickname))
}

// AdminUpdateAccountMember updates user in account (admin API)
//
//	@Summary		Update user in account
//	@Description	Update a user-account relationship
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			uid	path		uint					true	"uid"
//	@Param			aid	path		uint					true	"aid"
//	@Param			req	body		any						true	"Role and access mode"
//	@Success		200	{object}	resputil.Response[any]	"Returns added username and account name"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/projects/update/{aid}/{uid} [post]
//
//nolint:dupl // AdminUpdateAccountMember and AdminAddAccountMember have similar structure but different business logic
func (mgr *AccountMgr) AdminUpdateAccountMember(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	queue, err := mgr.validateAccount(c, req.QueueID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	user, err := mgr.validateUser(c, req.UserID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	var reqBody UpdateUserProjectReq
	if err = c.ShouldBindJSON(&reqBody); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	role, err := mgr.parseAndValidateRole(reqBody.Role)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	accessMode, err := mgr.parseAndValidateAccessMode(reqBody.AccessMode)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if err := mgr.updateUserAccount(c, req.QueueID, req.UserID, role, accessMode, reqBody.Quota); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("Update User %s for %s", user.Name, queue.Nickname))
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

type AccountBillingMemberResp struct {
	IssueAmountOverride         *float64   `json:"issueAmountOverride,omitempty"`
	UserID                      uint       `json:"userId"`
	Username                    string     `json:"username"`
	Nickname                    string     `json:"nickname"`
	PeriodFreeBalance           float64    `json:"periodFreeBalance"`
	ExtraBalance                float64    `json:"extraBalance"`
	TotalAvailable              float64    `json:"totalAvailable"`
	LastIssuedAt                *time.Time `json:"lastIssuedAt"`
	NextIssueAt                 *time.Time `json:"nextIssueAt"`
	EffectiveIssueAmount        float64    `json:"effectiveIssueAmount"`
	EffectiveIssuePeriodMinutes int        `json:"effectiveIssuePeriodMinutes"`
}

type UpdateAccountBillingMemberIssueAmountReq struct {
	IssueAmountOverride *service.BillingAmountInput `json:"issueAmountOverride"`
}

// AdminListAccountMembers gets list of users in account (admin API)
//
//	@Summary		Get users in account
//	@Description	SQL query with join
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			aid	path		uint					true	"aid"
//	@Success		200	{object}	resputil.Response[any]	"User account entries"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/projects/userIn/{aid} [get]
//
//nolint:dupl // AdminListAccountMembers and AdminListUsersOutOfAccount have similar structure but different query logic
func (mgr *AccountMgr) AdminListAccountMembers(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	_, err := mgr.validateAccount(c, req.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resp, err := mgr.getUsersInAccount(c, req.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

func (mgr *AccountMgr) AdminListAccountBillingMembers(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	if _, err := mgr.validateAccount(c, req.ID); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resp, err := mgr.getBillingMembersInAccount(c, req.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

func (mgr *AccountMgr) AdminUpdateAccountBillingMemberIssueAmount(c *gin.Context) {
	var uriReq struct {
		AccountID uint `uri:"aid" binding:"required"`
		UserID    uint `uri:"uid" binding:"required"`
	}
	var req UpdateAccountBillingMemberIssueAmountReq

	if err := c.ShouldBindUri(&uriReq); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid uri params: %v", err))
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	mgr.writeAccountBillingMemberIssueAmountResp(c, uriReq.AccountID, uriReq.UserID, &req)
}

type PutUserInProjectUriReq struct {
	AccountId uint `uri:"aid" binding:"required"`
}

type PutUserInProjectReq struct {
	UserId     uint                                  `json:"uid" binding:"required"`
	Role       *string                               `json:"role"`
	AccessMode *string                               `json:"accessmode" gorm:"access_mode"`
	Quota      *datatypes.JSONType[model.QueueQuota] `json:"quota"`
}

type UserPutUserInProjectReq struct {
	Role       *string                               `json:"role"`
	AccessMode *string                               `json:"accessmode" gorm:"access_mode"`
	Quota      *datatypes.JSONType[model.QueueQuota] `json:"quota"`
}

type PutUserInProjectResp struct {
	AccountId uint `json:"aid" binding:"required"`
	UserId    uint `json:"uid" binding:"required"`
}

// AdminUpdateAccountMemberPartial batch updates user-account relationship (admin API)
//
//	@Summary		Batch update user-account relationship
//	@Description	Batch update user-account relationship (partial update)
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			aid	path		uint					true	"aid"
//	@Param			req	body		PutUserInProjectReq	true	"更新数据"
//	@Success		200	{object}	resputil.Response[PutUserInProjectResp]	"更新结果"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/projects/userIn/{aid} [put]
func (mgr *AccountMgr) AdminUpdateAccountMemberPartial(c *gin.Context) {
	uriReq := PutUserInProjectUriReq{}
	req := &PutUserInProjectReq{}
	if err := c.ShouldBindUri(&uriReq); err != nil {
		resputil.Error(c, fmt.Sprintf("validate PutUserInProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate PutUserInProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	_, err := mgr.validateAccount(c, uriReq.AccountId)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if err := mgr.putUserInAccount(c, uriReq.AccountId, req.UserId, req.Role, req.AccessMode, req.Quota); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	klog.Infof("user %d in account %d updated", req.UserId, uriReq.AccountId)
	ret := &PutUserInProjectResp{
		AccountId: uriReq.AccountId,
		UserId:    req.UserId,
	}
	resputil.Success(c, ret)
}

func checkResource(_ *gin.Context, ls v1.ResourceList) error {
	for k, v := range ls {
		if i, ok := v.AsInt64(); ok && i < 0 {
			return fmt.Errorf("resource %s invalid, is %d", k, i)
		}
	}
	return nil
}

// AdminListUsersOutOfAccount gets list of users not in account (admin API)
//
//	@Summary		Get users not in account
//	@Description	SQL query with subquery
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			aid	path		uint					true	"aid"
//	@Success		200	{object}	resputil.Response[any]	"User account entries"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/projects/userOutOf/{aid} [get]
//
//nolint:dupl // AdminListUsersOutOfAccount and AdminListAccountMembers have similar structure but different query logic
func (mgr *AccountMgr) AdminListUsersOutOfAccount(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	_, err := mgr.validateAccount(c, req.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resp, err := mgr.getUsersOutOfAccount(c, req.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

// AdminRemoveAccountMember removes user from account (admin API)
//
//	@Summary		Remove user from account
//	@Description	Delete user-account relationship
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			uid	path		uint					true	"uid"
//	@Param			aid	path		uint					true	"aid"
//	@Success		200	{object}	resputil.Response[any]	"返回添加的用户名和队列名"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/projects/update/{aid}/{uid} [delete]
func (mgr *AccountMgr) AdminRemoveAccountMember(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	queue, err := mgr.validateAccount(c, req.QueueID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	user, err := mgr.validateUser(c, req.UserID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if err := mgr.deleteUserAccount(c, req.QueueID, req.UserID); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("delete User %s for %s", user.Name, queue.Nickname))
}

// ========== Common logic functions (internal, not exported) ==========

// validateAccount validates if account exists
func (mgr *AccountMgr) validateAccount(c *gin.Context, accountID uint) (*model.Account, error) {
	q := query.Account
	account, err := q.WithContext(c).Where(q.ID.Eq(accountID)).First()
	if err != nil {
		return nil, fmt.Errorf("account not found: account ID %d does not exist", accountID)
	}
	return account, nil
}

// validateUser validates if user exists
func (mgr *AccountMgr) validateUser(c *gin.Context, userID uint) (*model.User, error) {
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(userID)).First()
	if err != nil {
		return nil, fmt.Errorf("user not found: user ID %d does not exist", userID)
	}
	return user, nil
}

// validateRole validates if role value is valid
func (mgr *AccountMgr) validateRole(role uint64) error {
	if role < uint64(model.RoleGuest) || role > uint64(model.RoleAdmin) {
		return fmt.Errorf("invalid role value: %d, valid values are %d (guest), %d (user), %d (admin)",
			role, model.RoleGuest, model.RoleUser, model.RoleAdmin)
	}
	return nil
}

// validateAccessMode validates if access mode value is valid
// Note: Currently, the frontend only supports RO (read-only) and RW (read-write) modes.
// NA (not-allowed) and AO (append-only) modes are not exposed in the UI.
func (mgr *AccountMgr) validateAccessMode(accessMode uint64) error {
	if accessMode != uint64(model.AccessModeRO) && accessMode != uint64(model.AccessModeRW) {
		return fmt.Errorf("invalid access mode value: %d, valid values are %d (RO - read-only), %d (RW - read-write)",
			accessMode, model.AccessModeRO, model.AccessModeRW)
	}
	return nil
}

// parseAndValidateRole parses role string and validates it
func (mgr *AccountMgr) parseAndValidateRole(roleStr string) (model.Role, error) {
	role, err := strconv.ParseUint(roleStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid role parameter: %w", err)
	}
	if err := mgr.validateRole(role); err != nil {
		return 0, err
	}
	// Check for uint8 overflow
	if role > maxUint8Value {
		return 0, fmt.Errorf("role value %d exceeds uint8 range", role)
	}
	return model.Role(role), nil
}

// parseAndValidateAccessMode parses access mode string and validates it
func (mgr *AccountMgr) parseAndValidateAccessMode(accessStr string) (model.AccessMode, error) {
	access, err := strconv.ParseUint(accessStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid access mode parameter: %w", err)
	}
	if err := mgr.validateAccessMode(access); err != nil {
		return 0, err
	}
	// Check for uint8 overflow
	if access > maxUint8Value {
		return 0, fmt.Errorf("access mode value %d exceeds uint8 range", access)
	}
	return model.AccessMode(access), nil
}

// checkAccountAdmin checks if user is account admin
func (mgr *AccountMgr) checkAccountAdmin(c *gin.Context, userID, accountID uint) error {
	uq := query.UserAccount
	userAccount, err := uq.WithContext(c).Where(uq.UserID.Eq(userID), uq.AccountID.Eq(accountID)).First()
	if err != nil {
		return fmt.Errorf("user %d is not in account %d", userID, accountID)
	}
	if userAccount.Role != model.RoleAdmin {
		return fmt.Errorf("user %d is not an admin of account %d", userID, accountID)
	}
	return nil
}

// checkUserInAccount checks if user is in account (does not require admin role)
func (mgr *AccountMgr) checkUserInAccount(c *gin.Context, userID, accountID uint) error {
	uq := query.UserAccount
	_, err := uq.WithContext(c).Where(uq.UserID.Eq(userID), uq.AccountID.Eq(accountID)).First()
	if err != nil {
		return fmt.Errorf("user %d is not in account %d", userID, accountID)
	}
	return nil
}

// isDefaultAccount checks if account is the default account (by ID or name)
func (mgr *AccountMgr) isDefaultAccount(c *gin.Context, accountID uint) (bool, error) {
	// Check by ID first (most common case)
	if accountID == model.DefaultAccountID {
		return true, nil
	}
	// Check by name as fallback
	account, err := mgr.validateAccount(c, accountID)
	if err != nil {
		return false, err
	}
	return account.Name == "default", nil
}

// createUserAccount creates user-account relationship
func (mgr *AccountMgr) createUserAccount(
	c *gin.Context,
	accountID, userID uint,
	role model.Role,
	accessMode model.AccessMode,
	quota v1.ResourceList,
) error {
	// Prevent adding user to default account
	isDefault, err := mgr.isDefaultAccount(c, accountID)
	if err != nil {
		return fmt.Errorf("failed to check if account is default: %w", err)
	}
	if isDefault {
		return fmt.Errorf("cannot add user to default account (account ID: %d)", accountID)
	}

	account, err := mgr.validateAccount(c, accountID)
	if err != nil {
		return err
	}
	_, err = mgr.validateUser(c, userID)
	if err != nil {
		return err
	}

	err = query.Q.Transaction(func(tx *query.Query) error {
		uq := query.UserAccount

		// Check if user already exists in account
		_, err = tx.UserAccount.WithContext(c).Where(uq.AccountID.Eq(accountID), uq.UserID.Eq(userID)).First()
		if err == nil {
			return fmt.Errorf("user %d is already in account %d", userID, accountID)
		}

		q := model.QueueQuota{Capability: quota}
		if len(quota) == 0 && account.UserDefaultQuota != nil {
			q.Capability = account.UserDefaultQuota.Data().Capability
		}

		token := util.GetToken(c)
		queueName := vcqueue.GetUserQueueName(accountID, userID)
		parentQueueName := vcqueue.GetAccountLogicQueueName(accountID)
		if err := vcqueue.EnsureAccountQueueExists(c, mgr.client, token, accountID); err != nil {
			return fmt.Errorf("failed to ensure account queue exists: %w", err)
		}
		if err := vcqueue.CreateQueue(c, mgr.client, token, queueName, &parentQueueName, &q); err != nil {
			return fmt.Errorf("failed to create volcano queue for user %d in account %d: %w", userID, accountID, err)
		}

		userQueue := model.UserAccount{
			UserID:     userID,
			AccountID:  accountID,
			Role:       role,
			AccessMode: accessMode,
		}
		userQueue.Quota = datatypes.NewJSONType(q)

		if err := tx.UserAccount.WithContext(c).Create(&userQueue); err != nil {
			return fmt.Errorf("failed to create user-account relationship: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}
	if mgr.billingService != nil && mgr.isBillingFeatureEnabled(c.Request.Context()) {
		if issueErr := mgr.billingService.IssueUserAccountNow(c.Request.Context(), userID, accountID); issueErr != nil {
			klog.Warningf(
				"immediate billing issue after user-account create failed, accountID=%d, userID=%d, err=%v",
				accountID,
				userID,
				issueErr,
			)
		}
	}
	return nil
}

// updateUserAccount updates user-account relationship
func (mgr *AccountMgr) updateUserAccount(
	c *gin.Context,
	accountID, userID uint,
	role model.Role,
	accessMode model.AccessMode,
	quota v1.ResourceList,
) error {
	token := util.GetToken(c)
	err := query.Q.Transaction(func(tx *query.Query) error {
		userQueue, err := tx.UserAccount.WithContext(c).Where(tx.UserAccount.AccountID.Eq(accountID), tx.UserAccount.UserID.Eq(userID)).First()
		if err != nil {
			return fmt.Errorf("user %d is not in account %d", userID, accountID)
		}

		// Prevent role modification for default account
		isDefault, err := mgr.isDefaultAccount(c, accountID)
		if err != nil {
			return fmt.Errorf("failed to check if account is default: %w", err)
		}
		if isDefault && userQueue.Role != role {
			return fmt.Errorf("cannot modify user role in default account (account ID: %d)", accountID)
		}

		account, err := mgr.validateAccount(c, accountID)
		if err != nil {
			return err
		}
		_, err = mgr.validateUser(c, userID)
		if err != nil {
			return err
		}

		// 应用默认配额逻辑：用户配额 > 账户默认配额 > 无限制
		finalQuota := v1.ResourceList{}
		if !isDefault {
			finalQuota = quota
			if len(finalQuota) == 0 && account.UserDefaultQuota != nil {
				finalQuota = account.UserDefaultQuota.Data().Capability
			}
		}
		userQueue.Role = role
		userQueue.AccessMode = accessMode
		userQueue.Quota = datatypes.NewJSONType(model.QueueQuota{
			Capability: finalQuota,
		})

		_, err = tx.
			UserAccount.
			WithContext(c).
			Where(tx.UserAccount.AccountID.Eq(accountID), tx.UserAccount.UserID.Eq(userID)).
			Updates(userQueue)
		if err != nil {
			return fmt.Errorf("failed to update user-account relationship: %w", err)
		}

		if isDefault {
			return nil
		}

		if err := vcqueue.EnsureAccountQueueExists(c, mgr.client, token, accountID); err != nil {
			return fmt.Errorf("failed to ensure account queue exists: %w", err)
		}
		if err := vcqueue.EnsureUserQueueExists(c, mgr.client, token, accountID, userID); err != nil {
			return fmt.Errorf("failed to ensure user queue exists: %w", err)
		}
		queueName := vcqueue.GetUserQueueName(accountID, userID)
		if err := vcqueue.UpdateQueue(c, mgr.client, queueName, model.QueueQuota{
			Capability: finalQuota,
		}); err != nil {
			return fmt.Errorf("failed to update volcano queue for user %d in account %d: %w", userID, accountID, err)
		}

		return nil
	})

	return err
}

// deleteUserAccount deletes user-account relationship
func (mgr *AccountMgr) deleteUserAccount(c *gin.Context, accountID, userID uint) error {
	// Prevent deletion from default account
	isDefault, err := mgr.isDefaultAccount(c, accountID)
	if err != nil {
		return fmt.Errorf("failed to check if account is default: %w", err)
	}
	if isDefault {
		return fmt.Errorf("cannot remove user from default account (account ID: %d)", accountID)
	}

	account, err := mgr.validateAccount(c, accountID)
	if err != nil {
		return err
	}
	user, err := mgr.validateUser(c, userID)
	if err != nil {
		return err
	}

	err = query.Q.Transaction(func(tx *query.Query) error {
		userQueue, err := tx.UserAccount.WithContext(c).Where(tx.UserAccount.AccountID.Eq(accountID), tx.UserAccount.UserID.Eq(userID)).First()
		if err != nil {
			return fmt.Errorf("user %d is not in account %d", userID, accountID)
		}

		if _, err := tx.UserAccount.WithContext(c).Delete(userQueue); err != nil {
			return fmt.Errorf("failed to delete user-account relationship: %w", err)
		}

		userQueueName := vcqueue.GetUserQueueName(account.ID, user.ID)
		if err := vcqueue.DeleteQueue(c, mgr.client, userQueueName); err != nil {
			return fmt.Errorf("failed to delete volcano queue for user %d in account %d: %w", userID, accountID, err)
		}

		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

// getUsersInAccount gets list of users in account
func (mgr *AccountMgr) getUsersInAccount(c *gin.Context, accountID uint) ([]UserProjectGetResp, error) {
	u := query.User
	uq := query.UserAccount

	var resp []UserProjectGetResp
	exec := u.WithContext(c).Join(uq, uq.UserID.EqCol(u.ID)).Where(uq.DeletedAt.IsNull())
	exec = exec.Select(u.ID, u.Name, uq.Role, uq.AccessMode, u.Attributes, uq.Quota)
	if err := exec.Where(uq.AccountID.Eq(accountID)).Distinct().Scan(&resp); err != nil {
		return nil, fmt.Errorf("get userProject failed, detail: %w", err)
	}
	return resp, nil
}

func (mgr *AccountMgr) getBillingMembersInAccount(c *gin.Context, accountID uint) ([]AccountBillingMemberResp, error) {
	if mgr.billingService == nil || !mgr.isBillingFeatureEnabled(c.Request.Context()) {
		return []AccountBillingMemberResp{}, nil
	}

	account, err := mgr.validateAccount(c, accountID)
	if err != nil {
		return nil, err
	}

	ua := query.UserAccount
	u := query.User

	userAccounts, err := ua.WithContext(c).
		Where(ua.AccountID.Eq(accountID), ua.DeletedAt.IsNull()).
		Order(ua.UserID).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query account billing members: %w", err)
	}

	userIDs := make([]uint, 0, len(userAccounts))
	for i := range userAccounts {
		userIDs = append(userIDs, userAccounts[i].UserID)
	}

	usersByID := make(map[uint]*model.User, len(userIDs))
	if len(userIDs) > 0 {
		users, err := u.WithContext(c).
			Where(u.ID.In(userIDs...), u.DeletedAt.IsNull()).
			Find()
		if err != nil {
			return nil, fmt.Errorf("failed to load billing member users: %w", err)
		}
		for i := range users {
			usersByID[users[i].ID] = users[i]
		}
	}

	resp := make([]AccountBillingMemberResp, 0, len(userAccounts))
	for i := range userAccounts {
		userAccount := userAccounts[i]
		user := usersByID[userAccount.UserID]
		if user == nil {
			continue
		}
		resp = append(resp, mgr.buildAccountBillingMemberResp(c, account, userAccount, user))
	}
	return resp, nil
}

func (mgr *AccountMgr) buildAccountBillingMemberResp(
	c *gin.Context,
	account *model.Account,
	userAccount *model.UserAccount,
	user *model.User,
) AccountBillingMemberResp {
	issueAmount, issuePeriod := mgr.billingService.ResolveEffectiveIssueConfigForUserAccount(
		c.Request.Context(),
		userAccount,
		account,
	)
	nextIssueAt := mgr.billingService.ComputeNextIssueAt(account.BillingLastIssuedAt, issuePeriod, time.Now())

	var issueAmountOverride *float64
	if userAccount.BillingIssueAmountOverride != nil {
		override := service.ToDisplayPoints(*userAccount.BillingIssueAmountOverride)
		issueAmountOverride = &override
	}

	return AccountBillingMemberResp{
		IssueAmountOverride:         issueAmountOverride,
		UserID:                      user.ID,
		Username:                    user.Name,
		Nickname:                    user.Attributes.Data().Nickname,
		PeriodFreeBalance:           service.ToDisplayPoints(userAccount.PeriodFreeBalance),
		ExtraBalance:                service.ToDisplayPoints(user.ExtraBalance),
		TotalAvailable:              service.ToDisplayPoints(userAccount.PeriodFreeBalance + user.ExtraBalance),
		LastIssuedAt:                account.BillingLastIssuedAt,
		NextIssueAt:                 nextIssueAt,
		EffectiveIssueAmount:        service.ToDisplayPoints(issueAmount),
		EffectiveIssuePeriodMinutes: issuePeriod,
	}
}

func (mgr *AccountMgr) updateAccountBillingMemberIssueAmount(
	c *gin.Context,
	accountID, userID uint,
	req *UpdateAccountBillingMemberIssueAmountReq,
) (*AccountBillingMemberResp, error) {
	if err := mgr.requireBillingEnabled(c.Request.Context()); err != nil {
		return nil, err
	}

	account, err := mgr.validateAccount(c, accountID)
	if err != nil {
		return nil, err
	}

	u := query.User
	user, err := u.WithContext(c).
		Where(u.ID.Eq(userID), u.DeletedAt.IsNull()).
		First()
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	ua := query.UserAccount
	userAccount, err := ua.WithContext(c).
		Where(ua.AccountID.Eq(accountID), ua.UserID.Eq(userID), ua.DeletedAt.IsNull()).
		First()
	if err != nil {
		return nil, fmt.Errorf("user %d is not in account %d", userID, accountID)
	}

	if req != nil && req.IssueAmountOverride != nil && req.IssueAmountOverride.MicroPoints() < 0 {
		return nil, fmt.Errorf("issue amount override must be non-negative")
	}

	updateValue := any(nil)
	if req != nil && req.IssueAmountOverride != nil {
		updateValue = req.IssueAmountOverride.MicroPoints()
	}

	if _, err := ua.WithContext(c).
		Where(ua.AccountID.Eq(accountID), ua.UserID.Eq(userID), ua.DeletedAt.IsNull()).
		Updates(map[string]any{
			"billing_issue_amount_override": updateValue,
		}); err != nil {
		return nil, fmt.Errorf("failed to update billing issue amount override: %w", err)
	}

	userAccount.BillingIssueAmountOverride = nil
	if req != nil && req.IssueAmountOverride != nil {
		override := req.IssueAmountOverride.MicroPoints()
		userAccount.BillingIssueAmountOverride = &override
	}

	resp := mgr.buildAccountBillingMemberResp(c, account, userAccount, user)
	return &resp, nil
}

// getUsersOutOfAccount gets list of users not in account
func (mgr *AccountMgr) getUsersOutOfAccount(c *gin.Context, accountID uint) ([]UserProjectGetResp, error) {
	u := query.User
	uq := query.UserAccount
	var uids []uint

	if err := uq.WithContext(c).Select(uq.UserID).Where(uq.AccountID.Eq(accountID)).Scan(&uids); err != nil {
		return nil, fmt.Errorf("failed to scan user IDs: %w", err)
	}

	var resp []UserProjectGetResp
	exec := u.WithContext(c).Where(u.ID.NotIn(uids...)).Distinct()
	if err := exec.Scan(&resp); err != nil {
		return nil, fmt.Errorf("failed to get users out of account: %w", err)
	}
	return resp, nil
}

// putUserInAccount batch updates user-account relationship (partial update)
// validateRoleUpdateForDefaultAccount validates role update for default account
func (mgr *AccountMgr) validateRoleUpdateForDefaultAccount(
	c *gin.Context,
	accountID, userID uint,
	roleStr string,
) error {
	uq := query.UserAccount
	userQueue, err := uq.WithContext(c).Where(uq.AccountID.Eq(accountID), uq.UserID.Eq(userID)).First()
	if err != nil {
		return fmt.Errorf("user %d is not in account %d", userID, accountID)
	}

	role, err := mgr.parseAndValidateRole(roleStr)
	if err != nil {
		return err
	}

	// Prevent role modification for default account
	if userQueue.Role != role {
		return fmt.Errorf("cannot modify user role in default account (account ID: %d)", accountID)
	}
	return nil
}

// buildUserAccountUpdates builds update map for user-account relationship
func (mgr *AccountMgr) buildUserAccountUpdates(
	c *gin.Context,
	role *string,
	accessMode *string,
	quota *datatypes.JSONType[model.QueueQuota],
	account *model.Account,
	isDefault bool,
) (map[string]any, error) {
	updates := make(map[string]any)

	if role != nil {
		roleVal, err := mgr.parseAndValidateRole(*role)
		if err != nil {
			return nil, err
		}
		updates["role"] = roleVal
	}

	if accessMode != nil {
		accessVal, err := mgr.parseAndValidateAccessMode(*accessMode)
		if err != nil {
			return nil, err
		}
		updates["access_mode"] = accessVal
	}

	if quota != nil {
		finalQuota := quota.Data()
		if !isDefault && len(finalQuota.Capability) == 0 && account != nil && account.UserDefaultQuota != nil {
			finalQuota.Capability = account.UserDefaultQuota.Data().Capability
		}

		if err := checkResource(c, finalQuota.Guaranteed); err != nil {
			return nil, fmt.Errorf("invalid quota guaranteed resources: %w", err)
		}
		if err := checkResource(c, finalQuota.Deserved); err != nil {
			return nil, fmt.Errorf("invalid quota deserved resources: %w", err)
		}
		if err := checkResource(c, finalQuota.Capability); err != nil {
			return nil, fmt.Errorf("invalid quota capability resources: %w", err)
		}
		updates["quota"] = datatypes.NewJSONType(finalQuota)
	}

	return updates, nil
}

func (mgr *AccountMgr) putUserInAccount(
	c *gin.Context,
	accountID, userID uint,
	role *string,
	accessMode *string,
	quota *datatypes.JSONType[model.QueueQuota],
) error {
	token := util.GetToken(c)

	// Check if trying to modify role in default account
	isDefault, err := mgr.isDefaultAccount(c, accountID)
	if err != nil {
		return fmt.Errorf("failed to check if account is default: %w", err)
	}
	if isDefault && role != nil {
		if err := mgr.validateRoleUpdateForDefaultAccount(c, accountID, userID, *role); err != nil {
			return err
		}
	}

	return query.Q.Transaction(func(tx *query.Query) error {
		userQueue, err := mgr.getUserAccountForUpdate(c, tx, accountID, userID)
		if err != nil {
			return err
		}

		account, err := mgr.loadAccountForPartialUserUpdate(c, accountID, quota)
		if err != nil {
			return err
		}

		updates, err := mgr.buildUserAccountUpdates(c, role, accessMode, quota, account, isDefault)
		if err != nil {
			return err
		}
		if len(updates) == 0 {
			return nil
		}

		finalQuota, err := mgr.applyPartialUserAccountUpdates(c, tx, accountID, userID, userQueue, updates)
		if err != nil {
			return err
		}

		return mgr.syncPartialUserQueueQuota(c, token, accountID, userID, finalQuota, quota, isDefault)
	})
}

func (mgr *AccountMgr) getUserAccountForUpdate(c *gin.Context, tx *query.Query, accountID, userID uint) (*model.UserAccount, error) {
	uq := tx.UserAccount
	userQueue, err := uq.WithContext(c).
		Where(uq.AccountID.Eq(accountID), uq.UserID.Eq(userID)).
		First()
	if err != nil {
		return nil, fmt.Errorf("user %d is not in account %d", userID, accountID)
	}
	return userQueue, nil
}

func (mgr *AccountMgr) loadAccountForPartialUserUpdate(
	c *gin.Context,
	accountID uint,
	quota *datatypes.JSONType[model.QueueQuota],
) (*model.Account, error) {
	if quota == nil {
		return nil, nil
	}
	return mgr.validateAccount(c, accountID)
}

func (mgr *AccountMgr) applyPartialUserAccountUpdates(
	c *gin.Context,
	tx *query.Query,
	accountID, userID uint,
	userQueue *model.UserAccount,
	updates map[string]any,
) (model.QueueQuota, error) {
	uq := tx.UserAccount
	if _, err := uq.WithContext(c).
		Where(uq.AccountID.Eq(accountID), uq.UserID.Eq(userID)).
		Updates(updates); err != nil {
		return model.QueueQuota{}, fmt.Errorf("failed to update user-account relationship: %w", err)
	}

	finalQuota := userQueue.Quota.Data()
	if updatedQuota, ok := updates["quota"].(datatypes.JSONType[model.QueueQuota]); ok {
		finalQuota = updatedQuota.Data()
	}
	return finalQuota, nil
}

func (mgr *AccountMgr) syncPartialUserQueueQuota(
	c *gin.Context,
	token util.JWTMessage,
	accountID, userID uint,
	finalQuota model.QueueQuota,
	quota *datatypes.JSONType[model.QueueQuota],
	isDefault bool,
) error {
	if quota == nil || isDefault {
		return nil
	}

	if err := vcqueue.EnsureAccountQueueExists(c, mgr.client, token, accountID); err != nil {
		return fmt.Errorf("failed to ensure account queue exists: %w", err)
	}
	if err := vcqueue.EnsureUserQueueExists(c, mgr.client, token, accountID, userID); err != nil {
		return fmt.Errorf("failed to ensure user queue exists: %w", err)
	}

	queueName := vcqueue.GetUserQueueName(accountID, userID)
	if err := vcqueue.UpdateQueue(c, mgr.client, queueName, finalQuota); err != nil {
		return fmt.Errorf("failed to update volcano queue for user %d in account %d: %w", userID, accountID, err)
	}
	return nil
}

// ========== User API functions (require account admin permission) ==========

// UserAddAccountMember adds user to account (user API)
//
//	@Summary		Add user to account (account admin)
//	@Description	Account admin adds user to account
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			uid	path		uint					true	"uid"
//	@Param			aid	path		uint					true	"aid"
//	@Param			req	body		any						true	"Role and access mode"
//	@Success		200	{object}	resputil.Response[any]	"Returns added username and account name"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		403	{object}	resputil.Response[any]	"Forbidden - not account admin"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/accounts/{aid}/users/{uid} [post]
//
//nolint:dupl // UserAddAccountMember and UserUpdateAccountMember have similar structure but different business logic
func (mgr *AccountMgr) UserAddAccountMember(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)
	// Check if current user is account admin
	if err := mgr.checkAccountAdmin(c, token.UserID, req.QueueID); err != nil {
		resputil.HTTPError(c, httpStatusForbidden, "Forbidden: User is not account admin", resputil.NotSpecified)
		return
	}

	queue, err := mgr.validateAccount(c, req.QueueID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	user, err := mgr.validateUser(c, req.UserID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	var reqBody UpdateUserProjectReq
	if err = c.ShouldBindJSON(&reqBody); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	role, err := mgr.parseAndValidateRole(reqBody.Role)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	accessMode, err := mgr.parseAndValidateAccessMode(reqBody.AccessMode)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if err := mgr.createUserAccount(c, req.QueueID, req.UserID, role, accessMode, reqBody.Quota); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("Add User %s for %s", user.Name, queue.Nickname))
}

// UserUpdateAccountMember updates user in account (user API)
//
//	@Summary		Update user in account (account admin)
//	@Description	Account admin updates user information in account
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			uid	path		uint					true	"uid"
//	@Param			aid	path		uint					true	"aid"
//	@Param			req	body		any						true	"Role and access mode"
//	@Success		200	{object}	resputil.Response[any]	"Returns updated username and account name"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		403	{object}	resputil.Response[any]	"Forbidden - not account admin"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/accounts/{aid}/users/{uid}/update [post]
//
//nolint:dupl // UserUpdateAccountMember and UserAddAccountMember have similar structure but different business logic
func (mgr *AccountMgr) UserUpdateAccountMember(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)
	// Check if current user is account admin
	if err := mgr.checkAccountAdmin(c, token.UserID, req.QueueID); err != nil {
		resputil.HTTPError(c, httpStatusForbidden, "Forbidden: User is not account admin", resputil.NotSpecified)
		return
	}

	queue, err := mgr.validateAccount(c, req.QueueID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	user, err := mgr.validateUser(c, req.UserID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	var reqBody UpdateUserProjectReq
	if err = c.ShouldBindJSON(&reqBody); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	role, err := mgr.parseAndValidateRole(reqBody.Role)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	accessMode, err := mgr.parseAndValidateAccessMode(reqBody.AccessMode)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if err := mgr.updateUserAccount(c, req.QueueID, req.UserID, role, accessMode, reqBody.Quota); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("Update User %s for %s", user.Name, queue.Nickname))
}

// UserRemoveAccountMember removes user from account (user API)
//
//	@Summary		Remove user from account (account admin)
//	@Description	Account admin removes user from account
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			uid	path		uint					true	"uid"
//	@Param			aid	path		uint					true	"aid"
//	@Success		200	{object}	resputil.Response[any]	"Returns removed username and account name"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		403	{object}	resputil.Response[any]	"Forbidden - not account admin"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/accounts/{aid}/users/{uid} [delete]
func (mgr *AccountMgr) UserRemoveAccountMember(c *gin.Context) {
	var req UserProjectReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)
	// Check if current user is account admin
	if err := mgr.checkAccountAdmin(c, token.UserID, req.QueueID); err != nil {
		resputil.HTTPError(c, httpStatusForbidden, "Forbidden: User is not account admin", resputil.NotSpecified)
		return
	}

	queue, err := mgr.validateAccount(c, req.QueueID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	user, err := mgr.validateUser(c, req.UserID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if err := mgr.deleteUserAccount(c, req.QueueID, req.UserID); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("delete User %s for %s", user.Name, queue.Nickname))
}

// UserListAccountMembers gets list of users in account (user API)
//
//	@Summary		Get users in account
//	@Description	Get list of users in account (requires user to be in account)
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			aid	path		uint					true	"aid"
//	@Success		200	{object}	resputil.Response[any]	"User account entries"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		403	{object}	resputil.Response[any]	"Forbidden - user not in account"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/accounts/{aid}/users [get]
//
//nolint:dupl // UserListAccountMembers and UserListUsersOutOfAccount have similar structure but different query logic
func (mgr *AccountMgr) UserListAccountMembers(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)
	// Check if current user is in account (does not require admin role)
	if err := mgr.checkUserInAccount(c, token.UserID, req.ID); err != nil {
		resputil.HTTPError(c, httpStatusForbidden, "Forbidden: User is not in account", resputil.NotSpecified)
		return
	}

	_, err := mgr.validateAccount(c, req.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resp, err := mgr.getUsersInAccount(c, req.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

func (mgr *AccountMgr) UserListAccountBillingMembers(c *gin.Context) {
	if err := mgr.requireUserFacingBillingEnabled(c.Request.Context()); err != nil {
		handleAccountBillingConfigUpdateErr(c, err)
		return
	}
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)
	if err := mgr.checkAccountAdmin(c, token.UserID, req.ID); err != nil {
		resputil.HTTPError(c, httpStatusForbidden, "Forbidden: User is not account admin", resputil.NotSpecified)
		return
	}

	resp, err := mgr.getBillingMembersInAccount(c, req.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

// UserListUsersOutOfAccount gets list of users not in account (user API)
//
//	@Summary		Get users not in account
//	@Description	Get list of users not in account (requires user to be in account)
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			aid	path		uint					true	"aid"
//	@Success		200	{object}	resputil.Response[any]	"User account entries"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		403	{object}	resputil.Response[any]	"Forbidden - user not in account"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/accounts/{aid}/users/out [get]
//
//nolint:dupl // UserListUsersOutOfAccount and UserListAccountMembers have similar structure but different query logic
func (mgr *AccountMgr) UserListUsersOutOfAccount(c *gin.Context) {
	var req ProjectGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate UserProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)
	// Check if current user is in account (does not require admin role)
	if err := mgr.checkUserInAccount(c, token.UserID, req.ID); err != nil {
		resputil.HTTPError(c, httpStatusForbidden, "Forbidden: User is not in account", resputil.NotSpecified)
		return
	}

	_, err := mgr.validateAccount(c, req.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resp, err := mgr.getUsersOutOfAccount(c, req.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}

// UserUpdateAccountMemberPartial batch updates user-account relationship (user API)
//
//	@Summary		Batch update user-account relationship (account admin)
//	@Description	Account admin batch updates user-account relationship (partial update)
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			aid	path		uint					true	"aid"
//	@Param			uid	path		uint					true	"uid"
//	@Param			req	body		UserPutUserInProjectReq	true	"Update data"
//	@Success		200	{object}	resputil.Response[PutUserInProjectResp]	"Update result"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		403	{object}	resputil.Response[any]	"Forbidden - not account admin"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/accounts/{aid}/users/{uid} [put]
func (mgr *AccountMgr) UserUpdateAccountMemberPartial(c *gin.Context) {
	var uriReq struct {
		AccountId uint `uri:"aid" binding:"required"`
		UserID    uint `uri:"uid" binding:"required"`
	}
	req := &UserPutUserInProjectReq{}

	if err := c.ShouldBindUri(&uriReq); err != nil {
		resputil.Error(c, fmt.Sprintf("validate PutUserInProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate PutUserInProject parameters failed, detail: %v", err), resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)
	// Check if current user is account admin
	if err := mgr.checkAccountAdmin(c, token.UserID, uriReq.AccountId); err != nil {
		resputil.HTTPError(c, httpStatusForbidden, "Forbidden: User is not account admin", resputil.NotSpecified)
		return
	}

	_, err := mgr.validateAccount(c, uriReq.AccountId)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Use userID from URI, ignore UserId in request body
	if err := mgr.putUserInAccount(c, uriReq.AccountId, uriReq.UserID, req.Role, req.AccessMode, req.Quota); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	klog.Infof("user %d in account %d updated", uriReq.UserID, uriReq.AccountId)
	ret := &PutUserInProjectResp{
		AccountId: uriReq.AccountId,
		UserId:    uriReq.UserID,
	}
	resputil.Success(c, ret)
}

func (mgr *AccountMgr) UserUpdateAccountBillingMemberIssueAmount(c *gin.Context) {
	var uriReq struct {
		AccountID uint `uri:"aid" binding:"required"`
		UserID    uint `uri:"uid" binding:"required"`
	}
	var req UpdateAccountBillingMemberIssueAmountReq

	if err := mgr.requireUserFacingBillingEnabled(c.Request.Context()); err != nil {
		handleAccountBillingConfigUpdateErr(c, err)
		return
	}
	if err := c.ShouldBindUri(&uriReq); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid uri params: %v", err))
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	token := util.GetToken(c)
	if err := mgr.checkAccountAdmin(c, token.UserID, uriReq.AccountID); err != nil {
		resputil.HTTPError(c, httpStatusForbidden, "Forbidden: User is not account admin", resputil.NotSpecified)
		return
	}

	mgr.writeAccountBillingMemberIssueAmountResp(c, uriReq.AccountID, uriReq.UserID, &req)
}
