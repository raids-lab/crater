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
	"time"

	"github.com/gin-gonic/gin"
	ldap "github.com/go-ldap/ldap/v3"
	imrocreq "github.com/imroc/req/v3"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/samber/lo"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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

func NewAuthMgr(_ RegisterConfig) Manager {
	return &AuthMgr{
		name:     "auth",
		client:   &http.Client{},
		req:      imrocreq.C().DevMode(),
		tokenMgr: util.GetTokenMgr(),
		openAPI:  config.GetConfig().ACT.OpenAPI,
	}
}

func (mgr *AuthMgr) GetName() string { return mgr.name }

func (mgr *AuthMgr) RegisterPublic(g *gin.RouterGroup) {
	g.POST("login", mgr.Login)
	g.POST("signup", mgr.Signup)
	g.POST("refresh", mgr.RefreshToken)
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

	AccountContext struct {
		Queue        string           `json:"queue"`        // Current Queue Name
		RoleQueue    model.Role       `json:"roleQueue"`    // User role of the queue
		RolePlatform model.Role       `json:"rolePlatform"` // User role of the platform
		AccessQueue  model.AccessMode `json:"accessQueue"`  // User access mode of the queue
		AccessPublic model.AccessMode `json:"accessPublic"` // User access mode of the platform
		Space        string           `json:"space"`        // User pvc subpath the platform
	}
)

type AuthMethod string

const (
	AuthMethodNormal  AuthMethod = "normal"
	AuthMethodACTLDAP AuthMethod = "act-ldap"
	AuthMethodACTAPI  AuthMethod = "act-api"
)

// Login godoc
// @Summary 用户登录
// @Description 校验用户身份，生成包含当前用户和项目的 JWT Token
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body LoginReq false "查询参数"
// @Success 200 {object} resputil.Response[LoginResp] "登录成功，返回 JWT Token 和默认个人项目"
// @Failure 400 {object} resputil.Response[any]	"请求参数错误"
// @Failure 401 {object} resputil.Response[any]	"用户名或密码错误"
// @Failure 500 {object} resputil.Response[any]	"数据库交互错误"
// @Router /auth/login [post]
//
//nolint:gocyclo // TODO(liyilong): refactor this
func (mgr *AuthMgr) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	l := logutils.Log.WithFields(logutils.Fields{
		"username": req.Username,
		"auth":     req.AuthMethod,
	})

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
		// Username can only contain lowercase letters, numbers
		if !regexp.MustCompile(`^[a-z0-9]+$`).MatchString(username) {
			l.Error("invalid username")
			resputil.HTTPError(c, http.StatusBadRequest, "Invalid username", resputil.InvalidRequest)
			return
		}
	default:
		resputil.HTTPError(c, http.StatusBadRequest, "Invalid auth method", resputil.InvalidRequest)
		return
	}

	// Check if request auth method is valid
	var attributes model.UserAttribute
	switch req.AuthMethod {
	case AuthMethodACTAPI:
		if err := mgr.actAPIAuth(c, token, &attributes); err != nil {
			l.Error("invalid token: ", err)
			resputil.HTTPError(c, http.StatusUnauthorized, "Invalid token", resputil.NotSpecified)
			return
		}
	case AuthMethodACTLDAP:
		if err := mgr.actLDAPAuth(c, username, password); err != nil {
			l.Error("invalid credentials: ", err)
			resputil.HTTPError(c, http.StatusUnauthorized, "Invalid credentials", resputil.NotSpecified)
			return
		}
	case AuthMethodNormal:
		if err := mgr.normalAuth(c, username, password); err != nil {
			l.Error("invalid credentials: ", err)
			resputil.HTTPError(c, http.StatusUnauthorized, "Invalid credentials", resputil.NotSpecified)
			return
		}
	default:
		l.Error("invalid auth method: ", req.AuthMethod)
		resputil.HTTPError(c, http.StatusBadRequest, "Invalid auth method", resputil.InvalidRequest)
		return
	}

	// Check if the user exists
	user, err := shouldCreateOrUpdateUser(c, &req, &attributes)
	if err != nil {
		l.Error("create or update user", err)
		resputil.Error(c, "Create or update user failed", resputil.NotSpecified)
		return
	}

	if user.Status != model.StatusActive {
		l.Error("user is not active")
		resputil.HTTPError(c, http.StatusUnauthorized, "User is not active", resputil.NotSpecified)
		return
	}

	q := query.Account
	uq := query.UserAccount
	lastUserQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID)).Last()
	if err != nil {
		l.Error("user has no queue", err)
		resputil.Error(c, "User must has at least one queue", resputil.UserNotAllowed)
		return
	}

	lastQueue, err := q.WithContext(c).Where(q.ID.Eq(lastUserQueue.AccountID)).First()
	if err != nil {
		l.Error("user has no queue", err)
		resputil.Error(c, "User must has at least one queue", resputil.UserNotAllowed)
		return
	}

	// Generate JWT tokens
	jwtMessage := util.JWTMessage{
		UserID:           user.ID,
		Username:         user.Name,
		QueueID:          lastQueue.ID,
		QueueName:        lastQueue.Name,
		RoleQueue:        lastUserQueue.Role,
		AccessMode:       lastUserQueue.AccessMode,
		PublicAccessMode: user.AccessMode,
		RolePlatform:     user.Role,
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
			AccessPublic: user.AccessMode,
			Space:        user.Space,
		},
		User: user.Attributes.Data(),
	}
	resputil.Success(c, loginResponse)
}

