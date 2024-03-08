package handlers

import (
	"fmt"
	"net/http"

	"github.com/aisystem/ai-protal/pkg/config"
	"github.com/aisystem/ai-protal/pkg/crclient"
	"github.com/aisystem/ai-protal/pkg/db/user"
	"github.com/aisystem/ai-protal/pkg/models"
	resputil "github.com/aisystem/ai-protal/pkg/server/response"
	ldap "github.com/go-ldap/ldap/v3"

	//"github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"
	"github.com/gin-gonic/gin"
)

type LoginMgr struct {
	LoginUsecase user.DBService //domain.LoginUsecase
	TokenConf    *config.TokenConf
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

func NewLoginMgr(tokenConf *config.TokenConf) *LoginMgr {
	return &LoginMgr{
		LoginUsecase: user.NewDBService(),
		TokenConf:    tokenConf,
	}
}

func (lc *LoginMgr) RegisterRoute(group *gin.RouterGroup) {
	group.POST("/login", lc.Login)
}

func (lc *LoginMgr) Login(c *gin.Context) {
	var request LoginRequest //domain.LoginRequest

	err := c.ShouldBind(&request)
	if err != nil {
		resputil.HttpError(c, http.StatusBadRequest, err.Error(), 40002)
		return
	}
	user := models.User{
		UserName:  request.UserName,
		Role:      "",
		Password:  request.Password,
		NameSpace: crclient.NameSpace,
	}
	//act管理员认证
	l, err := ldap.Dial("tcp", "192.168.0.9:389")
	if err != nil {
		fmt.Println("连接失败", err)
	}
	err = l.Bind("***REMOVED***", "***REMOVED***")
	if err != nil {
		fmt.Println("管理员认证失败", err)
	}
	//act管理员搜索用户
	searchRequest := ldap.NewSearchRequest(
		"OU=Lab,OU=ACT,DC=lab,DC=act,DC=buaa,Dc=edu,DC=cn", // 搜索基准DN
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(sAMAccountName=%s)", request.UserName), // 过滤条件
		[]string{"dn"}, // 返回的属性列表
		nil,
	)

	// 执行搜索请求
	searchResult, err := l.Search(searchRequest)
	if err != nil {
		fmt.Println(err)
	}

	if len(searchResult.Entries) != 1 {
		fmt.Println("用户不存在或返回条目过多")
	}

	//用户存在，验证用户密码
	if len(searchResult.Entries) == 1 {
		userDN := searchResult.Entries[0].DN
		err = l.Bind(userDN, request.Password)
		if err != nil {
			fmt.Println("认证失败")
			resputil.HttpError(c, http.StatusUnauthorized, "Invalid credentials", 40101)
			return

		}
	}

	// user, err := lc.LoginUsecase.GetByUserName(request.UserName)
	// if err != nil {
	// 	resputil.HttpError(c, http.StatusNotFound, "User not found with the given name", 40401)
	// 	return
	// }

	// if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)) != nil {
	// 	resputil.HttpError(c, http.StatusUnauthorized, "Invalid credentials", 40101)
	// 	return
	// }

	accessToken, err := lc.LoginUsecase.CreateAccessToken(&user, lc.TokenConf.AccessTokenSecret, lc.TokenConf.AccessTokenExpiryHour)
	if err != nil {
		resputil.HttpError(c, http.StatusInternalServerError, err.Error(), 50010)
		return
	}

	refreshToken, err := lc.LoginUsecase.CreateRefreshToken(&user, lc.TokenConf.RefreshTokenSecret, lc.TokenConf.RefreshTokenExpiryHour)
	if err != nil {
		resputil.HttpError(c, http.StatusInternalServerError, err.Error(), 50011)
		return
	}

	loginResponse := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Role:         user.Role,
	}

	c.JSON(http.StatusOK, loginResponse)
}
