package handler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/rand"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/alert"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/utils"
)

// 邮箱验证码缓存
var verifyCodeCache = make(map[string]string)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewContextMgr)
}

type ContextMgr struct {
	name   string
	client client.Client
}

func NewContextMgr(conf *RegisterConfig) Manager {
	return &ContextMgr{
		name:   "context",
		client: conf.Client,
	}
}

func (mgr *ContextMgr) GetName() string { return mgr.name }

func (mgr *ContextMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ContextMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("quota", mgr.GetQuota)
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
// @Summary Get the queue information
// @Description query the queue information by client-go
// @Tags Context
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "Volcano Queue Quota"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "other errors"
// @Router /v1/context/queue [get]
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

type (
	UserInfoResp struct {
		ID        uint                                    `json:"id"`
		Name      string                                  `json:"name"`
		Attribute datatypes.JSONType[model.UserAttribute] `json:"attributes"`
	}
)

// UpdateUserAttributes godoc
// @Summary Update user attributes
// @Description Update the attributes of the current user
// @Tags Context
// @Accept json
// @Produce json
// @Security Bearer
// @Param attributes body model.UserAttribute true "User attributes"
// @Success 200 {object} resputil.Response[any] "User attributes updated"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/context/attributes [put]
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
// @Summary Send User Verification Code for email
// @Description generate random code and save, send it to the user's email
// @Tags Context
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "Successfully send email verification code to user"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "other errors"
// @Router /v1/context/email/code [post]
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
// @Summary Update after judging Verification Code for email
// @Description judge code and update email for user
// @Tags Context
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "User email updated successfully"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "other errors"
// @Router /v1/context/email/update [post]
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
		logutils.Log.Error("Failed to update LastEmailVerifiedAt", err)
	}

	resputil.Success(c, "User email updated successfully")
}
