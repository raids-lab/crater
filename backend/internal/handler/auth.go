package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	tokenMgr *util.TokenManager
}

func NewAuthMgr(_ *RegisterConfig) Manager {
	return &AuthMgr{
		name:     "auth",
		client:   &http.Client{},
		req:      imrocreq.C(),
		tokenMgr: util.GetTokenMgr(),
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
		AuthMethod AuthMethod `json:"auth" binding:"required"` // [normal, ldap]
		Username   *string    `json:"username"`                // (ldap, normal)
		Password   *string    `json:"password"`                // (ldap, normal)
		Token      *string    `json:"token"`                   // Legacy ACT API token (deprecated)
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
		Version VersionInfo         `json:"version"`
	}

	AccountContext struct {
		Queue        string           `json:"queue"`        // Current Queue Name
		RoleQueue    model.Role       `json:"roleQueue"`    // User role of the queue
		RolePlatform model.Role       `json:"rolePlatform"` // User role of the platform
		AccessQueue  model.AccessMode `json:"accessQueue"`  // User access mode of the queue
		AccessPublic model.AccessMode `json:"accessPublic"` // User access mode of the platform
		Space        string           `json:"space"`        // User pvc subpath the platform
	}

	VersionInfo struct {
		AppVersion string `json:"appVersion"` // Application version (tag name or short SHA)
		CommitSHA  string `json:"commitSHA"`  // Full commit SHA
		BuildType  string `json:"buildType"`  // Build type (release or development)
		BuildTime  string `json:"buildTime"`  // Build time in UTC
	}
)

// Global version information
var globalVersionInfo VersionInfo

// SetVersionInfo sets the global version information
func SetVersionInfo(appVersion, commitSHA, buildType, buildTime string) {
	globalVersionInfo = VersionInfo{
		AppVersion: appVersion,
		CommitSHA:  commitSHA,
		BuildType:  buildType,
		BuildTime:  buildTime,
	}
}

// GetVersionInfo returns the global version information
func GetVersionInfo() VersionInfo {
	return globalVersionInfo
}

type AuthMode string

const (
	AuthModeLDAP   AuthMode = "ldap"
	AuthModeNormal AuthMode = "normal"
)

type AuthModeResp struct {
	EnableLDAP           bool `json:"enableLdap"`
	EnableNormalLogin    bool `json:"enableNormalLogin"`
	EnableNormalRegister bool `json:"enableNormalRegister"`
}

type AuthMethod string

const (
	AuthMethodNormal AuthMethod = "normal"
	AuthMethodLDAP   AuthMethod = "ldap"
)

type UIDSource string

const (
	UIDSourceNone     UIDSource = "none"
	UIDSourceLDAP     UIDSource = "ldap"
	UIDSourceExternal UIDSource = "external"
	UIDSourceDefault  UIDSource = "default"
)

const LogLevelDebug = 4

// GetAuthMode godoc
//
//	@Summary		获取后端用户认证模式
//	@Description	返回后端部署的认证模式配置
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	resputil.Response[AuthModeResp]	"启用认证类型及注册配置"
//	@Failure		500	{object}	resputil.Response[any]			"获取相关配置时错误"
//	@Router			/auth/mode [get]
func (mgr *AuthMgr) GetAuthMode(c *gin.Context) {
	conf := config.GetConfig().Auth
	resp := AuthModeResp{
		EnableLDAP:           conf.LDAP.Enable,
		EnableNormalLogin:    conf.Normal.AllowLogin,
		EnableNormalRegister: conf.Normal.AllowRegister,
	}
	resputil.Success(c, resp)
}

// Check godoc
//
//	@Summary		验证用户token并返回用户信息
//	@Description	验证Authorization header中的Bearer token，返回用户信息、上下文和版本信息
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[CheckResp]	"token验证成功，返回用户信息、上下文和版本信息"
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
		User:    user.Attributes.Data(),
		Version: GetVersionInfo(),
	}

	resputil.Success(c, checkResponse)
}

