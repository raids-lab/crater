package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	ldap "github.com/go-ldap/ldap/v3"
	imrocreq "github.com/imroc/req/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewAuthMgr)
}

type AuthMgr struct {
	name     string
	client   *http.Client
	req      *imrocreq.Client
	openAPI  config.ACTOpenAPI
	tokenMgr *util.TokenManager
}

func NewAuthMgr(_ *RegisterConfig) Manager {
	return &AuthMgr{
		name:     "auth",
		client:   &http.Client{},
		req:      imrocreq.C(),
		tokenMgr: util.GetTokenMgr(),
		openAPI:  config.GetConfig().RaidsLab.OpenAPI,
	}
}

func (mgr *AuthMgr) GetName() string { return mgr.name }

func (mgr *AuthMgr) RegisterPublic(g *gin.RouterGroup) {
	g.POST("login", mgr.Login)
	g.GET("check", mgr.Check)
	g.POST("signup", mgr.Signup)
	g.POST("refresh", mgr.RefreshToken)
	g.GET("mode", mgr.GetAuthMode)
}

func (mgr *AuthMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("switch", mgr.SwitchQueue) // 切换项目 /switch
}

func (mgr *AuthMgr) RegisterAdmin(_ *gin.RouterGroup) {}

type (
	LoginReq struct {
		AuthMethod AuthMethod `json:"auth" binding:"required"` // [normal, act-ldap, act-api]
		Username   *string    `json:"username"`                // (act-ldap, normal)
		Password   *string    `json:"password"`                // (act-ldap, normal)
		Token      *string    `json:"token"`                   // (act-api)
	}

	LoginResp struct {
		AccessToken  string              `json:"accessToken"`
		RefreshToken string              `json:"refreshToken"`
		Context      AccountContext      `json:"context"`
		User         model.UserAttribute `json:"user"`
	}

	CheckResp struct {
		Context AccountContext      `json:"context"`
		User    model.UserAttribute `json:"user"`
	}

	AccountContext struct {
		Queue        string           `json:"queue"`        // Current Queue Name
		RoleQueue    model.Role       `json:"roleQueue"`    // User role of the queue
		RolePlatform model.Role       `json:"rolePlatform"` // User role of the platform
		AccessQueue  model.AccessMode `json:"accessQueue"`  // User access mode of the queue
		AccessPublic model.AccessMode `json:"accessPublic"` // User access mode of the platform
		Space        string           `json:"space"`        // User pvc subpath the platform
	}
)

type AuthMode string

const (
	AuthModeACT    AuthMode = "act"
	AuthModeNormal AuthMode = "normal"
)

type AuthMethod string

const (
	AuthMethodNormal  AuthMethod = "normal"
	AuthMethodACTLDAP AuthMethod = "act-ldap"
	AuthMethodACTAPI  AuthMethod = "act-api"

	UserNamePattern = `^[a-z][a-z0-9-]*$`
)

// GetAuthMode godoc
//
//	@Summary		获取后端用户认证模式
//	@Description	返回后端部署的config值
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	resputil.Response[string]	"启用认证类型"
//	@Failure		400	{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500	{object}	resputil.Response[any]		"获取相关配置时错误"
//	@Router			/auth/mode [get]
func (mgr *AuthMgr) GetAuthMode(c *gin.Context) {
	if config.GetConfig().RaidsLab.Enable {
		resputil.Success(c, "act")
		return
	}
	resputil.Success(c, "normal")
}

