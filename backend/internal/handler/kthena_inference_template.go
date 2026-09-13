package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

const (
	maxKthenaInferenceTemplates          = 50
	maxKthenaInferenceTemplateNameRunes  = 64
	maxKthenaInferenceTemplateDescRunes  = 512
	maxKthenaInferenceTemplateConfigSize = 64 * 1024
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewKthenaInferenceTemplateMgr)
}

// KthenaInferenceTemplateMgr owns private, account-scoped deployment presets.
// It intentionally does not share the legacy global job-template implementation:
// those templates are public, whereas inference presets must never cross user or
// account boundaries.
type KthenaInferenceTemplateMgr struct {
	name          string
	db            *gorm.DB
	configService *service.ConfigService
}

func NewKthenaInferenceTemplateMgr(conf *RegisterConfig) Manager {
	return &KthenaInferenceTemplateMgr{
		// Route managers are mounted below GetName(). Sharing the Kthena root
		// keeps private templates at /v1/kthena/inference-templates rather than
		// exposing an unrelated top-level API namespace.
		name:          "kthena",
		db:            query.GetDB(),
		configService: conf.ConfigService,
	}
}

func (mgr *KthenaInferenceTemplateMgr) GetName() string { return mgr.name }

func (mgr *KthenaInferenceTemplateMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *KthenaInferenceTemplateMgr) RegisterProtected(g *gin.RouterGroup) {
	templates := g.Group("inference-templates", mgr.requireKthenaInferenceEnabled)
	templates.GET("", mgr.ListKthenaInferenceTemplates)
	templates.POST("", mgr.CreateKthenaInferenceTemplate)
	templates.PUT(":id", mgr.UpdateKthenaInferenceTemplate)
	templates.DELETE(":id", mgr.DeleteKthenaInferenceTemplate)
}

func (mgr *KthenaInferenceTemplateMgr) RegisterAdmin(_ *gin.RouterGroup) {}

func (mgr *KthenaInferenceTemplateMgr) requireKthenaInferenceEnabled(c *gin.Context) {
	if mgr.configService == nil || !mgr.configService.IsKthenaInferenceEnabled(c.Request.Context()) {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("Kthena inference feature is disabled"))
		c.Abort()
		return
	}
	c.Next()
}

// KthenaInferenceTemplateReq keeps the complete reusable form payload in a
// version-tolerant JSON object. The create-deployment endpoint remains the
// validation authority when the user applies a saved template.
type KthenaInferenceTemplateReq struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config" binding:"required" swaggertype:"object"`
}

type kthenaInferenceTemplateIDReq struct {
	ID uint `uri:"id" binding:"required"`
}