func (mgr *AuthMgr) performAuthentication(
	c *gin.Context,
	method AuthMethod,
	username, password string,
	attr *model.UserAttribute,
) (allowRegister bool, err error) {
	conf := config.GetConfig().Auth
	switch method {
	case AuthMethodLDAP:
		if !conf.LDAP.Enable {
			resputil.HTTPError(c, http.StatusForbidden, "LDAP authentication is disabled", resputil.InvalidRequest)
			return false, errors.New("ldap disabled")
		}
		if err := mgr.actLDAPAuth(c, username, password, attr); err != nil {
			if errors.Is(err, ErrorInvalidCredentials) {
				resputil.HTTPError(c, http.StatusUnauthorized, "Invalid username or password", resputil.InvalidCredentials)
				return false, err
			}
			if errors.Is(err, ErrorLdapUserNotFound) {
				resputil.HTTPError(c, http.StatusUnauthorized, "User not found or too many entries returned", resputil.LdapUserNotFound)
				return false, err
			}
			klog.Errorf("LDAP auth failed for user %s: %v", username, err)
			resputil.HTTPError(c, http.StatusUnauthorized, fmt.Sprintf("LDAP authentication failed: %v", err), resputil.LdapError)
			return false, err
		}
		return true, nil
	case AuthMethodNormal:
		if conf.LDAP.Enable && !conf.Normal.AllowLogin {
			resputil.HTTPError(c, http.StatusForbidden, "Normal login is disabled by administrator", resputil.UserNotAllowed)
			return false, errors.New("normal login disabled")
		}
		if err := mgr.normalAuth(c, username, password); err != nil {
			resputil.HTTPError(c, http.StatusUnauthorized, "Invalid credentials", resputil.InvalidCredentials)
			return false, err
		}
		return false, nil
	default:
		resputil.BadRequestError(c, "Invalid authentication method")
		return false, errors.New("invalid auth method")
	}
}

func (mgr *AuthMgr) handleLoginError(c *gin.Context, err error) {
	if errors.Is(err, ErrorMustRegister) {
		resputil.HTTPError(c, http.StatusUnauthorized, "User must register before login", resputil.MustRegister)
	} else if errors.Is(err, ErrorUIDServerConnect) {
		resputil.HTTPError(c, http.StatusBadGateway, "Can't connect to UID server", resputil.UidServiceError)
	} else if errors.Is(err, ErrorUIDServerNotFound) {
		resputil.HTTPError(c, http.StatusNotFound, "UID not found", resputil.UidNotFound)
	} else {
		klog.Errorf("getOrCreateUser failed: %v", err)
		resputil.HTTPError(c, http.StatusInternalServerError, "Create or update user failed", resputil.NotSpecified)
	}
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
func (mgr *AuthMgr) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if req.Token != nil && *req.Token != "" {
		msg := "Legacy token login is no longer supported, just use your LDAP account"
		resputil.HTTPError(c, http.StatusForbidden, msg, resputil.LegacyTokenNotSupported)
		return
	}

	if req.Username == nil || req.Password == nil {
		resputil.BadRequestError(c, "Username or password not provided")
		return
	}
	username := *req.Username
	password := *req.Password

	// Username validation - allow common LDAP characters (letters, numbers, dots, underscores, dashes)
	if username == "" {
		resputil.BadRequestError(c, "Username cannot be empty")
		return
	}

	var attributes model.UserAttribute
	allowRegister, err := mgr.performAuthentication(c, req.AuthMethod, username, password, &attributes)
	if err != nil {
		// Error already handled in performAuthentication
		return
	}

	// Check if the user exists, and should create user or return error
	user, err := mgr.getOrCreateUser(c, &req, &attributes, allowRegister)
	if err != nil {
		mgr.handleLoginError(c, err)
		return
	}

	if err = mgr.updateUserIfNeeded(c, user, &attributes); err != nil {
		klog.Errorf("updateUserIfNeeded failed: %v", err)
		resputil.Error(c, "Update user attributes failed", resputil.NotSpecified)
		return
	}

	if user.Status != model.StatusActive {
		resputil.HTTPError(c, http.StatusUnauthorized, "User is not active", resputil.NotSpecified)
		return
	}

	q := query.Account
	uq := query.UserAccount

	lastUserQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID)).Last()
	if err != nil {
		resputil.Error(c, "User must has at least one queue", resputil.UserNotAllowed)
		return
	}

	lastQueue, err := q.WithContext(c).Where(q.ID.Eq(lastUserQueue.AccountID)).First()
	if err != nil {
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
	// Ensure ID and Name are populated in the response attributes
	respAttr := user.Attributes.Data()
	respAttr.ID = user.ID
	respAttr.Name = user.Name

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
		User: respAttr,
	}
	resputil.Success(c, loginResponse)
}

