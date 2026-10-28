package handler

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
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
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewAuthMgr)
}

type AuthMgr struct {
	name           string
	client         *http.Client
	req            *imrocreq.Client
	tokenMgr       *util.TokenManager
	billingService *service.BillingService
	userBanService *service.UserBanService
}

func NewAuthMgr(conf *RegisterConfig) Manager {
	return &AuthMgr{
		name:           "auth",
		client:         &http.Client{},
		req:            imrocreq.C(),
		tokenMgr:       util.GetTokenMgr(),
		billingService: conf.BillingService,
		userBanService: conf.UserBanService,
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
		Version      VersionInfo         `json:"version"`
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
		AppVersion string `json:"appVersion"` // Release version or development build metadata
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
	EnableLDAP           bool   `json:"enableLdap"`
	LDAPAlias            string `json:"ldapAlias,omitempty"`
	LDAPHelp             string `json:"ldapHelp,omitempty"`
	EnableNormalLogin    bool   `json:"enableNormalLogin"`
	EnableNormalRegister bool   `json:"enableNormalRegister"`
}

type AuthMethod string

const (
	AuthMethodNormal AuthMethod = "normal"
	AuthMethodLDAP   AuthMethod = "ldap"
)

type UIDSource string

const (
	UIDSourceNone     UIDSource = UIDSource(config.UIDSourceNone)
	UIDSourceLDAP     UIDSource = UIDSource(config.UIDSourceLDAP)
	UIDSourceRID      UIDSource = UIDSource(config.UIDSourceRID)
	UIDSourceExternal UIDSource = UIDSource(config.UIDSourceExternal)
	UIDSourceDefault  UIDSource = UIDSource(config.UIDSourceDefault)
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
		LDAPAlias:            conf.LDAP.Alias,
		LDAPHelp:             conf.LDAP.Help,
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
	header := c.GetHeader("Authorization")
	if header == "" {
		resputil.Success(c, nil)
		return
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("Provide a valid bearer token or sign in again."))
		return
	}
	claims, err := mgr.tokenMgr.CheckToken(parts[1])
	if err != nil {
		resputil.HandleError(c, util.TokenValidationError(err))
		return
	}
	user, err := mgr.loadSessionUser(c, claims.UserID)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	accountContext, _, err := loadLoginContext(c, user, claims.AccountID)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	attributes := user.Attributes.Data()
	attributes.ID, attributes.Name = user.ID, user.Name
	resputil.Success(c, CheckResp{Context: accountContext, User: attributes, Version: GetVersionInfo()})
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
			return false, bizerr.Forbidden.LoginMethodDisabled.New("LDAP login is disabled. Use an enabled login method or contact the " +
				"platform " +
				"administrator.")
		}
		if err := mgr.actLDAPAuth(c, username, password, attr); err != nil {
			return false, err
		}
		return true, nil
	case AuthMethodNormal:
		if !conf.Normal.AllowLogin {
			return false, bizerr.Forbidden.LoginMethodDisabled.New("Local login is disabled. Use LDAP login or contact the platform " +
				"administrator.")
		}
		return false, mgr.normalAuth(c, username, password)
	default:
		return false, bizerr.BadRequest.ParameterError.New("Authentication method must be normal or ldap.")
	}
}

func (mgr *AuthMgr) handleLoginError(c *gin.Context, err error) {
	if _, ok := bizerr.FromError(err); !ok {
		err = bizerr.Internal.AccountInitializationFailed.Wrap(err, "Platform account initialization could not be completed. Retry or "+
			"contact the "+
			"platform administrator.")
	}
	resputil.HandleError(c, err)
}

func (mgr *AuthMgr) requireLoginUser(c context.Context, user *model.User) error {
	if err := util.ValidatePlatformUser(user); err != nil {
		return err
	}
	if mgr.userBanService != nil {
		return mgr.userBanService.RequireCapability(c, user.ID, service.UserBanCapabilityPlatformAccess)
	}
	return nil
}

