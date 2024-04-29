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
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	"golang.org/x/crypto/bcrypt"
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
		Username   string `json:"username" binding:"required"` // 用户名
		Password   string `json:"password" binding:"required"` // 密码
		AuthMethod string `json:"auth" binding:"required"`     // 认证方式 [normal, act]
	}

	LoginResp struct {
		AccessToken  string          `json:"accessToken"`
		RefreshToken string          `json:"refreshToken"`
		Context      PlatformContext `json:"context"`
	}

	PlatformContext struct {
		PID          uint       `json:"projectID"`    // Default Project ID
		ProjectRole  model.Role `json:"projectRole"`  // User role of the project
		PlatformRole model.Role `json:"platformRole"` // User role of the platform
	}
)

const (
	AuthMethodNormal = "normal"
	AuthMethodACT    = "act"
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
	p := query.Project
	up := query.UserProject
	user, err := u.WithContext(c).Where(u.Name.Eq(req.Username)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// User exists in the auth method but not in the database, create a new user
			user, err = mgr.createUserAndProject(c, req.Username)
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

	// Get personal project for the user
	var projects []payload.ProjectResp
	err = up.WithContext(c).Where(up.UserID.Eq(user.ID), p.IsPersonal.Is(true)).
		Select(p.ID, p.Name, up.Role, p.IsPersonal, p.Status).Join(p, p.ID.EqCol(up.ProjectID)).Scan(&projects)
	if err != nil {
		l.Error("DB error", err)
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	if len(projects) != 1 {
		l.Error("user has no personal project or too many personal projects")
		resputil.HTTPError(c, http.StatusUnauthorized, "User has no project", resputil.NotSpecified)
		return
	}

	// Get personal project id for the user (as the default project)
	// Each user has a personal project (with the same name as the user)

	// Generate JWT tokens
	jwtMessage := util.JWTMessage{
		UserID:       user.ID,
		ProjectID:    projects[0].ID,
		ProjectRole:  projects[0].Role,
		PlatformRole: user.Role,
	}
	accessToken, refreshToken, err := mgr.tokenMgr.CreateTokens(&jwtMessage)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}
	loginResponse := LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Context: PlatformContext{
			PID:          projects[0].ID,
			ProjectRole:  projects[0].Role,
			PlatformRole: user.Role,
		},
	}
	resputil.Success(c, loginResponse)
}

// createUserAndProject is called by Login and SingUp, create a new user and a personal project for the user
func (mgr *AuthMgr) createUserAndProject(c *gin.Context, name string) (*model.User, error) {
	db := query.Use(query.DB)

	var path string
	var userID, projectID uint

	err := db.Transaction(func(tx *query.Query) error {
		u := tx.User
		p := tx.Project
		up := tx.UserProject
		s := tx.Space

		// Create a new user with non-admin role and active status
		user := model.User{
			Name:     name,
			Nickname: nil,
			Password: nil,
			Role:     model.RoleAdmin, // todo: change to model.RoleUser
			Status:   model.StatusActive,
		}
		if err := u.WithContext(c).Create(&user); err != nil {
			return err
		}
		userID = user.ID
		// Create a personal project for the user
		project := model.Project{
			Name:          user.Name,
			Description:   nil,
			Namespace:     config.GetConfig().Workspace.Namespace,
			Status:        model.StatusActive,
			IsPersonal:    true,
			EmbeddedQuota: model.QuotaDefault,
		}
		if err := p.WithContext(c).Create(&project); err != nil {
			return err
		}
		projectID = project.ID
		// Create a user-project relationship without quota limit
		userProject := model.UserProject{
			UserID:        user.ID,
			ProjectID:     project.ID,
			Role:          model.RoleAdmin,
			AccessMode:    model.AccessModeRW,
			EmbeddedQuota: model.QuotaUnlimited,
		}
		if err := up.WithContext(c).Create(&userProject); err != nil {
			return err
		}

		// Create a space for the project, folder path is generated by uuid
		uuidStr := uuid.New().String()
		folderPath := fmt.Sprintf("/%s-%d", uuidStr[:8], project.ID)
		space := model.Space{
			ProjectID: project.ID,
			Path:      folderPath,
		}
		if err := s.WithContext(c).Create(&space); err != nil {
			return err
		}
		path = folderPath

		return nil
	})
	if err != nil {
		return nil, err
	}
	err = mgr.createPersonalDir(c, path, userID, projectID)
	if err != nil {
		return nil, err
	}
	// TODO: refactor
	// Notify the task controller that a new user is created
	oldQuota := models.Quota{
		UserName:  name,
		NameSpace: crclient.NameSpace,
		HardQuota: models.ResourceListToJSON(models.DefaultQuota),
	}
	mgr.taskController.AddUser(oldQuota.UserName, &oldQuota)

	user, err := query.User.WithContext(c).Where(query.User.Name.Eq(name)).First()
	return user, err
}

