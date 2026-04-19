package handler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/rand"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/alert"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
	"github.com/raids-lab/crater/pkg/vcqueue"
)

// 邮箱验证码缓存
var verifyCodeCache = make(map[string]string)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewContextMgr)
}

type ContextMgr struct {
	name          string
	client        client.Client
	configService *service.ConfigService
	queueQuotaSvc *service.PrequeueService
}

func NewContextMgr(conf *RegisterConfig) Manager {
	return &ContextMgr{
		name:          "context",
		client:        conf.Client,
		configService: conf.ConfigService,
		queueQuotaSvc: conf.PrequeueService,
	}
}

func (mgr *ContextMgr) GetName() string { return mgr.name }

func (mgr *ContextMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ContextMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("prequeue", mgr.GetPrequeueStatus)
	g.GET("quota", mgr.GetQuota)
	g.GET("job-resource-summary", mgr.GetJobResourceSummary)
	g.POST("resource-limit-check", mgr.CheckResourceLimit)
	g.PUT("attributes", mgr.UpdateUserAttributes)
	g.POST("email/code", mgr.SendUserVerificationCode)
	g.POST("email/update", mgr.UpdateUserEmail)
}

func (mgr *ContextMgr) RegisterAdmin(_ *gin.RouterGroup) {}

type (
	JobResourceSummaryScope string
	SendCodeReq             struct {
		Email string `json:"email"`
	}
	CheckCodeReq struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	ResourceLimitCheckReq struct {
		RequestedResources map[string]string `json:"requestedResources"`
	}
	PrequeueFeatureStatusResp struct {
		BackfillEnabled bool `json:"backfillEnabled"`
	}
	JobResourceSummaryUsageResp struct {
		Used  string  `json:"used"`
		Limit *string `json:"limit,omitempty"`
	}
	JobResourceSummaryAcceleratorResp struct {
		Resource string  `json:"resource"`
		Used     string  `json:"used"`
		Limit    *string `json:"limit,omitempty"`
	}
	JobResourceSummaryResp struct {
		QueueName    string                              `json:"queueName"`
		QuotaEnabled bool                                `json:"quotaEnabled"`
		OccupiedJobs int                                 `json:"occupiedJobs"`
		CPU          JobResourceSummaryUsageResp         `json:"cpu"`
		Memory       JobResourceSummaryUsageResp         `json:"memory"`
		Accelerators []JobResourceSummaryAcceleratorResp `json:"accelerators"`
	}
)

const (
	jobResourceSummaryScopePersonal JobResourceSummaryScope = "personal"
	jobResourceSummaryScopeAccount  JobResourceSummaryScope = "account"
)