func (mgr *AuthMgr) loadSessionUser(c context.Context, userID uint) (*model.User, error) {
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(userID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerr.Auth.TokenInvalid.Wrap(err, "This account no longer exists. Sign in again or contact the platform administrator.")
	}
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "Account lookup is unavailable. Retry or contact the platform administrator.")
	}
	if err := mgr.requireLoginUser(c, user); err != nil {
		return nil, err
	}
	return user, nil
}

// A zero project ID selects the latest association for a fresh login. Session
// checks, refreshes, and switches always supply their explicit project ID.
func loadLoginContext(c context.Context, user *model.User, projectID uint) (AccountContext, util.JWTMessage, error) {
	var accountContext AccountContext
	var claims util.JWTMessage
	uq, q := query.UserAccount, query.Account
	associations := uq.WithContext(c).Where(uq.UserID.Eq(user.ID))
	if projectID != 0 {
		associations = associations.Where(uq.AccountID.Eq(projectID))
	} else {
		associations = associations.Where(uq.AccessMode.Neq(uint8(model.AccessModeNA)))
	}
	association, err := associations.Last()
	if err != nil {
		return accountContext, claims, loginQueueError(err)
	}
	if association.AccessMode == model.AccessModeNA {
		return accountContext, claims, bizerr.Forbidden.NoProjectAccess.New("This account cannot access the selected project. Contact " +
			"the platform " +
			"administrator.")
	}
	project, err := q.WithContext(c).Where(q.ID.Eq(association.AccountID)).First()
	if err != nil {
		return accountContext, claims, loginQueueError(err)
	}
	publicAccess := model.AccessModeNA
	if association.AccountID == model.DefaultAccountID {
		publicAccess = association.AccessMode
	} else {
		publicAssociation, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID), uq.AccountID.Eq(model.DefaultAccountID)).First()
		if err == nil {
			publicAccess = publicAssociation.AccessMode
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return accountContext, claims, bizerr.Internal.DatabaseError.Wrap(err, "Public project permissions could not be loaded. "+
				"Retry or contact the platform "+
				"administrator.")
		}
	}
	accountContext = AccountContext{Queue: project.Name, RoleQueue: association.Role, RolePlatform: user.Role,
		AccessQueue: association.AccessMode, AccessPublic: publicAccess, Space: user.Space}
	claims = util.JWTMessage{UserID: user.ID, Username: user.Name, AccountID: project.ID, AccountName: project.Name,
		RoleAccount: association.Role, RolePlatform: user.Role, AccountAccessMode: association.AccessMode, PublicAccessMode: publicAccess}
	return accountContext, claims, nil
}

func (mgr *AuthMgr) respondLogin(c *gin.Context, user *model.User, projectID uint) {
	accountContext, claims, err := loadLoginContext(c, user, projectID)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	access, refresh, err := mgr.tokenMgr.CreateTokens(&claims)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, "Session creation could not be completed. Retry or contact the "+
			"platform "+
			"administrator."))
		return
	}
	attributes := user.Attributes.Data()
	attributes.ID, attributes.Name = user.ID, user.Name
	resputil.Success(c, LoginResp{
		AccessToken: access, RefreshToken: refresh, Context: accountContext,
		User: attributes, Version: GetVersionInfo(),
	})
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
//	@Failure		401		{object}	resputil.Response[any]			"账号凭证错误或账号未激活"
//	@Failure		403		{object}	resputil.Response[any]			"登录方式禁用、平台账号受限或缺少项目关联"
//	@Failure		409		{object}	resputil.Response[any]			"账号创建冲突"
//	@Failure		500		{object}	resputil.Response[any]			"目录服务、身份映射或数据库异常"
//	@Failure		502		{object}	resputil.Response[any]			"外部 UID 服务不可用或响应不完整"
//	@Failure		504		{object}	resputil.Response[any]			"LDAP 认证超时"
//	@Router			/auth/login [post]
func (mgr *AuthMgr) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "Provide a valid authentication request with all required "+
			"fields."))
		return
	}

	if req.Token != nil && *req.Token != "" {
		msg := "Legacy token login is no longer supported, just use your LDAP account"
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New(msg))
		return
	}

	if req.Username == nil || req.Password == nil {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("Provide both username and password."))
		return
	}
	username := *req.Username
	password := *req.Password

	if strings.TrimSpace(username) == "" || password == "" {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("Username and password must not be empty."))
		return
	}

	var attributes model.UserAttribute
	allowRegister, err := mgr.performAuthentication(c, req.AuthMethod, username, password, &attributes)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	// Check if the user exists, and should create user or return error
	user, err := mgr.getOrCreateUser(c, &req, &attributes, allowRegister)
	if err != nil {
		mgr.handleLoginError(c, err)
		return
	}

	if err := mgr.requireLoginUser(c, user); err != nil {
		resputil.HandleError(c, err)
		return
	}
	if err = mgr.updateUserIfNeeded(c, user, &attributes); err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "User profile synchronization could not be completed. Retry "+
			"or contact the "+
			"platform administrator."))
		return
	}
	mgr.respondLogin(c, user, 0)
}

