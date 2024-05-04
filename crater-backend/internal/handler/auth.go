package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	ldap "github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuthMgr struct {
	client         *http.Client
	tokenMgr       *util.TokenManager
	taskController *aitaskctl.TaskController
}

func NewAuthMgr(taskController *aitaskctl.TaskController, client *http.Client) Manager {
	return &AuthMgr{
		client:         client,
		tokenMgr:       util.GetTokenMgr(),
		taskController: taskController,
	}
}

func (mgr *AuthMgr) RegisterPublic(g *gin.RouterGroup) {
	g.POST("/login", mgr.Login)
	g.POST("/refresh", mgr.RefreshToken)
}

func (mgr *AuthMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("", mgr.SwitchProject) // 切换项目 /switch
}

func (mgr *AuthMgr) RegisterAdmin(_ *gin.RouterGroup) {}

type (
	LoginReq struct {
		Username   string `json:"username" binding:"required"` // 用户�?
		Password   string `json:"password" binding:"required"` // 密码
		AuthMethod string `json:"auth" binding:"required"`     // 认证方式 [normal, act]
	}

	LoginResp struct {
		AccessToken  string      `json:"accessToken"`
		RefreshToken string      `json:"refreshToken"`
		Context      UserContext `json:"context"`
	}

	UserContext struct {
		Queue        string     `json:"queue"`        // Current Queue Name
		RoleQueue    model.Role `json:"roleQueue"`    // User role of the queue
		RolePlatform model.Role `json:"rolePlatform"` // User role of the platform
	}
)

const (
	AuthMethodNormal = "normal"
	AuthMethodACT    = "act"
)

// Login godoc
// @Summary 用户登录
// @Description 校验用户身份，生成包含当前用户和项目�? JWT Token
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body LoginReq false "查询参数"
// @Success 200 {object} resputil.Response[LoginResp] "登录成功，返�? JWT Token 和默认个人项�?"
// @Failure 400 {object} resputil.Response[any]	"请求参数错误"
// @Failure 401 {object} resputil.Response[any]	"用户名或密码错误"
// @Failure 500 {object} resputil.Response[any]	"数据库交互错�?"
// @Router /login [post]
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

	// Check if request auth method is valid
	switch req.AuthMethod {
	case AuthMethodACT:
		if err := mgr.actAuth(req.Username, req.Password); err != nil {
			l.Error("invalid credentials: ", err)
			resputil.HTTPError(c, http.StatusUnauthorized, "Invalid credentials", resputil.NotSpecified)
			return
		}
	case AuthMethodNormal:
		if err := mgr.normalAuth(c, req.Username, req.Password); err != nil {
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
	u := query.User
	user, err := u.WithContext(c).Where(u.Name.Eq(req.Username)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// User exists in the auth method but not in the database, create a new user
			user, err = mgr.createUser(c, req.Username)
			if err != nil {
				l.Error("create new user", err)
				resputil.Error(c, "Create user failed", resputil.NotSpecified)
				return
			}
		} else {
			// Other DB error
			l.Error(err)
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
	}
	if user.Status != model.StatusActive {
		l.Error("user is not active")
		resputil.HTTPError(c, http.StatusUnauthorized, "User is not active", resputil.NotSpecified)
		return
	}

	// Generate JWT tokens
	jwtMessage := util.JWTMessage{
		UserID:       user.ID,
		Username:     user.Name,
		QueueID:      util.QueueIDNull,
		QueueName:    util.QueueNameNull,
		RoleQueue:    model.RoleGuest,
		RolePlatform: user.Role,
	}
	accessToken, refreshToken, err := mgr.tokenMgr.CreateTokens(&jwtMessage)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}
	loginResponse := LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Context: UserContext{
			Queue:        util.QueueNameNull,
			RoleQueue:    model.RoleGuest,
			RolePlatform: user.Role,
		},
	}
	resputil.Success(c, loginResponse)
}

