package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/rand"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/alert"
	"github.com/raids-lab/crater/pkg/utils"
)

// 邮箱验证码缓存
var verifyCodeCache = make(map[string]string)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewContextMgr)
}

type ContextMgr struct {
	name           string
	client         client.Client
	billingService *service.BillingService
}

func NewContextMgr(conf *RegisterConfig) Manager {
	return &ContextMgr{
		name:           "context",
		client:         conf.Client,
		billingService: conf.BillingService,
	}
}

func (mgr *ContextMgr) GetName() string { return mgr.name }

func (mgr *ContextMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ContextMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("quota", mgr.GetQuota)
	g.GET("billing/summary", mgr.GetBillingSummary)
	g.PUT("attributes", mgr.UpdateUserAttributes)
	g.POST("email/code", mgr.SendUserVerificationCode)
	g.POST("email/update", mgr.UpdateUserEmail)
}

func (mgr *ContextMgr) RegisterAdmin(_ *gin.RouterGroup) {}

type (
	SendCodeReq struct {
		Email string `json:"email"`
	}
	CheckCodeReq struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
)

// GetQuota godoc
//
//	@Summary		Get the queue information
//	@Description	query the queue information by client-go
//	@Tags			Context
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"Volcano Queue Quota"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"other errors"
//	@Router			/v1/context/queue [get]
func (mgr *ContextMgr) GetQuota(c *gin.Context) {
	token := util.GetToken(c)

	// Get jobs from database for current user and account
	j := query.Job
	jobs, err := j.WithContext(c).Where(
		j.UserID.Eq(token.UserID),
		j.AccountID.Eq(token.AccountID),
		j.Status.In(string(batch.Running), string(batch.Pending)),
	).Find()
	if err != nil {
		resputil.Error(c, "Failed to query jobs", resputil.NotSpecified)
		return
	}

	// Calculate allocated resources from running jobs
	allocated := v1.ResourceList{}
	for _, job := range jobs {
		for name, quantity := range job.Resources.Data() {
			if existing, exists := allocated[name]; exists {
				existing.Add(quantity)
				allocated[name] = existing
			} else {
				allocated[name] = quantity
			}
		}
	}

	resources := newAllocatedQuotaResources(allocated)

	// Get resource limits from database user
	ua := query.UserAccount
	userAccount, err := ua.WithContext(c).Where(ua.AccountID.Eq(token.AccountID), ua.UserID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, "Failed to query user account", resputil.NotSpecified)
		return
	}
	capability := userAccount.Quota.Data().Capability
	applyQuotaResourceList(resources, capability, setQuotaCapability)

	resputil.Success(c, buildQuotaResp(resources))
}

type (
	UserInfoResp struct {
		ID        uint                                    `json:"id"`
		Name      string                                  `json:"name"`
		Attribute datatypes.JSONType[model.UserAttribute] `json:"attributes"`
	}
)

type BillingSummaryResp struct {
	PeriodFreeBalance          float64    `json:"periodFreeBalance"`
	ExtraBalance               float64    `json:"extraBalance"`
	TotalAvailable             float64    `json:"totalAvailable"`
	LastIssuedAt               *time.Time `json:"lastIssuedAt"`
	NextIssueAt                *time.Time `json:"nextIssueAt"`
	EffectiveIssueAmount       float64    `json:"effectiveIssueAmount"`
	EffectiveIssuePeriodMinute int        `json:"effectiveIssuePeriodMinutes"`
}