func loginQueueError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerr.Forbidden.NoProjectAccess.Wrap(err, "No accessible project is associated with this account. Contact the platform "+
			"administrator.")
	}
	return bizerr.Internal.DatabaseError.Wrap(err, "Project lookup is unavailable. Retry or contact the platform administrator.")
}

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
	if allowCreate && attr.Nickname == "" && req.Username != nil {
		attr.Nickname = *req.Username
	}

	u := query.User
	user, err := u.WithContext(c).Where(u.Name.Eq(attr.Name)).First()
	if err == nil {
		return user, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "Account lookup is unavailable. Retry or contact the platform administrator.")
	}

	// User not found in the database
	if allowCreate {
		// User exists in the auth method but not in the database, create a new user
		return mgr.createUser(c, attr.Name, nil, attr)
	}

	return nil, bizerr.Auth.MustRegister.New("Register a local account before signing in.")
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

	syncPtrField(&currentAttr.UID, newAttr.UID, &changed)
	syncPtrField(&currentAttr.GID, newAttr.GID, &changed)

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

func (mgr *AuthMgr) initialUserAttributes(
	c context.Context, name string, attrFromLDAP *model.UserAttribute,
) (model.UserAttribute, error) {
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
			// Deprecated: Internal ACT UID server (Winbind wrapper).
			// Keeping this for backward compatibility as per request.
			klog.Warningf("using deprecated UID source 'external' for user %s; "+
				"please consider switching to 'rid' for direct LDAP SID calculation", name)
			result, err := mgr.fetchExternalUID(c, name, uidConf.ExternalService.URL)
			if err != nil {
				return model.UserAttribute{}, err
			}
			userAttribute.UID = ptr.To(result.UID)
			userAttribute.GID = ptr.To(result.GID)
		case UIDSourceLDAP, UIDSourceRID:
			// Already in userAttribute from attrFromLDAP (calculated in actLDAPAuth for RID mode)
		case UIDSourceDefault, UIDSourceNone, "":
			userAttribute.UID = ptr.To("1001")
			userAttribute.GID = ptr.To("1001")
		}
	}

	return userAttribute, nil
}