func (mgr *AuthMgr) createPersonalDir(c *gin.Context, path string, userid, projectid uint) error {
	client := mgr.client
	jwtMessage := util.JWTMessage{
		UserID:       userid,
		ProjectID:    projectid,
		ProjectRole:  model.RoleAdmin,
		PlatformRole: model.RoleUser,
	}
	accessToken, _, err := mgr.tokenMgr.CreateTokens(&jwtMessage)
	if err != nil {
		return errors.New("create token err:" + err.Error())
	}
	baseurl := "http://crater.***REMOVED***/api/ss"
	uRL := baseurl + path
	// 创建请求
	req, err := http.NewRequestWithContext(c.Request.Context(), "MKCOL", uRL, http.NoBody)
	if err != nil {
		return fmt.Errorf("can't create request:%s", err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
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

func (mgr *AuthMgr) actAuth(username, password string) error {
	authConfig := config.GetConfig()
	// ACT 管理员认证
	l, err := ldap.Dial("tcp", authConfig.ACT.Auth.Address)
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

type SwitchProjectReq struct {
	ProjectID uint `json:"id" binding:"required"`
}

// SwitchProject godoc
// @Summary 类似登录，切换项目并返回新的 JWT Token
// @Description 读取body中的项目ID，生成新的 JWT Token
// @Tags Auth
// @Accept json
// @Produce json
// @Security Bearer
// @Param project_id body SwitchProjectReq true "项目ID"
// @Success 200 {object} resputil.Response[LoginResp] "用户上下文"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/switch [post]
func (mgr *AuthMgr) SwitchProject(c *gin.Context) {
	var req SwitchProjectReq
	if err := c.ShouldBind(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token, _ := util.GetToken(c)

	// Check if the dist project exists
	up := query.UserProject
	p := query.Project

	var projects []payload.ProjectResp
	err := up.WithContext(c).Where(up.UserID.Eq(token.UserID), up.ProjectID.Eq(req.ProjectID), p.Status.Eq(uint8(model.StatusActive))).
		Select(p.ID, p.Name, up.Role, p.IsPersonal, p.Status).Join(p, p.ID.EqCol(up.ProjectID)).Scan(&projects)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	if len(projects) != 1 {
		resputil.HTTPError(c, http.StatusUnauthorized, "User has no project", resputil.NotSpecified)
		return
	}

	// Get personal project id for the user (as the default project)
	// Each user has a personal project (with the same name as the user)

	// Generate JWT tokens
	jwtMessage := util.JWTMessage{
		UserID:       token.UserID,
		ProjectID:    projects[0].ID,
		ProjectRole:  projects[0].Role,
		PlatformRole: token.PlatformRole,
	}
	accessToken, refreshToken, err := mgr.tokenMgr.CreateTokens(&jwtMessage)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}
	loginResponse := LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Context: PlatformContext{
			PID:          projects[0].ID,
			ProjectRole:  projects[0].Role,
			PlatformRole: token.PlatformRole,
		},
	}
	resputil.Success(c, loginResponse)
}