// Check godoc
//
//	@Summary		验证用户token并返回用户信息
//	@Description	验证Authorization header中的Bearer token，返回用户信息和上下文
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[CheckResp]	"token验证成功，返回用户信息和上下文"
//	@Failure		401	{object}	resputil.Response[any]			"token无效或已过期"
//	@Failure		500	{object}	resputil.Response[any]			"服务器内部错误"
//	@Router			/auth/check [get]
func (mgr *AuthMgr) Check(c *gin.Context) {
	// 从Authorization header中提取token
	authHeader := c.Request.Header.Get("Authorization")
	parts := strings.Split(authHeader, " ")
	if len(parts) < 2 || parts[0] != "Bearer" {
		resputil.Success(c, nil)
		return
	}

	token := parts[1]

	// 验证token
	jwtMessage, err := mgr.tokenMgr.CheckToken(token)
	if err != nil {
		resputil.HTTPError(c, http.StatusUnauthorized, err.Error(), resputil.TokenExpired)
		return
	}

	// 从数据库获取用户信息
	u := query.User
	q := query.Account
	uq := query.UserAccount

	user, err := u.WithContext(c).Where(u.ID.Eq(jwtMessage.UserID)).First()
	if err != nil {
		resputil.Success(c, nil)
		return
	}

	// 检查用户状态
	if user.Status != model.StatusActive {
		resputil.Success(c, nil)
		return
	}

	// 获取当前队列信息
	currentQueue, err := q.WithContext(c).Where(q.ID.Eq(jwtMessage.AccountID)).First()
	if err != nil {
		resputil.Success(c, nil)
		return
	}

	// 获取用户队列信息
	userQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID), uq.AccountID.Eq(jwtMessage.AccountID)).First()
	if err != nil {
		resputil.Success(c, nil)
		return
	}

	// 获取公共访问权限
	publicAccessMode := model.AccessModeNA
	defaultUserQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID), uq.AccountID.Eq(model.DefaultAccountID)).First()
	if err == nil {
		publicAccessMode = defaultUserQueue.AccessMode
	}

	// 构造响应
	checkResponse := CheckResp{
		Context: AccountContext{
			Queue:        currentQueue.Name,
			RoleQueue:    userQueue.Role,
			RolePlatform: user.Role,
			AccessQueue:  userQueue.AccessMode,
			AccessPublic: publicAccessMode,
			Space:        user.Space,
		},
		User: user.Attributes.Data(),
	}

	resputil.Success(c, checkResponse)
}