// createUser is called when the user is not found in the database
func (mgr *AuthMgr) createUser(c context.Context, name string, password *string, attrFromLDAP *model.UserAttribute) (*model.User, error) {
	if invalidPlatformSpaceName(name) {
		return nil, bizerr.Internal.IdentityMappingFailed.New("The account name cannot be mapped to a platform space. Contact the " +
			"platform " +
			"administrator.")
	}
	userAttribute, err := mgr.initialUserAttributes(c, name, attrFromLDAP)
	if err != nil {
		return nil, err
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

	billingEnabled := mgr.billingService != nil && mgr.billingService.IsFeatureEnabled(c)
	err = query.Q.Transaction(func(tx *query.Query) error {
		if _, err := tx.Account.WithContext(c).Where(tx.Account.ID.Eq(model.DefaultAccountID)).First(); err != nil {
			return bizerr.Internal.AccountInitializationFailed.Wrap(err, "The default project is unavailable for account creation. "+
				"Contact the platform "+
				"administrator.")
		}
		if err := tx.User.WithContext(c).Create(&user); err != nil {
			return err
		}

		// Add the default user account and initialize its billing balance atomically.
		userAccount := model.UserAccount{
			UserID:     user.ID,
			AccountID:  model.DefaultAccountID,
			Role:       model.RoleUser,
			AccessMode: model.AccessModeRO,
		}
		if err := tx.UserAccount.WithContext(c).Create(&userAccount); err != nil {
			return err
		}

		if billingEnabled {
			return mgr.billingService.IssueUserAccountNowInTransaction(
				c,
				tx,
				user.ID,
				model.DefaultAccountID,
			)
		}
		return nil
	})
	if err != nil {
		return nil, accountCreationError(err)
	}

	return &user, nil
}

func accountCreationError(err error) error {
	var sqlStateError interface{ SQLState() string }
	if errors.Is(err, gorm.ErrDuplicatedKey) ||
		(errors.As(err, &sqlStateError) && sqlStateError.SQLState() == "23505") {
		return bizerr.Conflict.ResourceAlreadyExists.Wrap(err, "This username is already registered. Sign in or retry login.")
	}
	if _, ok := bizerr.FromError(err); ok {
		return err
	}
	return bizerr.Internal.AccountInitializationFailed.Wrap(err, "Platform account initialization could not be completed. Retry or "+
		"contact the "+
		"platform administrator.")
}

func (mgr *AuthMgr) normalAuth(c *gin.Context, username, password string) error {
	// Local registration uses bcrypt, which supports at most 72 password bytes.
	// Reject longer input rather than accepting credentials with a truncated suffix.
	if len(password) > maxLocalPasswordBytes {
		return bizerr.BadRequest.ParameterError.New("Local passwords must not exceed 72 bytes.")
	}
	u := query.User
	user, err := u.WithContext(c).Where(u.Name.Eq(username)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerr.Auth.RegisterNotFound.New("No matching local account was found. Check the username or register an account.")
	}
	if err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "Account lookup is unavailable. Retry or contact the platform administrator.")
	}
	if user.Password == nil {
		return bizerr.Auth.PasswordLoginUnavailable.New("This account has no local password. Use its original login method or contact " +
			"the platform administrator.")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return bizerr.Auth.InvalidCredentials.Wrap(err, "Username or password is incorrect.")
		}
		return bizerr.Internal.ServiceError.Wrap(err, "The stored account credentials cannot be verified. Contact the platform "+
			"administrator.")
	}
	return nil
}

const maxLocalPasswordBytes = 72

const authenticationTimeout = 8 * time.Second

// ldapAuthenticator keeps password verification and identity mapping testable without a live directory.
type ldapAuthenticator interface {
	Bind(username, password string) error
	Search(request *ldap.SearchRequest) (*ldap.SearchResult, error)
}

func (mgr *AuthMgr) actLDAPAuth(ctx context.Context, username, password string, attr *model.UserAttribute) error {
	conf := config.GetConfig()
	ctx, cancel := context.WithTimeout(ctx, authenticationTimeout)
	defer cancel()
	l, err := ldap.DialURL(conf.Auth.LDAP.Server.Address,
		ldap.DialWithDialer(&net.Dialer{Timeout: authenticationTimeout}))
	if err != nil {
		return ldapServiceError(err, "The LDAP server cannot be reached. Retry or contact the platform administrator.")
	}
	defer l.Close()
	l.SetTimeout(authenticationTimeout)
	stop := context.AfterFunc(ctx, func() { _ = l.Close() })
	defer stop()
	err = mgr.authenticateLDAP(l, conf, username, password, attr)
	if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return bizerr.GatewayTimeout.GatewayTimeout.Wrap(err, "LDAP login timed out. Retry or contact the platform administrator.")
	}
	return err
}

func ldapServiceError(err error, message string) error {
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkError) && networkError.Timeout()) {
		return bizerr.GatewayTimeout.GatewayTimeout.Wrap(err, "LDAP login timed out. Retry or contact the platform administrator.")
	}
	return bizerr.Internal.ThirdPartyApiError.Wrap(err, message)
}