func (mgr *ContextMgr) GetBillingSummary(c *gin.Context) {
	resp := BillingSummaryResp{}
	if mgr.billingService == nil || !mgr.billingService.IsUserFacingEnabled(c.Request.Context()) {
		resputil.Success(c, resp)
		return
	}

	token := util.GetToken(c)
	u := query.User
	a := query.Account
	uaQuery := query.UserAccount

	user, err := u.WithContext(c).
		Where(u.ID.Eq(token.UserID), u.DeletedAt.IsNull()).
		First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to load user: %v", err), resputil.NotSpecified)
		return
	}
	account, err := a.WithContext(c).
		Where(a.ID.Eq(token.AccountID), a.DeletedAt.IsNull()).
		First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to load account: %v", err), resputil.NotSpecified)
		return
	}
	ua, err := uaQuery.WithContext(c).
		Where(
			uaQuery.UserID.Eq(token.UserID),
			uaQuery.AccountID.Eq(token.AccountID),
			uaQuery.DeletedAt.IsNull(),
		).
		First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to load user-account relation: %v", err), resputil.NotSpecified)
		return
	}

	issueAmount, periodMinutes := mgr.billingService.ResolveEffectiveIssueConfigForUserAccount(
		c.Request.Context(),
		ua,
		account,
	)
	nextIssueAt := mgr.billingService.ComputeNextIssueAt(account.BillingLastIssuedAt, periodMinutes, time.Now())

	resp = BillingSummaryResp{
		PeriodFreeBalance:          service.ToDisplayPoints(ua.PeriodFreeBalance),
		ExtraBalance:               service.ToDisplayPoints(user.ExtraBalance),
		TotalAvailable:             service.ToDisplayPoints(ua.PeriodFreeBalance + user.ExtraBalance),
		LastIssuedAt:               account.BillingLastIssuedAt,
		NextIssueAt:                nextIssueAt,
		EffectiveIssueAmount:       service.ToDisplayPoints(issueAmount),
		EffectiveIssuePeriodMinute: periodMinutes,
	}
	resputil.Success(c, resp)
}

// UpdateUserAttributes godoc
//
//	@Summary		Update user attributes
//	@Description	Update the attributes of the current user
//	@Tags			Context
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			attributes	body		model.UserAttribute		true	"User attributes"
//	@Success		200			{object}	resputil.Response[any]	"User attributes updated"
//	@Failure		400			{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500			{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/context/attributes [put]
func (mgr *ContextMgr) UpdateUserAttributes(c *gin.Context) {
	token := util.GetToken(c)
	u := query.User

	var attributes model.UserAttribute
	if err := c.ShouldBindJSON(&attributes); err != nil {
		resputil.BadRequestError(c, "Invalid request body")
		return
	}

	user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, "User not found", resputil.NotSpecified)
		return
	}

	// Fix UID and GID are not allowed to be updated
	oldAttributes := user.Attributes.Data()
	attributes.ID = oldAttributes.ID
	attributes.UID = oldAttributes.UID
	attributes.GID = oldAttributes.GID

	user.Attributes = datatypes.NewJSONType(attributes)
	if err := u.WithContext(c).Save(user); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to update user attributes:  %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, "User attributes updated successfully")
}

// SendUserVerificationCode godoc
//
//	@Summary		Send User Verification Code for email
//	@Description	generate random code and save, send it to the user's email
//	@Tags			Context
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"Successfully send email verification code to user"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"other errors"
//	@Router			/v1/context/email/code [post]
func (mgr *ContextMgr) SendUserVerificationCode(c *gin.Context) {
	token := util.GetToken(c)
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, "User not found", resputil.NotSpecified)
		return
	}
	var req SendCodeReq
	if err = c.ShouldBind(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	receiver := user.Attributes.Data()
	receiver.Email = &req.Email
	verifyCode := fmt.Sprintf("%06d", getRandomCode())
	verifyCodeCache[req.Email] = verifyCode

	alertMgr := alert.GetAlertMgr()

	if err = alertMgr.SendVerificationCode(c, verifyCode, &receiver); err != nil {
		fmt.Println("Send Alarm Email failed:", err)
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "Successfully send email verification code to user")
}

func getRandomCode() int {
	RANGE := 1000000
	return rand.Intn(RANGE)
}

// UpdateUserEmail godoc
//
//	@Summary		Update after judging Verification Code for email
//	@Description	judge code and update email for user
//	@Tags			Context
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"User email updated successfully"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"other errors"
//	@Router			/v1/context/email/update [post]
func (mgr *ContextMgr) UpdateUserEmail(c *gin.Context) {
	token := util.GetToken(c)
	u := query.User
	_, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, "User not found", resputil.NotSpecified)
		return
	}

	var req CheckCodeReq
	if err := c.ShouldBind(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if req.Code != verifyCodeCache[req.Email] {
		resputil.Error(c, "Wrong verification Code", resputil.NotSpecified)
		return
	}

	// update user's LastEmailVerifiedAt
	curTime := utils.GetLocalTime()
	if _, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).Update(u.LastEmailVerifiedAt, curTime); err != nil {
		klog.Error("Failed to update LastEmailVerifiedAt", err)
	}

	resputil.Success(c, "User email updated successfully")
}
