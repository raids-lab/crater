package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	jobtemplateservice "github.com/raids-lab/crater/internal/service/jobtemplate"
	"github.com/raids-lab/crater/internal/util"
)

const jobTemplateMaxSearchRunes = 128

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewJobTemplateMgr)
}

type JobTemplateMgr struct {
	name string
}

func NewJobTemplateMgr(_ *RegisterConfig) Manager {
	return &JobTemplateMgr{
		name: "jobtemplate",
	}
}
func (mgr *JobTemplateMgr) GetName() string { return mgr.name }

func (mgr *JobTemplateMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *JobTemplateMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.ListJobTemplates)
	g.GET("/list", mgr.ListJobTemplate)
	g.POST("/create", mgr.CreateJobTemplate)
	g.DELETE("/delete/:id", mgr.DeleteJobTemplate)
	g.GET("/:id", mgr.GetJobTemplate)
	g.PUT("/update/:id", mgr.UpdateJobTemplate)
}

func (mgr *JobTemplateMgr) RegisterAdmin(_ *gin.RouterGroup) {
	// Implement admin-specific routes here if needed
}

type JobTemplateresp struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Describe  string    `json:"describe"`
	Document  string    `json:"document"`
	Template  string    `json:"template"`
	CreatedAt time.Time `json:"createdAt"`

	UserInfo model.UserInfo `json:"userInfo"`
}

type jobTemplateListQuery struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=200"`
	Search   string `form:"search"`
	Owner    string `form:"owner,default=all" binding:"oneof=all mine others"`
	Sort     string `form:"sort,default=-createdAt" binding:"oneof=createdAt -createdAt"`
}

func bindJobTemplateListQuery(c *gin.Context) (jobTemplateListQuery, error) {
	var request jobTemplateListQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		return jobTemplateListQuery{}, bizerr.BadRequest.ParameterError.Wrap(
			err,
			"invalid job template list query",
		)
	}
	var err error
	request.Search, err = pagination.NormalizePageQuery(request.Search, request.Page, request.PageSize)
	if err != nil {
		return jobTemplateListQuery{}, err
	}
	return request, nil
}

func (request jobTemplateListQuery) offset() int {
	return (request.Page - 1) * request.PageSize
}

func convertJobTemplateResp(templates []*model.Jobtemplate) []JobTemplateresp {
	result := make([]JobTemplateresp, 0, len(templates))
	for _, template := range templates {
		result = append(result, JobTemplateresp{
			ID:        template.ID,
			Name:      template.Name,
			Describe:  template.Describe,
			Document:  template.Document,
			Template:  template.Template,
			CreatedAt: template.CreatedAt,
			UserInfo: model.UserInfo{
				Username: template.User.Name,
				Nickname: template.User.Nickname,
			},
		})
	}
	return result
}

// ListJobTemplates godoc
//
//	@Summary		List job templates
//	@Description	List job templates with server-side pagination, search, owner filtering, and sorting
//	@Tags			jobtemplate
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			page		query	int		false	"Page number"
//	@Param			page_size	query	int		false	"Page size, 1-200"
//	@Param			search		query	string	false	"Search template names"
//	@Param			owner		query	string	false	"Owner scope: all, mine, or others"
//	@Param			sort		query	string	false	"Sort by createdAt or -createdAt"
//	@Success		200	{object}	resputil.Response[resputil.Page[JobTemplateresp]]	"Job template page"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/jobtemplate [get]
func (mgr *JobTemplateMgr) ListJobTemplates(c *gin.Context) {
	request, err := bindJobTemplateListQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	token := util.GetToken(c)
	templates, total, err := jobtemplateservice.List(c, jobtemplateservice.ListOptions{
		Offset: request.offset(),
		Limit:  request.PageSize,
		Search: request.Search,
		Owner:  request.Owner,
		Sort:   request.Sort,
		UserID: token.UserID,
	})
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list job templates failed"))
		return
	}

	resputil.Success(c, resputil.NewPage(
		convertJobTemplateResp(templates),
		total,
		request.Page,
		request.PageSize,
	))
}

// swagger
//
//	@Summary		展示所有作业模板
//	@Description	展示所有作业模板
//	@Tags			jobtemplate
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"成功返回值描述"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/jobtemplate/list [get]
func (mgr *JobTemplateMgr) ListJobTemplate(c *gin.Context) {
	j := query.Jobtemplate
	templates, err := j.WithContext(c).
		Preload(j.User).
		Where(j.ID.IsNotNull()).
		Find()
	if err != nil {
		resputil.Error(c, "Can't get jobtemplates", resputil.NotSpecified)
		return
	}
	resputil.Success(c, convertJobTemplateResp(templates))
}

type GetJobTemplateReq struct {
	ID uint `uri:"id" binding:"required"` // 关键修复：使用 uri 标签
}