func (mgr *AuthMgr) authenticateLDAP(l ldapAuthenticator, cfg *config.Config, username, password string, attr *model.UserAttribute) error {
	conf := cfg.Auth.LDAP
	if err := l.Bind(conf.Server.BindDN, conf.Server.BindPassword); err != nil {
		return ldapServiceError(err, "The LDAP service account could not authenticate. Contact the platform "+
			"administrator to check the directory configuration.")
	}
	filter := fmt.Sprintf("(%s=%s)", conf.AttributeMapping.Username, ldap.EscapeFilter(username))
	searchRequest := ldap.NewSearchRequest(conf.Server.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter, mgr.prepareLDAPAttributes(cfg), nil)
	searchResult, err := l.Search(searchRequest)
	if err != nil {
		return ldapServiceError(err, "The LDAP directory could not be searched. Retry or contact the platform administrator.")
	}
	if len(searchResult.Entries) == 0 {
		return bizerr.Auth.RegisterNotFound.New("No matching LDAP account was found. Check the username or contact the directory " +
			"administrator.")
	}
	if len(searchResult.Entries) != 1 {
		return bizerr.Internal.ThirdPartyApiError.New("The LDAP username matched multiple directory entries. Contact the platform " +
			"administrator to check the directory configuration.")
	}
	entry := searchResult.Entries[0]
	if err := l.Bind(entry.DN, password); err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			return bizerr.Auth.InvalidCredentials.Wrap(err, "LDAP username or password is incorrect, or the directory rejected the account. "+
				"Check your credentials or contact the directory administrator.")
		}
		return ldapServiceError(err, "The LDAP server could not verify this account. Retry or contact the directory administrator.")
	}
	mgr.populateUserAttributes(cfg, username, entry, attr)
	uidConf := conf.UID
	switch UIDSource(uidConf.Source) {
	case UIDSourceLDAP:
		uid := entry.GetAttributeValue(uidConf.LDAPAttribute.UID)
		gid := entry.GetAttributeValue(uidConf.LDAPAttribute.GID)
		if !validUnixID(uid) || !validUnixID(gid) {
			return bizerr.Internal.IdentityMappingFailed.New("LDAP authentication succeeded, but UID or GID attributes are missing or " +
				"invalid. Contact the platform administrator.")
		}
		attr.UID, attr.GID = ptr.To(uid), ptr.To(gid)
	case UIDSourceRID:
		uid, err := calculateUIDFromRID(entry, uidConf.RID.SIDAttribute, uidConf.RID.Offset, username)
		if err != nil {
			return bizerr.Internal.IdentityMappingFailed.Wrap(err, "LDAP authentication succeeded, but the user SID could not be mapped "+
				"to a UID. "+
				"Contact the platform administrator.")
		}
		gid, err := calculateGIDFromPrimaryGroup(entry, uidConf.RID.PGIDAttribute, uidConf.RID.Offset, username)
		if err != nil {
			return bizerr.Internal.IdentityMappingFailed.Wrap(err, "LDAP authentication succeeded, but the primary group could not be "+
				"mapped to a "+
				"GID. Contact the platform administrator.")
		}
		attr.UID, attr.GID = ptr.To(strconv.Itoa(uid)), ptr.To(strconv.Itoa(gid))
	case UIDSourceDefault, UIDSourceNone, UIDSourceExternal, "":
	default:
		return bizerr.Internal.IdentityMappingFailed.New("The configured identity source is unsupported. Contact the platform " +
			"administrator.")
	}
	return nil
}