var (
	ErrorMustRegister       = errors.New("user must be registered before login")
	ErrorUIDServerConnect   = errors.New("can't connect to UID server")
	ErrorUIDServerNotFound  = errors.New("UID not found")
	ErrorInvalidCredentials = errors.New("invalid username or password")
	ErrorLdapUserNotFound   = errors.New("user not found or too many entries returned")
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
		return mgr.createUser(c, attr.Name, nil, attr)
	}

	return nil, ErrorMustRegister
}

func syncField[T comparable](current *T, incoming T, changed *bool) {
	if *current != incoming {
		*current = incoming
		*changed = true
	}
}

func syncPtrField[T comparable](current **T, incoming *T, changed *bool) {
	if incoming != nil && (*current == nil || **current != *incoming) {
		*current = incoming
		*changed = true
	}
}

func (mgr *AuthMgr) syncUserAttributes(
	user *model.User,
	currentAttr *model.UserAttribute,
	newAttr *model.UserAttribute,
) bool {
	changed := false

	// Ensure ID and Name are populated in Attributes JSONB
	syncField(&currentAttr.ID, user.ID, &changed)
	syncField(&currentAttr.Name, user.Name, &changed)

	// Mandatory sync for most fields
	if newAttr.Nickname != "" && user.Nickname != newAttr.Nickname {
		user.Nickname = newAttr.Nickname
		currentAttr.Nickname = newAttr.Nickname
		changed = true
	} else {
		syncField(&currentAttr.Nickname, user.Nickname, &changed)
	}

	// Email protection: only sync if email is NOT verified
	if newAttr.Email != nil && (currentAttr.Email == nil || *currentAttr.Email != *newAttr.Email) {
		if user.LastEmailVerifiedAt == nil {
			currentAttr.Email = newAttr.Email
			changed = true
		} else {
			klog.V(LogLevelDebug).Infof("Skip syncing email for user %s because email is already verified", user.Name)
		}
	}

	syncPtrField(&currentAttr.Teacher, newAttr.Teacher, &changed)
	syncPtrField(&currentAttr.Group, newAttr.Group, &changed)
	syncPtrField(&currentAttr.Phone, newAttr.Phone, &changed)
	syncPtrField(&currentAttr.ExpiredAt, newAttr.ExpiredAt, &changed)

	if newAttr.UID != nil && (currentAttr.UID == nil || *currentAttr.UID != *newAttr.UID) {
		currentAttr.UID = newAttr.UID
		currentAttr.GID = newAttr.GID
		changed = true
	}

	return changed
}