// Login godoc
//
//	@Summary		用户登录
//	@Description	校验用户身份，生成包含当前用户和项目的 JWT Token
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			data	body		LoginReq						false	"查询参数"
//	@Success		200		{object}	resputil.Response[LoginResp]	"登录成功，返回 JWT Token 和默认个人项目"
//	@Failure		400		{object}	resputil.Response[any]			"请求参数错误"
//	@Failure		401		{object}	resputil.Response[any]			"用户名或密码错误"
//	@Failure		500		{object}	resputil.Response[any]			"数据库交互错误"
//	@Router			/auth/login [post]
//
//nolint:gocyclo // TODO(liyilong): refactor this
func (mgr *AuthMgr) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var username, password, token string
	switch req.AuthMethod {
	case AuthMethodACTAPI:
		if req.Token == nil {
			resputil.HTTPError(c, http.StatusBadRequest, "Token not provided", resputil.InvalidRequest)
			return
		}
		token = *req.Token
	case AuthMethodACTLDAP, AuthMethodNormal:
		if req.Username == nil || req.Password == nil {
			resputil.HTTPError(c, http.StatusBadRequest, "Username or password not provided", resputil.InvalidRequest)
			return
		}
		username = *req.Username
		password = *req.Password
		// Username must start with lowercase letter and can only contain lowercase letters, numbers, and hyphens
		if !regexp.MustCompile(UserNamePattern).MatchString(username) {
			klog.Error("invalid username")
			resputil.HTTPError(c, http.StatusBadRequest, "Invalid username", resputil.InvalidRequest)
			return
		}
	default:
		resputil.HTTPError(c, http.StatusBadRequest, "Invalid auth method", resputil.InvalidRequest)
		return
	}

	// Check if request auth method is valid
	var attributes model.UserAttribute
	allowRegister := false
	switch req.AuthMethod {
	case AuthMethodACTAPI:
		if err := mgr.actAPIAuth(c, token, &attributes); err != nil {
			klog.Error("invalid token: ", err)
			resputil.HTTPError(c, http.StatusUnauthorized, "Invalid token", resputil.NotSpecified)
			return
		}
		allowRegister = true
	case AuthMethodACTLDAP:
		if err := mgr.actLDAPAuth(c, username, password); err != nil {
			klog.Error("invalid credentials: ", err)
			resputil.HTTPError(c, http.StatusUnauthorized, "Invalid credentials", resputil.InvalidCredentials)
			return
		}
		allowRegister = !config.GetConfig().RaidsLab.Enable
	case AuthMethodNormal:
		if err := mgr.normalAuth(c, username, password); err != nil {
			klog.Error("invalid credentials: ", err)
			resputil.HTTPError(c, http.StatusUnauthorized, "Invalid credentials", resputil.InvalidCredentials)
			return
		}
	default:
		klog.Error("invalid auth method: ", req.AuthMethod)
		resputil.HTTPError(c, http.StatusBadRequest, "Invalid auth method", resputil.InvalidRequest)
		return
	}

	// Check if the user exists, and should create user or return error
	user, err := mgr.getOrCreateUser(c, &req, &attributes, allowRegister)
	if err != nil {
		if errors.Is(err, ErrorMustRegister) {
			klog.Error("user must register before login")
			resputil.Error(c, "User must register before login", resputil.MustRegister)
			return
		} else if errors.Is(err, ErrorUIDServerConnect) {
			klog.Error("can't connect to UID server")
			resputil.Error(c, "Can't connect to UID server", resputil.RegisterTimeout)
			return
		} else if errors.Is(err, ErrorUIDServerNotFound) {
			klog.Error("UID not found")
			resputil.Error(c, "UID not found", resputil.RegisterNotFound)
			return
		} else {
			klog.Error("create or update user", err)
			resputil.Error(c, "Create or update user failed", resputil.NotSpecified)
			return
		}
	}

	if err = updateUserIfNeeded(c, user, &attributes); err != nil {
		klog.Error("create or update user", err)
		resputil.Error(c, "Create or update user failed", resputil.NotSpecified)
		return
	}

	if user.Status != model.StatusActive {
		klog.Error("user is not active")
		resputil.HTTPError(c, http.StatusUnauthorized, "User is not active", resputil.NotSpecified)
		return
	}

	q := query.Account
	uq := query.UserAccount

	lastUserQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID)).Last()
	if err != nil {
		klog.Error("user has no queue", err)
		resputil.Error(c, "User must has at least one queue", resputil.UserNotAllowed)
		return
	}

	lastQueue, err := q.WithContext(c).Where(q.ID.Eq(lastUserQueue.AccountID)).First()
	if err != nil {
		klog.Error("user has no queue", err)
		resputil.Error(c, "User must has at least one queue", resputil.UserNotAllowed)
		return
	}

	publicAccessMode := model.AccessModeNA
	defaultUserQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID), uq.AccountID.Eq(model.DefaultAccountID)).First()
	if err == nil {
		publicAccessMode = defaultUserQueue.AccessMode
	}

	// Generate JWT tokens
	jwtMessage := util.JWTMessage{
		UserID:            user.ID,
		Username:          user.Name,
		AccountID:         lastQueue.ID,
		AccountName:       lastQueue.Name,
		RoleAccount:       lastUserQueue.Role,
		AccountAccessMode: lastUserQueue.AccessMode,
		PublicAccessMode:  publicAccessMode,
		RolePlatform:      user.Role,
	}
	accessToken, refreshToken, err := mgr.tokenMgr.CreateTokens(&jwtMessage)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}
	loginResponse := LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Context: AccountContext{
			Queue:        lastQueue.Name,
			RoleQueue:    lastUserQueue.Role,
			RolePlatform: user.Role,
			AccessQueue:  lastUserQueue.AccessMode,
			AccessPublic: publicAccessMode,
			Space:        user.Space,
		},
		User: user.Attributes.Data(),
	}
	resputil.Success(c, loginResponse)
}

var (
	ErrorMustRegister      = errors.New("user must be registered before login")
	ErrorUIDServerConnect  = errors.New("can't connect to UID server")
	ErrorUIDServerNotFound = errors.New("UID not found")
)

