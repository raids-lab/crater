package tensorboard

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	tensorboardservice "github.com/raids-lab/crater/internal/service/tensorboard"
	interutil "github.com/raids-lab/crater/internal/util"
)

const (
	tensorboardAccessCookie = "crater_tensorboard_access"
	tensorboardIDLength     = 8
)

type TensorboardMgr struct {
	service *tensorboardservice.TensorboardService
}

//nolint:gochecknoinits // This is the standard way to register a Gin handler.
func init() {
	handler.Registers = append(handler.Registers, NewTensorboardMgr)
}

func NewTensorboardMgr(config *handler.RegisterConfig) handler.Manager {
	tensorboardService := tensorboardservice.NewTensorboardService(
		config.Client,
		config.ServiceManager,
	)
	return &TensorboardMgr{service: tensorboardService}
}

func (mgr *TensorboardMgr) GetName() string { return "tensorboard" }

func (mgr *TensorboardMgr) RegisterPublic(group *gin.RouterGroup) {
	group.GET("/auth", mgr.AuthorizeIngress)
}
func (mgr *TensorboardMgr) RegisterAdmin(_ *gin.RouterGroup) {}
func (mgr *TensorboardMgr) RegisterProtected(group *gin.RouterGroup) {
	group.POST("", mgr.UserCreate)
	group.GET("", mgr.UserList)
	group.GET("/source/:jobName", mgr.UserGetSourceConfig)
	group.DELETE("/:id", mgr.UserDelete)
	group.POST("/:id/extend", mgr.UserExtendTTL)
	group.POST("/:id/access", mgr.UserCreateAccessSession)
}

// UserCreateAccessSession authorizes a browser to open one owned TensorBoard panel.
//
//	@Summary		创建 TensorBoard 访问会话
//	@Description	校验面板所有权，并为该面板路径设置短期登录 Cookie
//	@Tags			TensorBoard
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		string	true	"TensorBoard 面板 ID"
//	@Success		200	{object}	resputil.Response[string]
//	@Failure		401	{object}	resputil.Response[any]
//	@Failure		403	{object}	resputil.Response[any]
//	@Failure		404	{object}	resputil.Response[any]
//	@Failure		500	{object}	resputil.Response[any]
//	@Router			/v1/tensorboard/{id}/access [post]
func (mgr *TensorboardMgr) UserCreateAccessSession(c *gin.Context) {
	token := interutil.GetToken(c)
	accessPath, err := mgr.service.GetAccessPath(c.Request.Context(), token.Username, c.Param("id"))
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	authToken, ok := bearerToken(c.GetHeader("Authorization"))
	if !ok {
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("invalid access token"))
		return
	}
	c.SetSameSite(http.SameSiteStrictMode)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     tensorboardAccessCookie,
		Value:    authToken,
		Path:     "/ingress/" + token.Username + "-" + c.Param("id"),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	resputil.Success(c, accessPath)
}

// AuthorizeIngress validates the login session and ownership for ingress-nginx.
//
//	@Summary		校验 TensorBoard Ingress 访问
//	@Description	供 ingress-nginx external-auth 子请求调用
//	@Tags			TensorBoard
//	@Success		204
//	@Failure		401
//	@Router			/tensorboard/auth [get]
func (mgr *TensorboardMgr) AuthorizeIngress(c *gin.Context) {
	cookie, err := c.Cookie(tensorboardAccessCookie)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	token, err := interutil.GetTokenMgr().CheckToken(cookie)
	if err != nil || !isOwnedTensorboardURL(c.GetHeader("X-Original-URL"), token.Username) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	returnToken := ""
	if len(parts) == 2 && parts[0] == "Bearer" {
		returnToken = parts[1]
	}
	return returnToken, returnToken != ""
}