func (mgr *AuthMgr) fetchExternalUID(ctx context.Context, username, url string) (*ActUIDServerSuccessResp, error) {
	ctx, cancel := context.WithTimeout(ctx, authenticationTimeout)
	defer cancel()
	var result ActUIDServerSuccessResp
	var errorResult ActUIDServerErrorResp
	response, err := mgr.req.R().SetContext(ctx).
		SetQueryParam("username", username).SetSuccessResult(&result).
		SetErrorResult(&errorResult).Get(url)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, bizerr.GatewayTimeout.GatewayTimeout.Wrap(err, "The UID service timed out during account creation. Retry or "+
				"contact the "+
				"platform administrator.")
		}
		return nil, bizerr.BadGateway.BadGateway.Wrap(err, "LDAP authentication succeeded, but the UID service could not provide an "+
			"identity for account creation. Retry or contact the platform administrator.")
	}
	if !response.IsSuccessState() || errorResult.Error != "" || !validUnixID(result.UID) || !validUnixID(result.GID) {
		return nil, bizerr.BadGateway.BadGateway.New("LDAP authentication succeeded, but the UID service did not return a usable UID " +
			"and GID for account creation. Contact the platform administrator.")
	}
	return &result, nil
}

func validUnixID(value string) bool {
	if value == "" {
		return false
	}
	_, err := strconv.ParseUint(value, 10, 32)
	return err == nil
}

const minSIDLengthBytes = 12

// calculateUIDFromRID computes UID from objectSid using RID + offset.
func calculateUIDFromRID(entry *ldap.Entry, sidAttr string, offset int, username string) (int, error) {
	if sidAttr == "" {
		sidAttr = "objectSid"
	}

	sidBytes := entry.GetRawAttributeValue(sidAttr)
	if len(sidBytes) == 0 {
		return 0, fmt.Errorf("SID attribute %s not found for user %s in RID mode", sidAttr, username)
	}

	rid, err := parseRIDFromSID(sidBytes)
	if err != nil {
		return 0, fmt.Errorf("failed to parse RID from SID for user %s: %w", username, err)
	}

	return mappedUnixID(uint64(rid), offset)
}

// calculateGIDFromPrimaryGroup computes GID from primaryGroupID using RID + offset.
func calculateGIDFromPrimaryGroup(entry *ldap.Entry, pgidAttr string, offset int, username string) (int, error) {
	if pgidAttr == "" {
		pgidAttr = "primaryGroupID"
	}

	pgidStr := entry.GetAttributeValue(pgidAttr)
	if pgidStr == "" {
		return 0, fmt.Errorf("primary group attribute %s not found for user %s in RID mode", pgidAttr, username)
	}

	pgid, err := strconv.ParseUint(pgidStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse primaryGroupID %s for user %s: %w", pgidStr, username, err)
	}

	return mappedUnixID(pgid, offset)
}

func mappedUnixID(rid uint64, offset int) (int, error) {
	if rid > uint64(^uint32(0)) {
		return 0, bizerr.Internal.IdentityMappingFailed.New("RID is outside the Unix identity range")
	}
	value := int64(rid) + int64(offset)
	if value < 0 || value > int64(^uint32(0)) {
		return 0, bizerr.Internal.IdentityMappingFailed.New("RID plus offset is outside the Unix identity range")
	}
	return int(value), nil
}

// parseRIDFromSID extracts the RID (Relative Identifier) from a binary Windows SID.
// A Windows SID binary format (MS-DTYP 2.4.2.2) consists of:
// - Revision (1 byte)
// - SubAuthorityCount (1 byte)
// - IdentifierAuthority (6 bytes)
// - SubAuthorities (SubAuthorityCount * 4 bytes)
// The RID is the last 4-byte sub-authority in little-endian.
func parseRIDFromSID(sidBytes []byte) (uint32, error) {
	if len(sidBytes) < minSIDLengthBytes {
		return 0, fmt.Errorf("invalid SID length (must be at least %d bytes): %d", minSIDLengthBytes, len(sidBytes))
	}

	if sidBytes[0] != 1 {
		return 0, bizerr.Internal.IdentityMappingFailed.New("The directory returned an unsupported SID revision. Contact the platform " +
			"administrator.")
	}

	// Verify the length matches the SubAuthorityCount
	n := int(sidBytes[1])
	expectedLen := 8 + (n * 4)
	if len(sidBytes) != expectedLen {
		return 0, fmt.Errorf("invalid SID binary format: length %d does not match sub-authority count %d (expected %d)",
			len(sidBytes), n, expectedLen)
	}

	if n < 1 {
		return 0, fmt.Errorf("invalid SID binary format: sub-authority count is zero")
	}

	// The RID is the last 4 bytes of the SID, stored in little-endian.
	pos := len(sidBytes) - 4
	return binary.LittleEndian.Uint32(sidBytes[pos:]), nil
}