// updateUserIfNeeded updates the user attributes if they have changed from LDAP.
func (mgr *AuthMgr) updateUserIfNeeded(
	c context.Context,
	user *model.User,
	newAttr *model.UserAttribute,
) error {
	currentAttr := user.Attributes.Data()
	changed := mgr.syncUserAttributes(user, &currentAttr, newAttr)

	if !changed {
		return nil
	}

	u := query.User
	if _, err := u.WithContext(c).
		Where(u.ID.Eq(user.ID)).
		Updates(map[string]any{
			"attributes": datatypes.NewJSONType(currentAttr),
			"nickname":   user.Nickname,
		}); err != nil {
		return err
	}

	user.Attributes = datatypes.NewJSONType(currentAttr)
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
func (mgr *AuthMgr) createUser(c context.Context, name string, password *string, attrFromLDAP *model.UserAttribute) (*model.User, error) {
	u := query.User
	uq := query.UserAccount
	userAttribute := model.UserAttribute{
		UID: ptr.To("1001"),
		GID: ptr.To("1001"),
	}

	// Normal user registration (not LDAP)
	if attrFromLDAP == nil {
		// Always use default UID/GID for normal registration
		userAttribute.UID = ptr.To("1001")
		userAttribute.GID = ptr.To("1001")
		// Set default name and nickname to username
		userAttribute.Name = name
		userAttribute.Nickname = name
	} else {
		// LDAP auto-registration: determine UID/GID based on system configuration source
		userAttribute = *attrFromLDAP
		uidConf := config.GetConfig().Auth.LDAP.UID
		switch UIDSource(uidConf.Source) {
		case UIDSourceExternal:
			uidServerURL := uidConf.ExternalService.URL
			var result ActUIDServerSuccessResp
			var errorResult ActUIDServerErrorResp
			if _, err := mgr.req.R().SetQueryParam("username", name).SetSuccessResult(&result).
				SetErrorResult(&errorResult).Get(uidServerURL); err != nil {
				return nil, ErrorUIDServerConnect
			}
			if errorResult.Error != "" {
				return nil, ErrorUIDServerNotFound
			}
			userAttribute.UID = ptr.To(result.UID)
			userAttribute.GID = ptr.To(result.GID)
		case UIDSourceLDAP:
			// Already in userAttribute from attrFromLDAP
		case UIDSourceNone, UIDSourceDefault, "":
			userAttribute.UID = ptr.To("1001")
			userAttribute.GID = ptr.To("1001")
		}
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
	if userAttribute.Nickname != "" {
		user.Nickname = userAttribute.Nickname
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
	}

	if err := uq.WithContext(c).Create(&userAccount); err != nil {
		return nil, err
	}

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

func (mgr *AuthMgr) actLDAPAuth(_ context.Context, username, password string, attr *model.UserAttribute) error {
	conf := config.GetConfig().Auth.LDAP
	// LDAP connection
	l, err := ldap.DialURL(conf.Server.Address)
	if err != nil {
		return fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer l.Close()

	// Admin bind
	err = l.Bind(conf.Server.BindDN, conf.Server.BindPassword)
	if err != nil {
		return fmt.Errorf("LDAP admin bind failed: %w", err)
	}

	attributes := mgr.prepareLDAPAttributes()

	// Search for user
	mapping := conf.AttributeMapping
	filter := fmt.Sprintf("(%s=%s)", mapping.Username, username)
	searchRequest := ldap.NewSearchRequest(
		conf.Server.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		attributes,
		nil,
	)

	searchResult, err := l.Search(searchRequest)
	if err != nil {
		return fmt.Errorf("LDAP search failed: %w", err)
	}

	if len(searchResult.Entries) != 1 {
		return ErrorLdapUserNotFound
	}

	entry := searchResult.Entries[0]
	userDN := entry.DN

	// User bind for password verification
	err = l.Bind(userDN, password)
	if err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			return ErrorInvalidCredentials
		}
		return fmt.Errorf("LDAP user bind failed: %w", err)
	}

	// Populate attributes
	mgr.populateUserAttributes(username, entry, attr)

	uidConf := config.GetConfig().Auth.LDAP.UID
	if UIDSource(uidConf.Source) == UIDSourceLDAP {
		attr.UID = ptr.To(entry.GetAttributeValue(uidConf.LDAPAttribute.UID))
		attr.GID = ptr.To(entry.GetAttributeValue(uidConf.LDAPAttribute.GID))
	}

	return nil
}

func (mgr *AuthMgr) prepareLDAPAttributes() []string {
	conf := config.GetConfig().Auth.LDAP
	attributes := []string{"dn"}
	mapping := conf.AttributeMapping
	if mapping.Username != "" {
		attributes = append(attributes, mapping.Username)
	}
	if mapping.DisplayName != "" {
		attributes = append(attributes, mapping.DisplayName)
	}
	if mapping.Email != "" {
		attributes = append(attributes, mapping.Email)
	}
	if mapping.Teacher != "" {
		attributes = append(attributes, mapping.Teacher)
	}
	if mapping.Group != "" {
		attributes = append(attributes, mapping.Group)
	}
	if mapping.Phone != "" {
		attributes = append(attributes, mapping.Phone)
	}
	if mapping.ExpiredAt != "" {
		attributes = append(attributes, mapping.ExpiredAt)
	}

	// Fetch UID/GID if from LDAP
	uidConf := config.GetConfig().Auth.LDAP.UID
	if UIDSource(uidConf.Source) == UIDSourceLDAP {
		if uidConf.LDAPAttribute.UID != "" {
			attributes = append(attributes, uidConf.LDAPAttribute.UID)
		}
		if uidConf.LDAPAttribute.GID != "" {
			attributes = append(attributes, uidConf.LDAPAttribute.GID)
		}
	}
	return attributes
}

func (mgr *AuthMgr) populateUserAttributes(username string, entry *ldap.Entry, attr *model.UserAttribute) {
	conf := config.GetConfig().Auth.LDAP
	mapping := conf.AttributeMapping

	attr.Name = username
	if mapping.DisplayName != "" {
		attr.Nickname = entry.GetAttributeValue(mapping.DisplayName)
	}
	if mapping.Email != "" {
		attr.Email = ptr.To(entry.GetAttributeValue(mapping.Email))
	}
	if mapping.Teacher != "" {
		attr.Teacher = ptr.To(entry.GetAttributeValue(mapping.Teacher))
	}
	if mapping.Group != "" {
		attr.Group = ptr.To(entry.GetAttributeValue(mapping.Group))
	}
	if mapping.Phone != "" {
		attr.Phone = ptr.To(entry.GetAttributeValue(mapping.Phone))
	}
	if mapping.ExpiredAt != "" {
		val := entry.GetAttributeValue(mapping.ExpiredAt)
		if val != "" && val != "0" && val != "9223372036854775807" {
			attr.ExpiredAt = convertFileTimeToRFC3339(val)
		}
	}
}

func convertFileTimeToRFC3339(val string) *string {
	// Correct FILETIME conversion:
	// FILETIME is 100-nanosecond intervals since January 1, 1601 (UTC).
	// Go time is since January 1, 1970 (UTC).
	// Offset is 11644473600 seconds.
	var ticks int64
	if _, err := fmt.Sscanf(val, "%d", &ticks); err != nil {
		klog.V(LogLevelDebug).Infof("Failed to parse FILETIME %s: %v", val, err)
		return nil
	}
	if ticks > 0 {
		sec := (ticks / 10000000) - 11644473600
		nsec := (ticks % 10000000) * 100
		t := time.Unix(sec, nsec).UTC()
		return ptr.To(t.Format(time.RFC3339))
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

	conf := config.GetConfig().Auth
	if !conf.Normal.AllowRegister {
		resputil.HTTPError(c, http.StatusForbidden, "Direct registration is disabled by administrator", resputil.InvalidRequest)
		return
	}

	u := query.User
	_, err := u.WithContext(c).Where(u.Name.Eq(req.Username)).First()
	if err == nil {
		resputil.HTTPError(c, http.StatusConflict, "User already exists", resputil.InvalidRequest)
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}

	user, err := mgr.createUser(c, req.Username, &req.Password, nil)
	if err != nil {
		mgr.handleCreateUserError(c, err)
		return
	}

	// Login logic after signup
	q := query.Account
	uq := query.UserAccount

	lastUserQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID)).Last()
	if err != nil {
		resputil.HTTPError(c, http.StatusForbidden, "User must has at least one queue", resputil.UserNotAllowed)
		return
	}

	lastQueue, err := q.WithContext(c).Where(q.ID.Eq(lastUserQueue.AccountID)).First()
	if err != nil {
		resputil.HTTPError(c, http.StatusForbidden, "User must has at least one queue", resputil.UserNotAllowed)
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
	// Ensure ID and Name are populated in the response attributes
	respAttr := user.Attributes.Data()
	respAttr.ID = user.ID
	respAttr.Name = user.Name

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
		User: respAttr,
	}
	resputil.Success(c, loginResponse)
}

func (mgr *AuthMgr) handleCreateUserError(c *gin.Context, err error) {
	if errors.Is(err, ErrorUIDServerConnect) {
		klog.Error("can't connect to UID server")
		resputil.HTTPError(c, http.StatusBadGateway, "Can't connect to UID server", resputil.UidServiceError)
	} else if errors.Is(err, ErrorUIDServerNotFound) {
		klog.Error("UID not found")
		resputil.HTTPError(c, http.StatusNotFound, "UID not found", resputil.UidNotFound)
	} else {
		klog.Error("create new user", err)
		resputil.HTTPError(c, http.StatusInternalServerError, "Create user failed", resputil.NotSpecified)
	}
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
	// Fetch user to populate User and Space
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, "User not found", resputil.NotSpecified)
		return
	}

	// Ensure ID and Name are populated in the response attributes
	respAttr := user.Attributes.Data()
	respAttr.ID = user.ID
	respAttr.Name = user.Name

	loginResponse := LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Context: AccountContext{
			Queue:        req.Queue,
			RoleQueue:    userQueue.Role,
			AccessQueue:  userQueue.AccessMode,
			RolePlatform: token.RolePlatform,
			AccessPublic: token.PublicAccessMode,
			Space:        user.Space,
		},
		User: respAttr,
	}
	resputil.Success(c, loginResponse)
}