func (mgr *AuthMgr) getOrCreateUser(
	c context.Context,
	req *LoginReq,
	attr *model.UserAttribute,
	allowCreate bool,
) (*model.User, error) {
	// initialize username and nickname
	if attr.Name == "" && req.Username != nil {
		attr.Name = *req.Username
	}
	if attr.Nickname == "" && req.Username != nil {
		attr.Nickname = *req.Username
	}

	u := query.User
	user, err := u.WithContext(c).Where(u.Name.Eq(attr.Name)).First()
	if err == nil {
		return user, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// User not found in the database
	if allowCreate {
		// User exists in the auth method but not in the database, create a new user
		return mgr.createUser(c, attr.Name, nil)
	}

	return nil, ErrorMustRegister
}

// updateUserIfNeeded updates the user attributes if they have changed.
// It takes a context, a user model, and the new attributes to be updated.
// It returns an error if the update fails.
func updateUserIfNeeded(
	c context.Context,
	user *model.User,
	attr *model.UserAttribute,
) error {
	// Return early if user attributes do not need updating
	attr.ID = user.ID

	// If don't need to update the user, return directly
	currentAttr := user.Attributes.Data()

	attr.Avatar = currentAttr.Avatar
	attr.UID = currentAttr.UID
	attr.GID = currentAttr.GID

	newAttr := datatypes.NewJSONType(*attr)

	// if attr contains email, this attr may be from act-api
	if currentAttr.ID != model.InvalidUserID &&
		(attr.Email == nil || reflect.DeepEqual(currentAttr, *attr)) {
		return nil
	}

	// dont update email if it has been set
	if currentAttr.Email != nil {
		attr.Email = currentAttr.Email
	}

	if currentAttr.ID != model.InvalidUserID &&
		(attr.Email == nil || reflect.DeepEqual(user.Attributes, newAttr)) {
		return nil
	}

	u := query.User
	if _, err := u.WithContext(c).
		Where(u.ID.Eq(user.ID)).
		Updates(map[string]any{
			"attributes": datatypes.NewJSONType(*attr),
			"nickname":   attr.Nickname,
		}); err != nil {
		return err
	}

	user.Attributes = newAttr
	return nil
}

type (
	ActUIDServerSuccessResp struct {
		GID string `json:"gid"`
		UID string `json:"uid"`
	}

	ActUIDServerErrorResp struct {
		Error string `json:"error"`
	}
)

// createUser is called when the user is not found in the database
func (mgr *AuthMgr) createUser(c context.Context, name string, password *string) (*model.User, error) {
	u := query.User
	uq := query.UserAccount
	userAttribute := model.UserAttribute{
		UID: ptr.To("1001"),
		GID: ptr.To("1001"),
	}
	// Custom Check if the user is valid in external UID server
	if config.GetConfig().RaidsLab.Enable {
		uidServerURL := config.GetConfig().RaidsLab.UIDServerURL
		var result ActUIDServerSuccessResp
		var errorResult ActUIDServerErrorResp
		if _, err := mgr.req.R().SetQueryParam("username", name).SetSuccessResult(&result).
			SetErrorResult(&errorResult).Get(uidServerURL); err != nil {
			return nil, ErrorUIDServerConnect
		}
		if errorResult.Error != "" {
			return nil, ErrorUIDServerNotFound
		}
		userAttribute.Email = ptr.To(name + "@act.buaa.edu.cn")
		userAttribute.UID = ptr.To(result.UID)
		userAttribute.GID = ptr.To(result.GID)
	}

	var hashedPassword *string
	if password != nil {
		passwordStr := *password
		hashed, err := bcrypt.GenerateFromPassword([]byte(passwordStr), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashedPassword = ptr.To(string(hashed))
	}

	user := model.User{
		Name:       name,
		Nickname:   name,
		Password:   hashedPassword,
		Role:       model.RoleUser,
		Status:     model.StatusActive,
		Space:      name,
		Attributes: datatypes.NewJSONType(userAttribute),
	}
	if err := u.WithContext(c).Create(&user); err != nil {
		return nil, err
	}

	// add default user queue
	userAccount := model.UserAccount{
		UserID:     user.ID,
		AccountID:  model.DefaultAccountID,
		Role:       model.RoleUser,
		AccessMode: model.AccessModeRO,
		// Quota:      datatypes.NewJSONType(model.DefaultQuota),
	}

	if err := uq.WithContext(c).Create(&userAccount); err != nil {
		return nil, err
	}

	// TODO: Create personal directory

	return &user, nil
}

func (mgr *AuthMgr) normalAuth(c *gin.Context, username, password string) error {
	u := query.User
	user, err := u.WithContext(c).Where(u.Name.Eq(username)).First()
	if err != nil {
		return fmt.Errorf("user not found")
	}

	p := user.Password
	if p == nil {
		return fmt.Errorf("user does not have a password")
	}

	if bcrypt.CompareHashAndPassword([]byte(*p), []byte(password)) != nil {
		return fmt.Errorf("wrong username or password")
	}
	return nil
}

type (
	ActAPIAuthReq struct {
		Token string `json:"token"`
		Stamp string `json:"stamp"`
		Sign  string `json:"sign"`
	}

	ActAPIAuthResp struct {
		Data struct {
			Account  string `json:"account"`
			Name     string `json:"name"`
			Email    string `json:"email"`
			Teacher  string `json:"teacher"`
			Group    string `json:"group"`
			AdExpire string `json:"ad_expire"`
		} `json:"data"`
		Sign string `json:"sign"`
	}
)

func (mgr *AuthMgr) actAPIAuth(_ context.Context, token string, attr *model.UserAttribute) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	h := hmac.New(sha256.New, []byte(mgr.openAPI.AccessToken))
	h.Write([]byte(token + timestamp))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Send request to ACT API with body v
	var result ActAPIAuthResp
	resp, err := mgr.req.R().
		SetBody(&ActAPIAuthReq{
			Token: token,
			Stamp: timestamp,
			Sign:  signature,
		}).
		SetHeader("Chameleon-Key", mgr.openAPI.ChameleonKey).
		SetHeader("Content-Type", "application/json").
		SetSuccessResult(&result).
		Post(mgr.openAPI.URL)
	if err != nil {
		return err
	}

	if !resp.IsSuccessState() {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// TODO(liyilong): 验证返回的签名

	attr.Name = result.Data.Account
	attr.Nickname = result.Data.Name
	attr.Email = &result.Data.Email
	attr.Teacher = &result.Data.Teacher
	attr.Group = &result.Data.Group
	attr.ExpiredAt = &result.Data.AdExpire

	return nil
}

func (mgr *AuthMgr) actLDAPAuth(_ context.Context, username, password string) error {
	authConfig := config.GetConfig()
	// ACT 管理员认证
	l, err := ldap.DialURL(authConfig.RaidsLab.LDAP.Address)
	if err != nil {
		return err
	}
	err = l.Bind(authConfig.RaidsLab.LDAP.UserName, authConfig.RaidsLab.LDAP.Password)
	if err != nil {
		return err
	}

	// ACT 管理员搜索用户
	searchRequest := ldap.NewSearchRequest(
		authConfig.RaidsLab.LDAP.SearchDN, // 搜索基准 DN
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(sAMAccountName=%s)", username), // 过滤条件
		[]string{"dn"}, // 返回的属性列表
		nil,
	)

	// 执行搜索请求
	searchResult, err := l.Search(searchRequest)
	if err != nil {
		return err
	}

	if len(searchResult.Entries) != 1 {
		return fmt.Errorf("user not found or too many entries returned")
	}

	// 用户存在，验证用户密码
	if len(searchResult.Entries) == 1 {
		userDN := searchResult.Entries[0].DN
		err = l.Bind(userDN, password)
		if err != nil {
			return err
		}
	}

	return nil
}

type (
	SignupReq struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
)

func (mgr *AuthMgr) Signup(c *gin.Context) {
	var req SignupReq
	if err := c.ShouldBind(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if config.GetConfig().RaidsLab.Enable {
		resputil.Error(c, "User must sign up with token", resputil.NotSpecified)
		return
	}

	u := query.User

	_, err := u.WithContext(c).Where(u.Name.Eq(req.Username)).First()
	if err == nil {
		resputil.Error(c, "User already exists", resputil.InvalidRequest)
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	_, err = mgr.createUser(c, req.Username, &req.Password)
	if err != nil {
		if errors.Is(err, ErrorUIDServerConnect) {
			klog.Error("can't connect to UID server")
			resputil.Error(c, "Can't connect to UID server", resputil.RegisterTimeout)
			return
		} else if errors.Is(err, ErrorUIDServerNotFound) {
			klog.Error("UID not found")
			resputil.Error(c, "UID not found", resputil.RegisterNotFound)
			return
		}
		klog.Error("create new user", err)
		resputil.Error(c, "Create user failed", resputil.NotSpecified)
		return
	}

	resputil.Success(c, "Signup successful")
}

type (
	RefreshReq struct {
		RefreshToken string `json:"refreshToken" binding:"required"` // 不需要添加 `Bearer ` 前缀
	}

	RefreshResp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
)

func (mgr *AuthMgr) RefreshToken(c *gin.Context) {
	var request RefreshReq

	if err := c.ShouldBind(&request); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	chaims, err := mgr.tokenMgr.CheckToken(request.RefreshToken)
	if err != nil {
		resputil.HTTPError(c, http.StatusUnauthorized, "User not found", resputil.NotSpecified)
		return
	}

	accessToken, refreshToken, err := mgr.tokenMgr.CreateTokens(&chaims)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}

	refreshTokenResponse := RefreshResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	resputil.Success(c, refreshTokenResponse)
}

type SwitchQueueReq struct {
	Queue string `json:"queue" binding:"required"`
}

// SwitchQueue godoc
//
//	@Summary		类似登录，切换项目并返回新的 JWT Token
//	@Description	读取body中的项目ID，生成新的 JWT Token
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			project_id	body		SwitchQueueReq					true	"项目ID"
//	@Success		200			{object}	resputil.Response[LoginResp]	"用户上下文"
//	@Failure		400			{object}	resputil.Response[any]			"请求参数错误"
//	@Failure		500			{object}	resputil.Response[any]			"其他错误"
//	@Router			/v1/auth/switch [post]
func (mgr *AuthMgr) SwitchQueue(c *gin.Context) {
	var req SwitchQueueReq
	if err := c.ShouldBind(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	if req.Queue == util.QueueNameNull {
		resputil.Error(c, "Queue not specified", resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)

	// Check queue
	q := query.Account
	uq := query.UserAccount

	queue, err := q.WithContext(c).Where(q.Name.Eq(req.Queue)).First()
	if err != nil {
		resputil.Error(c, "Queue not found", resputil.NotSpecified)
		return
	}

	userQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(token.UserID), uq.AccountID.Eq(queue.ID)).First()
	if err != nil {
		resputil.Error(c, "Queue not found", resputil.NotSpecified)
		return
	}

	// Generate new JWT tokens
	jwtMessage := util.JWTMessage{
		UserID:            token.UserID,
		Username:          token.Username,
		AccountID:         userQueue.AccountID,
		AccountName:       req.Queue,
		RoleAccount:       userQueue.Role,
		RolePlatform:      token.RolePlatform,
		AccountAccessMode: userQueue.AccessMode,
		PublicAccessMode:  token.PublicAccessMode,
	}
	accessToken, refreshToken, err := mgr.tokenMgr.CreateTokens(&jwtMessage)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}
	loginResponse := LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Context: AccountContext{
			Queue:        req.Queue,
			RoleQueue:    userQueue.Role,
			AccessQueue:  userQueue.AccessMode,
			RolePlatform: token.RolePlatform,
			AccessPublic: token.PublicAccessMode,
		},
	}
	resputil.Success(c, loginResponse)
}