// swagger
//
//	@Summary		获取作业模板
//	@Description	获取作业模板
//	@Tags			jobtemplate
//	@Accept			json
//	@Produce		json
//	@Param			id	path	int	true	"作业模板ID"	//	Swagger	注解正确
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"成功返回值描述"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/jobtemplate/{id} [get]
func (mgr *JobTemplateMgr) GetJobTemplate(c *gin.Context) {
	var req GetJobTemplateReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	j := query.Jobtemplate
	jobtemplate, err := j.WithContext(c).Preload(j.User).Where(j.ID.Eq(req.ID)).First()
	if err != nil {
		resputil.Error(c, "Failed to get job template", resputil.NotSpecified)
		return
	}

	jobtemplateResp := JobTemplateresp{
		ID:        jobtemplate.ID,
		Name:      jobtemplate.Name,
		Describe:  jobtemplate.Describe,
		Document:  jobtemplate.Document,
		Template:  jobtemplate.Template,
		CreatedAt: jobtemplate.CreatedAt,
		UserInfo: model.UserInfo{
			Username: jobtemplate.User.Name,
			Nickname: jobtemplate.User.Nickname,
		},
	}

	resputil.Success(c, jobtemplateResp)
}

type JobTemplateReq struct {
	Name     string `json:"name" binding:"required"`
	Describe string `json:"describe"`
	Document string `json:"document"`
	Template string `json:"template" binding:"required"`
}

// @Summary		创建作业模板
// @Description	创建作业模板
// @Tags			jobtemplate
// @Accept			json
// @Produce		json
// @Param			req	body		JobTemplateReq				true	"作业模板"
// @Success		200	{object}	resputil.Response[string]	"成功返回值描述"
// @Security		Bearer
// @Success		200	{object}	resputil.Response[any]	"成功返回值描述"
// @Failure		400	{object}	resputil.Response[any]	"Request parameter error"
// @Failure		500	{object}	resputil.Response[any]	"Other errors"
// @Router			/v1/jobtemplate/create [post]
func (mgr *JobTemplateMgr) CreateJobTemplate(c *gin.Context) {
	token := util.GetToken(c)
	var jobtemplateReq JobTemplateReq
	if err := c.ShouldBindJSON(&jobtemplateReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	jobtemplate := model.Jobtemplate{
		Name:     jobtemplateReq.Name,
		Describe: jobtemplateReq.Describe,
		Document: jobtemplateReq.Document,
		Template: jobtemplateReq.Template,
		UserID:   token.UserID,
	}

	if err := query.Jobtemplate.WithContext(c).Create(&jobtemplate); err != nil {
		resputil.Error(c, "Failed to create job template", resputil.NotSpecified)
		return
	}

	resputil.Success(c, gin.H{"id": jobtemplate.ID})
}

type DeleteJobtemplateReq struct {
	ID uint `uri:"id" binding:"required"` // 关键修复：使用 uri 标签
}

// DeleteJobTemplate godoc
//
//	@Summary		删除作业模板
//	@Description	删除作业模板
//	@Tags			jobtemplate
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int							true	"作业模板ID"	//	Swagger	注解正确
//	@Success		200	{object}	resputil.Response[string]	"成功返回值描述"
//	@Failure		400	{object}	resputil.Response[any]		"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]		"Other errors"
//	@Router			/v1/jobtemplate/delete/{id} [delete]
func (mgr *JobTemplateMgr) DeleteJobTemplate(c *gin.Context) {
	var req DeleteJobtemplateReq
	if err := c.ShouldBindUri(&req); err != nil { // ✅ 正确绑定路径参数
		resputil.BadRequestError(c, err.Error())
		return
	}

	result, err := query.Jobtemplate.WithContext(c).Where(query.Jobtemplate.ID.Eq(req.ID)).Delete()
	if err != nil {
		resputil.Error(c, "Failed to delete job template", resputil.NotSpecified)
		return
	}

	if result.RowsAffected == 0 {
		resputil.Error(c, "No job template found to delete", resputil.NotSpecified)
		return
	}

	resputil.Success(c, "Delete job template success")
}

type UpdateJobTemplateReq struct {
	ID       uint   `json:"id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Describe string `json:"describe"`
	Document string `json:"document"`
	Template string `json:"template" binding:"required"`
}

// swagger
//
//	@Summary		更新作业模板
//	@Description	更新作业模板
//	@Tags			jobtemplate
//	@Accept			json
//	@Produce		json
//	@Param			req	body	UpdateJobTemplateReq	true	"作业模板"
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[string]	"成功返回值描述"
//	@Failure		400	{object}	resputil.Response[any]		"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]		"Other errors"
//	@Router			/v1/jobtemplate/update [put]
func (mgr *JobTemplateMgr) UpdateJobTemplate(c *gin.Context) {
	var jobtemplateReq UpdateJobTemplateReq
	if err := c.ShouldBindJSON(&jobtemplateReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	jobtemplate := model.Jobtemplate{
		Name:     jobtemplateReq.Name,
		Describe: jobtemplateReq.Describe,
		Document: jobtemplateReq.Document,
		Template: jobtemplateReq.Template,
	}

	if _, err := query.Jobtemplate.WithContext(c).Where(query.Jobtemplate.ID.Eq(jobtemplateReq.ID)).Updates(&jobtemplate); err != nil {
		resputil.Error(c, "Failed to update job template", resputil.NotSpecified)
		return
	}

	resputil.Success(c, "Update job template success")
}