func isOwnedTensorboardURL(rawURL, username string) bool {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || username == "" {
		return false
	}
	routePrefix := "/ingress/" + username + "-"
	escapedPath := parsed.EscapedPath()
	if !strings.HasPrefix(escapedPath, routePrefix) {
		return false
	}
	remainder := strings.TrimPrefix(escapedPath, routePrefix)
	tensorboardID := strings.SplitN(remainder, "/", 2)[0]
	if len(tensorboardID) != tensorboardIDLength {
		return false
	}
	for _, character := range tensorboardID {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

// UserGetSourceConfig returns TensorBoard settings stored in the selected job configuration.
//
//	@Summary		获取 TensorBoard 来源任务配置
//	@Description	获取当前用户来源任务中声明的 TensorBoard 日志目录
//	@Tags			TensorBoard
//	@Produce		json
//	@Security		Bearer
//	@Param			jobName	path		string	true	"来源任务名称"
//	@Success		200		{object}	resputil.Response[payload.TensorboardSourceConfigResp]
//	@Failure		404		{object}	resputil.Response[any]
//	@Failure		500		{object}	resputil.Response[any]
//	@Router			/v1/tensorboard/source/{jobName} [get]
func (mgr *TensorboardMgr) UserGetSourceConfig(c *gin.Context) {
	token := interutil.GetToken(c)
	result, err := mgr.service.GetSourceConfig(c.Request.Context(), token.UserID, c.Param("jobName"))
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, result)
}

// UserCreate provisions a TensorBoard deployment, service, and ingress.
//
//	@Summary		创建 TensorBoard 面板
//	@Description	从当前用户个人空间的日志目录创建面板，也可关联一个或多个来源任务
//	@Tags			TensorBoard
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body		payload.CreateTensorboardReq	true	"TensorBoard 面板配置"
//	@Success		200		{object}	resputil.Response[payload.CreateTensorboardResp]
//	@Failure		400		{object}	resputil.Response[any]
//	@Failure		404		{object}	resputil.Response[any]
//	@Failure		409		{object}	resputil.Response[any]
//	@Failure		500		{object}	resputil.Response[any]
//	@Router			/v1/tensorboard [post]
func (mgr *TensorboardMgr) UserCreate(c *gin.Context) {
	var req payload.CreateTensorboardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid request body"))
		return
	}

	token := interutil.GetToken(c)
	result, err := mgr.service.Create(c.Request.Context(), token.UserID, token.Username, &req)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, result)
}

// UserExtendTTL extends the expiration time of a TensorBoard panel.
//
//	@Summary		延长 TensorBoard 面板有效期
//	@Description	重新设置当前用户 TensorBoard 面板的过期时间
//	@Tags			TensorBoard
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path		string				true	"TensorBoard 面板 ID"
//	@Param			data	body		payload.ExtendTTLReq	true	"有效期设置"
//	@Success		200		{object}	resputil.Response[string]
//	@Failure		400		{object}	resputil.Response[any]
//	@Failure		403		{object}	resputil.Response[any]
//	@Failure		404		{object}	resputil.Response[any]
//	@Failure		500		{object}	resputil.Response[any]
//	@Router			/v1/tensorboard/{id}/extend [post]
func (mgr *TensorboardMgr) UserExtendTTL(c *gin.Context) {
	var req payload.ExtendTTLReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid request body"))
		return
	}

	token := interutil.GetToken(c)
	result, err := mgr.service.ExtendTTL(c.Request.Context(), token.Username, c.Param("id"), &req)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, result)
}

// UserDelete removes a TensorBoard panel owned by the current user.
//
//	@Summary		删除 TensorBoard 面板
//	@Description	删除 TensorBoard 面板及其关联的网络资源
//	@Tags			TensorBoard
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		string	true	"TensorBoard 面板 ID"
//	@Success		200	{object}	resputil.Response[string]
//	@Failure		403	{object}	resputil.Response[any]
//	@Failure		404	{object}	resputil.Response[any]
//	@Failure		500	{object}	resputil.Response[any]
//	@Router			/v1/tensorboard/{id} [delete]
func (mgr *TensorboardMgr) UserDelete(c *gin.Context) {
	token := interutil.GetToken(c)
	if err := mgr.service.Delete(c.Request.Context(), token.Username, c.Param("id")); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "ok")
}

// UserList returns TensorBoard panels owned by the current user.
//
//	@Summary		获取 TensorBoard 面板列表
//	@Description	获取当前用户创建的 TensorBoard 面板
//	@Tags			TensorBoard
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[[]payload.TensorboardInfo]
//	@Failure		500	{object}	resputil.Response[any]
//	@Router			/v1/tensorboard [get]
func (mgr *TensorboardMgr) UserList(c *gin.Context) {
	token := interutil.GetToken(c)
	result, err := mgr.service.List(c.Request.Context(), token.Username)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, result)
}