func shouldCreateOrUpdateUser(
	c context.Context,
	req *LoginReq,
	attr *model.UserAttribute,
) (*model.User, error) {
	// initialize user attributes
	if attr.Name == "" && req.Username != nil {
		attr.Name = *req.Username
	}
	if attr.Nickname == "" && req.Username != nil {
		attr.Nickname = *req.Username
	}

	u := query.User
	user, err := u.WithContext(c).Where(u.Name.Eq(attr.Name)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// User exists in the auth method but not in the database, create a new user
			user, err = createUser(c, attr.Name, nil)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	attr.ID = user.ID

	// If don't need to update the user, return directly
	currentAttr := user.Attributes.Data()
	newAttr := datatypes.NewJSONType(*attr)
	if currentAttr.ID != model.InvalidUserID &&
		(attr.Email == nil || reflect.DeepEqual(user.Attributes, newAttr)) {
		return user, nil
	}

	// dont update email if it has been set
	if currentAttr.Email != nil {
		attr.Email = currentAttr.Email
	}

	if _, err := u.WithContext(c).Where(u.ID.Eq(user.ID)).
		Updates(map[string]any{
			"attributes": datatypes.NewJSONType(*attr),
			"nickname":   attr.Nickname,
		}); err != nil {
		return nil, err
	}

	user.Attributes = newAttr
	return user, nil
}

// createUser is called when the user is not found in the database
func createUser(c context.Context, name string, password *string) (*model.User, error) {
	u := query.User
	uq := query.UserAccount

	var hashedPassword *string
	if password != nil {
		passwordStr := *password
		hashed, err := bcrypt.GenerateFromPassword([]byte(passwordStr), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashedPassword = lo.ToPtr(string(hashed))
	}

	user := model.User{
		Name:     name,
		Nickname: name,
		Password: hashedPassword,
		Role:     model.RoleAdmin, // todo: change to model.RoleUser
		Status:   model.StatusActive,
		Space:    fmt.Sprintf("u-%s", name),
		Attributes: datatypes.NewJSONType(model.UserAttribute{
			Email: lo.ToPtr(name + "@***REMOVED***"),
		}),
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

	// TODO: Create personal directory

	return &user, nil
}

func (mgr *AuthMgr) CreatePersonalDir(c *gin.Context, user *model.User) error {
	client := mgr.client
	jwtMessage := util.JWTMessage{
		UserID:           user.ID,
		Username:         user.Name,
		QueueID:          util.QueueIDNull,
		QueueName:        util.QueueNameNull,
		RoleQueue:        model.RoleGuest,
		RolePlatform:     user.Role,
		AccessMode:       model.AccessModeRW,
		PublicAccessMode: model.AccessModeRO,
	}
	accessToken, _, err := mgr.tokenMgr.CreateTokens(&jwtMessage)
	if err != nil {
		return errors.New("create token err:" + err.Error())
	}
	baseurl := "http://crater.***REMOVED***/api/ss/"
	uRL := baseurl + user.Space
	// 创建请求
	req, err := http.NewRequestWithContext(c.Request.Context(), "MKCOL", uRL, http.NoBody)
	if err != nil {
		return fmt.Errorf("can't create request:%s", err.Error())
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("can't send request %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
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
	l, err := ldap.DialURL(authConfig.ACT.Auth.Address)
	if err != nil {
		return err
	}
	err = l.Bind(authConfig.ACT.Auth.UserName, authConfig.ACT.Auth.Password)
	if err != nil {
		return err
	}

	// ACT 管理员搜索用户
	searchRequest := ldap.NewSearchRequest(
		authConfig.ACT.Auth.SearchDN, // 搜索基准 DN
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

	l := logutils.Log.WithFields(logutils.Fields{
		"username": req.Username,
	})

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

	_, err = createUser(c, req.Username, &req.Password)
	if err != nil {
		l.Error("create new user", err)
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
// @Summary 类似登录，切换项目并返回新的 JWT Token
// @Description 读取body中的项目ID，生成新的 JWT Token
// @Tags Auth
// @Accept json
// @Produce json
// @Security Bearer
// @Param project_id body SwitchQueueReq true "项目ID"
// @Success 200 {object} resputil.Response[LoginResp] "用户上下文"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/auth/switch [post]
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
		UserID:           token.UserID,
		Username:         token.Username,
		QueueID:          userQueue.AccountID,
		QueueName:        req.Queue,
		RoleQueue:        userQueue.Role,
		RolePlatform:     token.RolePlatform,
		AccessMode:       userQueue.AccessMode,
		PublicAccessMode: token.PublicAccessMode,
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