// GetPrequeueStatus godoc
//
//	@Summary		获取回填提交开关状态
//	@Description	返回当前是否允许提交 backfill 作业
//	@Tags			Context
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[PrequeueFeatureStatusResp]	"当前状态"
//	@Failure		500	{object}	resputil.Response[any]						"服务器错误"
//	@Router			/v1/context/prequeue [get]
func (mgr *ContextMgr) GetPrequeueStatus(c *gin.Context) {
	if mgr.configService == nil {
		resputil.Error(c, "config service is not initialized", resputil.ServiceError)
		return
	}

	cfg, err := mgr.configService.GetPrequeueConfig(c.Request.Context())
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	resputil.Success(c, PrequeueFeatureStatusResp{BackfillEnabled: cfg.BackfillEnabled})
}

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

	// Query resource limits from user database
	resources := make(map[v1.ResourceName]payload.ResourceResp)

	// Process allocated resources
	for name, quantity := range allocated {
		if name == v1.ResourceCPU || name == v1.ResourceMemory || strings.Contains(string(name), "/") {
			resources[name] = payload.ResourceResp{
				Label: string(name),
				Allocated: ptr.To(payload.ResourceBase{
					Amount: quantity.Value(),
					Format: string(quantity.Format),
				}),
			}
		}
	}

	// Get resource limits from database user
	ua := query.UserAccount
	userAccount, err := ua.WithContext(c).Where(ua.AccountID.Eq(token.AccountID), ua.UserID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, "Failed to query user account", resputil.NotSpecified)
		return
	}
	capability := userAccount.Quota.Data().Capability
	for name := range resources {
		v, exists := capability[name]
		if !exists {
			continue
		}
		resource := resources[name]
		resource.Capability = ptr.To(payload.ResourceBase{
			Amount: v.Value(),
			Format: string(v.Format),
		})
		resources[name] = resource
	}

	// Extract CPU, memory and GPU resources
	cpu := resources[v1.ResourceCPU]
	cpu.Label = "cpu"
	memory := resources[v1.ResourceMemory]
	memory.Label = "mem"
	var gpus []payload.ResourceResp
	for name, resource := range resources {
		if strings.Contains(string(name), "/") {
			split := strings.Split(string(name), "/")
			if len(split) == 2 {
				resource.Label = split[1]
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

// GetJobResourceSummary godoc
//
//	@Summary		获取当前资源占用汇总
//	@Description	按个人或账户视角汇总运行中与排队中的作业资源占用，并返回队列内资源限制
//	@Tags			Context
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			scope	query		string									false	"资源占用视角"	default(personal)	Enums(personal,account)
//	@Success		200		{object}	resputil.Response[JobResourceSummaryResp]	"当前作业资源占用汇总"
//	@Failure		400		{object}	resputil.Response[any]					"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]					"服务器错误"
//	@Router			/v1/context/job-resource-summary [get]
func (mgr *ContextMgr) GetJobResourceSummary(c *gin.Context) {
	token := util.GetToken(c)
	scope, err := parseJobResourceSummaryScope(c.DefaultQuery("scope", string(jobResourceSummaryScopePersonal)))
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var summary *service.UserResourceUsageSummary
	switch scope {
	case jobResourceSummaryScopePersonal:
		summary, err = mgr.queueQuotaSvc.GetUserResourceUsageSummary(
			c.Request.Context(),
			token.UserID,
			token.AccountID,
			vcqueue.ResolveJobQueueName(token),
		)
	case jobResourceSummaryScopeAccount:
		if token.AccountID == model.DefaultAccountID {
			resputil.BadRequestError(c, "account scope is not supported for default account")
			return
		}
		summary, err = mgr.getAccountResourceUsageSummary(c, token)
	default:
		resputil.BadRequestError(c, fmt.Sprintf("invalid scope %q", scope))
		return
	}
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	accelerators := make([]JobResourceSummaryAcceleratorResp, 0, len(summary.Resources))
	for resourceName, item := range summary.Resources {
		if resourceName == string(v1.ResourceCPU) || resourceName == string(v1.ResourceMemory) {
			continue
		}
		if !strings.Contains(resourceName, "/") || !hasPositiveQuantity(item.Used) {
			continue
		}

		accelerator := JobResourceSummaryAcceleratorResp{
			Resource: resourceName,
			Used:     item.Used,
		}
		if item.HasLimit {
			accelerator.Limit = ptr.To(item.Limit)
		}
		accelerators = append(accelerators, accelerator)
	}

	sort.Slice(accelerators, func(i, j int) bool {
		return accelerators[i].Resource < accelerators[j].Resource
	})

	resputil.Success(c, JobResourceSummaryResp{
		QueueName:    summary.QueueName,
		QuotaEnabled: summary.QuotaEnabled,
		OccupiedJobs: summary.OccupiedJobs,
		CPU:          buildJobResourceSummaryUsage(summary.Resources[string(v1.ResourceCPU)]),
		Memory:       buildJobResourceSummaryUsage(summary.Resources[string(v1.ResourceMemory)]),
		Accelerators: accelerators,
	})
}

func parseJobResourceSummaryScope(raw string) (JobResourceSummaryScope, error) {
	scope := JobResourceSummaryScope(strings.TrimSpace(raw))
	switch scope {
	case "", jobResourceSummaryScopePersonal:
		return jobResourceSummaryScopePersonal, nil
	case jobResourceSummaryScopeAccount:
		return jobResourceSummaryScopeAccount, nil
	default:
		return "", fmt.Errorf("invalid scope %q", raw)
	}
}

func (mgr *ContextMgr) getAccountResourceUsageSummary(
	c *gin.Context,
	token util.JWTMessage,
) (*service.UserResourceUsageSummary, error) {
	queueName := vcqueue.GetAccountLogicQueueName(token.AccountID)
	queue := &scheduling.Queue{}
	if err := mgr.client.Get(c, client.ObjectKey{
		Name:      queueName,
		Namespace: config.GetConfig().Namespaces.Job,
	}, queue); err != nil {
		return nil, fmt.Errorf("failed to get account queue %s: %w", queueName, err)
	}

	occupiedJobs, err := mgr.queueQuotaSvc.CountAccountRunningJobs(c.Request.Context(), token.AccountID)
	if err != nil {
		return nil, err
	}

	return service.BuildQueueResourceUsageSummary(
		queueName,
		queue.Status.Allocated,
		queue.Spec.Capability,
		occupiedJobs,
	), nil
}

func buildJobResourceSummaryUsage(
	item service.UserResourceUsageSummaryItem,
) JobResourceSummaryUsageResp {
	resp := JobResourceSummaryUsageResp{
		Used: "0",
	}
	if item.Used != "" {
		resp.Used = item.Used
	}
	if item.HasLimit {
		resp.Limit = ptr.To(item.Limit)
	}
	return resp
}

func hasPositiveQuantity(value string) bool {
	if value == "" {
		return false
	}
	quantity, err := apiresource.ParseQuantity(value)
	if err != nil {
		return false
	}
	return quantity.MilliValue() > 0
}

type (
	UserInfoResp struct {
		ID        uint                                    `json:"id"`
		Name      string                                  `json:"name"`
		Attribute datatypes.JSONType[model.UserAttribute] `json:"attributes"`
	}
)

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

// CheckResourceLimit godoc
//
//	@Summary		检查用户资源使用是否超限
//	@Description	根据队列内资源限制配置，检查当前用户运行中作业资源加上本次请求资源是否超限
//	@Tags			Context
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			body	body		ResourceLimitCheckReq							false	"本次请求的资源"
//	@Success		200		{object}	resputil.Response[service.ResourceLimitCheckResult]	"检查结果"
//	@Failure		500		{object}	resputil.Response[any]							"服务器错误"
//	@Router			/v1/context/resource-limit-check [post]
func (mgr *ContextMgr) CheckResourceLimit(c *gin.Context) {
	token := util.GetToken(c)

	var req ResourceLimitCheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	result, err := mgr.queueQuotaSvc.CheckUserResourceLimit(
		c.Request.Context(),
		token.UserID,
		token.AccountID,
		vcqueue.ResolveJobQueueName(token),
		req.RequestedResources,
	)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	resputil.Success(c, result)
}
