package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	ldap "github.com/go-ldap/ldap/v3"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	quotasvc "github.com/raids-lab/crater/pkg/db/quota"
	usersvc "github.com/raids-lab/crater/pkg/db/user"
	"github.com/raids-lab/crater/pkg/models"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	"github.com/raids-lab/crater/pkg/util"
	"golang.org/x/crypto/bcrypt"
)

type AuthMgr struct {
	userService    usersvc.DBService
	quotaService   quotasvc.DBService
	tokenConf      *config.TokenConf
	taskController *aitaskctl.TaskController
	pvcClient      *crclient.PVCClient
}

func NewAuthMgr(taskController *aitaskctl.TaskController, tokenConf *config.TokenConf, pvcClient *crclient.PVCClient) *AuthMgr {
	return &AuthMgr{
		userService:    usersvc.NewDBService(),
		quotaService:   quotasvc.NewDBService(),
		tokenConf:      tokenConf,
		taskController: taskController,
		pvcClient:      pvcClient,
	}
}

func (mgr *AuthMgr) RegisterRoute(group *gin.RouterGroup) {
	group.POST("/login", mgr.Login)
	group.POST("/migrate", mgr.Migrate)
	group.POST("/refresh", mgr.RefreshToken)
}

type LoginRequest struct {
	UserName string `json:"userName" binding:"required"`
	Password string `json:"passWord" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Role         string `json:"role"`
}

func (mgr *AuthMgr) Login(c *gin.Context) {
	var request LoginRequest

	err := c.ShouldBind(&request)
	if err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.NotSpecified)
		return
	}

	// ACT 认证
	err = mgr.ACTAuthorization(request.UserName, request.Password)
	if err != nil {
		resputil.HTTPError(c, http.StatusUnauthorized, "Invalid credentials", resputil.NotSpecified)
		return
	}

	// 查找数据库中是否存在用户
	// BUG(TODO): 无法区分用户未存在和查询错误（如数据库未连接）
	user, err := mgr.userService.GetByUserName(request.UserName)
	if err != nil {
		// 用户不存在，注册用户
		user, err = mgr.signup(request.UserName)
		if err != nil {
			resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
			return
		}
	}

	accessToken, err := util.CreateAccessToken(user, mgr.tokenConf.AccessTokenSecret, mgr.tokenConf.AccessTokenExpiryHour)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}

	refreshToken, err := util.CreateRefreshToken(user, mgr.tokenConf.RefreshTokenSecret, mgr.tokenConf.RefreshTokenExpiryHour)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}

	loginResponse := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Role:         user.Role,
	}

	c.JSON(http.StatusOK, loginResponse)
}

func (mgr *AuthMgr) NormalAuthorization(username, password string) error {
	user, err := mgr.userService.GetByUserName(username)
	if err != nil {
		return fmt.Errorf("wrong username or password")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return fmt.Errorf("wrong username or password")
	}
	return nil
}

func (mgr *AuthMgr) ACTAuthorization(username, password string) error {
	// ACT 管理员认证
	l, err := ldap.Dial("tcp", "192.168.0.9:389")
	if err != nil {
		return err
	}
	err = l.Bind("***REMOVED***", "***REMOVED***")
	if err != nil {
		return err
	}

	// ACT 管理员搜索用户
	searchRequest := ldap.NewSearchRequest(
		"OU=Lab,OU=ACT,DC=lab,DC=act,DC=buaa,Dc=edu,DC=cn", // 搜索基准 DN
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

func (mgr *AuthMgr) signup(username string) (*models.User, error) {
	err := mgr.pvcClient.CreateUserHomePVC(username)
	if err != nil {
		return nil, err
	}

	user := models.User{
		UserName:  username,
		Password:  "",
		Role:      "user",
		NameSpace: crclient.NameSpace,
	}

	err = mgr.userService.Create(&user)
	if err != nil {
		return nil, err
	}

	quota := models.Quota{
		UserName:  user.UserName,
		NameSpace: crclient.NameSpace,
		HardQuota: models.ResourceListToJSON(models.DefaultQuota),
	}

	err = mgr.quotaService.Create(&quota)
	if err != nil {
		return nil, err
	}

	// 通知 TaskController 有新 Quota
	mgr.taskController.AddUser(quota.UserName, &quota)
	return &user, nil
}

type MigrateRequest struct {
	Name     string `json:"oldName" binding:"required"`
	Password string `json:"passWord" binding:"required"`
	NewName  string `json:"newName" binding:"required"`
}

type MigrateResponse struct{}

// Migrate is used to migrate the user from the old system to the new system
// 1. Check if the user exists in the old system
// 2. Copy the user's pvc from old namespace to new namespace
// 3. Create the user in the new system
func (mgr *AuthMgr) Migrate(c *gin.Context) {
	var request MigrateRequest

	err := c.ShouldBind(&request)
	if err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.NotSpecified)
		return
	}

	_, err = mgr.userService.GetByUserName(request.Name)
	if err != nil {
		resputil.HTTPError(c, http.StatusConflict, "User does not exist with the given Name", resputil.NotSpecified)
		return
	}

	// migrate old pvc and pv from old namespace
	oldNamespace := fmt.Sprintf("user-%s", request.Name)
	oldPvcName := fmt.Sprintf(crclient.UserHomePVC, request.Name)
	newPvcName := fmt.Sprintf(crclient.UserHomePVC, request.NewName)
	err = mgr.pvcClient.MigratePvcFromOldNamespace(oldNamespace, crclient.NameSpace, oldPvcName, newPvcName)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, "migrate old pvc and pv from old namespace wrong", resputil.NotSpecified)
		return
	}
	resputil.Success(c, nil)
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func (mgr *AuthMgr) RefreshToken(c *gin.Context) {
	var request RefreshTokenRequest

	err := c.ShouldBind(&request)
	if err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.NotSpecified)
		return
	}

	userinfo, err := util.CheckAndGetUser(request.RefreshToken, mgr.tokenConf.RefreshTokenSecret)
	if err != nil {
		resputil.HTTPError(c, http.StatusUnauthorized, "User not found", resputil.NotSpecified)
		return
	}

	accessToken, err := util.CreateAccessToken(&userinfo, mgr.tokenConf.AccessTokenSecret, mgr.tokenConf.AccessTokenExpiryHour)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}

	refreshToken, err := util.CreateRefreshToken(&userinfo, mgr.tokenConf.RefreshTokenSecret, mgr.tokenConf.RefreshTokenExpiryHour)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}

	refreshTokenResponse := RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	c.JSON(http.StatusOK, refreshTokenResponse)
}