// createUser is called when the user is not found in the database
func (mgr *AuthMgr) createUser(c *gin.Context, name string) (*model.User, error) {
	u := query.User
	user := model.User{
		Name:     name,
		Nickname: name,
		Password: nil,
		Role:     model.RoleAdmin, // todo: change to model.RoleUser
		Status:   model.StatusActive,
		Space:    fmt.Sprintf("u-%s", uuid.New().String()[:8]),
		Attributes: datatypes.NewJSONType(model.UserAttribute{
			Email: name + "@***REMOVED***",
		}),
	}
	if err := u.WithContext(c).Create(&user); err != nil {
		return nil, err
	}

	// TODO: Create personal directory

	// TODO: disable auto link to default queue in the future
	uq := query.UserQueue
	userQueue := model.UserQueue{
		UserID:     user.ID,
		QueueID:    model.DefaultQueueID,
		Role:       model.RoleUser,
		AccessMode: model.AccessModeRW,
	}
	if err := uq.WithContext(c).Create(&userQueue); err != nil {
		return nil, err
	}

	return &user, nil
}

func (mgr *AuthMgr) CreatePersonalDir(c *gin.Context, user *model.User) error {
	client := mgr.client
	jwtMessage := util.JWTMessage{
		UserID:       user.ID,
		Username:     user.Name,
		QueueID:      util.QueueIDNull,
		QueueName:    util.QueueNameNull,
		RoleQueue:    model.RoleGuest,
		RolePlatform: user.Role,
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
	req.Header.Set("Authorization", "Bearer "+accessToken)
	// 发�?�请�?
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

func (mgr *AuthMgr) actAuth(username, password string) error {
	authConfig := config.GetConfig()
	// ACT 管理员认�?
	l, err := ldap.DialURL(authConfig.ACT.Auth.Address)
	if err != nil {
		return err
	}
	err = l.Bind(authConfig.ACT.Auth.UserName, authConfig.ACT.Auth.Password)
	if err != nil {
		return err
	}

	// ACT 管理员搜索用�?
	searchRequest := ldap.NewSearchRequest(
		authConfig.ACT.Auth.SearchDN, // 搜索基准 DN
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(sAMAccountName=%s)", username), // 过滤条件
		[]string{"dn"}, // 返回的属性列�?
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

	// 用户存在，验证用户密�?
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
	RefreshReq struct {
		RefreshToken string `json:"refreshToken" binding:"required"` // 不需要添�? `Bearer ` 前缀
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

// SwitchProject godoc
// @Summary 类似登录，切换项目并返回新的 JWT Token
// @Description 读取body中的项目ID，生成新�? JWT Token
// @Tags Auth
// @Accept json
// @Produce json
// @Security Bearer
// @Param project_id body SwitchQueueReq true "项目ID"
// @Success 200 {object} resputil.Response[LoginResp] "用户上下�?"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/switch [post]
func (mgr *AuthMgr) SwitchProject(c *gin.Context) {
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
	q := query.Queue
	uq := query.UserQueue

	queue, err := q.WithContext(c).Where(q.Name.Eq(req.Queue)).First()
	if err != nil {
		resputil.Error(c, "Queue not found", resputil.NotSpecified)
		return
	}

	userQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(token.UserID), uq.QueueID.Eq(queue.ID)).First()
	if err != nil {
		resputil.Error(c, "Queue not found", resputil.NotSpecified)
		return
	}

	// Get personal project id for the user (as the default project)
	// Each user has a personal project (with the same name as the user)

	// Generate new JWT tokens
	jwtMessage := util.JWTMessage{
		UserID:       token.UserID,
		Username:     token.Username,
		QueueID:      userQueue.QueueID,
		QueueName:    req.Queue,
		RoleQueue:    userQueue.Role,
		RolePlatform: token.RolePlatform,
	}
	accessToken, refreshToken, err := mgr.tokenMgr.CreateTokens(&jwtMessage)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}
	loginResponse := LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Context: UserContext{
			Queue:        req.Queue,
			RoleQueue:    userQueue.Role,
			RolePlatform: token.RolePlatform,
		},
	}
	resputil.Success(c, loginResponse)
}