func (mgr *AuthMgr) prepareLDAPAttributes(cfg *config.Config) []string {
	conf := cfg.Auth.LDAP
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
	uidConf := conf.UID
	switch UIDSource(uidConf.Source) {
	case UIDSourceLDAP:
		if uidConf.LDAPAttribute.UID != "" {
			attributes = append(attributes, uidConf.LDAPAttribute.UID)
		}
		if uidConf.LDAPAttribute.GID != "" {
			attributes = append(attributes, uidConf.LDAPAttribute.GID)
		}
	case UIDSourceRID:
		sidAttr := uidConf.RID.SIDAttribute
		if sidAttr == "" {
			sidAttr = "objectSid"
		}
		pgidAttr := uidConf.RID.PGIDAttribute
		if pgidAttr == "" {
			pgidAttr = "primaryGroupID"
		}
		attributes = append(attributes, sidAttr, pgidAttr)
	}
	return attributes
}

func (mgr *AuthMgr) populateUserAttributes(cfg *config.Config, username string, entry *ldap.Entry, attr *model.UserAttribute) {
	conf := cfg.Auth.LDAP
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
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "Provide a valid authentication request with all required "+
			"fields."))
		return
	}

	if ok, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`, req.Username); !ok {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("Local usernames must contain 1 to 64 ASCII letters, digits, "+
			"dots, underscores "+
			"or hyphens, starting with a letter or digit."))
		return
	}
	if len(req.Password) > maxLocalPasswordBytes {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("Local passwords must not exceed 72 bytes."))
		return
	}

	conf := config.GetConfig().Auth
	if !conf.Normal.AllowRegister {
		resputil.HandleError(c, bizerr.Forbidden.RegistrationDisabled.New("Local registration is disabled. Use LDAP login or contact "+
			"the platform "+
			"administrator."))
		return
	}

	u := query.User
	_, err := u.WithContext(c).Where(u.Name.Eq(req.Username)).First()
	if err == nil {
		resputil.HandleError(c, bizerr.Conflict.ResourceAlreadyExists.New("This username is already registered. Sign in or choose another "+
			"username."))
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "Account lookup is unavailable. Retry or contact the platform "+
			"administrator."))
		return
	}

	user, err := mgr.createUser(c, req.Username, &req.Password, nil)
	if err != nil {
		mgr.handleLoginError(c, err)
		return
	}

	mgr.respondLogin(c, user, 0)
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
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "Provide a refresh token in a valid authentication request."))
		return
	}
	claims, err := mgr.tokenMgr.CheckRefreshToken(request.RefreshToken)
	if err != nil {
		resputil.HandleError(c, util.TokenValidationError(err))
		return
	}
	user, err := mgr.loadSessionUser(c, claims.UserID)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	_, current, err := loadLoginContext(c, user, claims.AccountID)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	access, refresh, err := mgr.tokenMgr.CreateTokens(&current)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, "Session renewal could not be completed. Retry or sign in again."))
		return
	}
	resputil.Success(c, RefreshResp{AccessToken: access, RefreshToken: refresh})
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
	var request SwitchQueueReq
	if err := c.ShouldBind(&request); err != nil || strings.TrimSpace(request.Queue) == "" {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("Provide the name of a project to switch to."))
		return
	}
	user, err := mgr.loadSessionUser(c, util.GetToken(c).UserID)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	q := query.Account
	project, err := q.WithContext(c).Where(q.Name.Eq(request.Queue)).First()
	if err != nil {
		resputil.HandleError(c, loginQueueError(err))
		return
	}
	mgr.respondLogin(c, user, project.ID)
}

func invalidPlatformSpaceName(name string) bool {
	const parentDirectory = ".."
	return name == "" || len(name) > 64 || name == "." ||
		name == parentDirectory || strings.ContainsAny(name, "/\\\x00\r\n")
}