type KthenaInferenceTemplateResp struct {
	ID          uint            `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config" swaggertype:"object"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// ListKthenaInferenceTemplates godoc
//
//	@Summary		List private Kthena deployment templates
//	@Description	List the current user's templates in the active account. Templates are never shared across users or accounts.
//	@Tags			kthena
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[[]KthenaInferenceTemplateResp]
//	@Failure		500	{object}	resputil.Response[any]
//	@Router			/v1/kthena/inference-templates [get]
func (mgr *KthenaInferenceTemplateMgr) ListKthenaInferenceTemplates(c *gin.Context) {
	if !mgr.templateDBReady(c) {
		return
	}
	token := util.GetToken(c)
	var templates []model.KthenaInferenceTemplate
	if err := mgr.db.WithContext(c.Request.Context()).
		Where("user_id = ? AND account_id = ?", token.UserID, token.AccountID).
		Order("updated_at DESC, id DESC").
		Limit(maxKthenaInferenceTemplates).
		Find(&templates).Error; err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list Kthena inference templates failed"))
		return
	}

	response := make([]KthenaInferenceTemplateResp, 0, len(templates))
	for index := range templates {
		response = append(response, kthenaInferenceTemplateToResp(&templates[index]))
	}
	resputil.Success(c, response)
}

// CreateKthenaInferenceTemplate godoc
//
//	@Summary		Create a private Kthena deployment template
//	@Description	Save the current deployment form as a private template for the current user and account.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		KthenaInferenceTemplateReq	true	"Template"
//	@Success		200		{object}	resputil.Response[KthenaInferenceTemplateResp]
//	@Failure		400		{object}	resputil.Response[any]
//	@Failure		409		{object}	resputil.Response[any]
//	@Failure		500		{object}	resputil.Response[any]
//	@Router			/v1/kthena/inference-templates [post]
func (mgr *KthenaInferenceTemplateMgr) CreateKthenaInferenceTemplate(c *gin.Context) {
	if !mgr.templateDBReady(c) {
		return
	}
	var req KthenaInferenceTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid Kthena inference template request"))
		return
	}
	if err := validateKthenaInferenceTemplateReq(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}

	token := util.GetToken(c)
	var count int64
	if err := mgr.db.WithContext(c.Request.Context()).Model(&model.KthenaInferenceTemplate{}).
		Where("user_id = ? AND account_id = ?", token.UserID, token.AccountID).
		Count(&count).Error; err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count Kthena inference templates failed"))
		return
	}
	if count >= maxKthenaInferenceTemplates {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("Kthena inference template limit reached"))
		return
	}

	var existing model.KthenaInferenceTemplate
	err := mgr.db.WithContext(c.Request.Context()).
		Where("user_id = ? AND account_id = ? AND name = ?", token.UserID, token.AccountID, req.Name).
		First(&existing).Error
	if err == nil {
		resputil.HandleError(c, bizerr.Conflict.ResourceAlreadyExists.New("Kthena inference template name already exists"))
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "check Kthena inference template name failed"))
		return
	}

	template := model.KthenaInferenceTemplate{
		UserID:      token.UserID,
		AccountID:   token.AccountID,
		Name:        req.Name,
		Description: req.Description,
		Config:      datatypes.JSON(append([]byte(nil), req.Config...)),
	}
	if err := mgr.db.WithContext(c.Request.Context()).Create(&template).Error; err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "create Kthena inference template failed"))
		return
	}
	resputil.Success(c, kthenaInferenceTemplateToResp(&template))
}

// UpdateKthenaInferenceTemplate godoc
//
//	@Summary		Update a private Kthena deployment template
//	@Description	Replace a template owned by the current user in the active account.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path		int						true	"Template ID"
//	@Param			request	body		KthenaInferenceTemplateReq	true	"Template"
//	@Success		200		{object}	resputil.Response[KthenaInferenceTemplateResp]
//	@Failure		400		{object}	resputil.Response[any]
//	@Failure		404		{object}	resputil.Response[any]
//	@Failure		500		{object}	resputil.Response[any]
//	@Router			/v1/kthena/inference-templates/{id} [put]
func (mgr *KthenaInferenceTemplateMgr) UpdateKthenaInferenceTemplate(c *gin.Context) {
	if !mgr.templateDBReady(c) {
		return
	}
	var idReq kthenaInferenceTemplateIDReq
	if err := c.ShouldBindUri(&idReq); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid Kthena inference template id"))
		return
	}
	var req KthenaInferenceTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid Kthena inference template request"))
		return
	}
	if err := validateKthenaInferenceTemplateReq(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}

	token := util.GetToken(c)
	template, ok := mgr.findPrivateTemplate(c, idReq.ID, token.UserID, token.AccountID)
	if !ok {
		return
	}
	if req.Name != template.Name {
		var duplicate model.KthenaInferenceTemplate
		err := mgr.db.WithContext(c.Request.Context()).
			Where("user_id = ? AND account_id = ? AND name = ?", token.UserID, token.AccountID, req.Name).
			First(&duplicate).Error
		if err == nil {
			resputil.HandleError(c, bizerr.Conflict.ResourceAlreadyExists.New("Kthena inference template name already exists"))
			return
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "check Kthena inference template name failed"))
			return
		}
	}

	template.Name = req.Name
	template.Description = req.Description
	template.Config = datatypes.JSON(append([]byte(nil), req.Config...))
	if err := mgr.db.WithContext(c.Request.Context()).Save(template).Error; err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "update Kthena inference template failed"))
		return
	}
	resputil.Success(c, kthenaInferenceTemplateToResp(template))
}

// DeleteKthenaInferenceTemplate godoc
//
//	@Summary		Delete a private Kthena deployment template
//	@Description	Delete one template owned by the current user in the active account.
//	@Tags			kthena
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"Template ID"
//	@Success		200	{object}	resputil.Response[string]
//	@Failure		404	{object}	resputil.Response[any]
//	@Failure		500	{object}	resputil.Response[any]
//	@Router			/v1/kthena/inference-templates/{id} [delete]
func (mgr *KthenaInferenceTemplateMgr) DeleteKthenaInferenceTemplate(c *gin.Context) {
	if !mgr.templateDBReady(c) {
		return
	}
	var idReq kthenaInferenceTemplateIDReq
	if err := c.ShouldBindUri(&idReq); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid Kthena inference template id"))
		return
	}
	token := util.GetToken(c)
	template, ok := mgr.findPrivateTemplate(c, idReq.ID, token.UserID, token.AccountID)
	if !ok {
		return
	}
	if err := mgr.db.WithContext(c.Request.Context()).Delete(template).Error; err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "delete Kthena inference template failed"))
		return
	}
	resputil.Success(c, "Kthena inference template deleted")
}

func (mgr *KthenaInferenceTemplateMgr) templateDBReady(c *gin.Context) bool {
	if mgr.db != nil {
		return true
	}
	resputil.HandleError(c, bizerr.Internal.DatabaseError.New("Kthena inference template storage is not initialized"))
	return false
}

func (mgr *KthenaInferenceTemplateMgr) findPrivateTemplate(
	c *gin.Context,
	id uint,
	userID uint,
	accountID uint,
) (*model.KthenaInferenceTemplate, bool) {
	template := &model.KthenaInferenceTemplate{}
	err := mgr.db.WithContext(c.Request.Context()).
		Where("id = ? AND user_id = ? AND account_id = ?", id, userID, accountID).
		First(template).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("Kthena inference template not found"))
		return nil, false
	}
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "find Kthena inference template failed"))
		return nil, false
	}
	return template, true
}

func validateKthenaInferenceTemplateReq(req *KthenaInferenceTemplateReq) error {
	if req == nil {
		return bizerr.BadRequest.MissingParameter.New("kthena inference template is required")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > maxKthenaInferenceTemplateNameRunes {
		return bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("template name must contain 1 to %d characters", maxKthenaInferenceTemplateNameRunes),
		)
	}
	if utf8.RuneCountInString(req.Description) > maxKthenaInferenceTemplateDescRunes {
		return bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("template description cannot exceed %d characters", maxKthenaInferenceTemplateDescRunes),
		)
	}
	if len(req.Config) == 0 || len(req.Config) > maxKthenaInferenceTemplateConfigSize {
		return bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("template config must contain at most %d bytes", maxKthenaInferenceTemplateConfigSize),
		)
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(req.Config, &config); err != nil || config == nil {
		return bizerr.BadRequest.InvalidRequest.New("template config must be a JSON object")
	}
	backendRaw, ok := config["backendType"]
	if !ok {
		return bizerr.BadRequest.MissingParameter.New("template config must include backendType")
	}
	var backendType string
	if err := json.Unmarshal(backendRaw, &backendType); err != nil || strings.TrimSpace(backendType) != kthenaBackendVLLM {
		return bizerr.BadRequest.ParameterError.New("template config backendType must be vLLM")
	}
	return nil
}

func kthenaInferenceTemplateToResp(
	template *model.KthenaInferenceTemplate,
) KthenaInferenceTemplateResp {
	if template == nil {
		return KthenaInferenceTemplateResp{}
	}
	return KthenaInferenceTemplateResp{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Config:      json.RawMessage(append([]byte(nil), template.Config...)),
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
	}
}
