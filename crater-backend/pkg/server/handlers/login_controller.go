package handlers

import (
	"net/http"

	"github.com/aisystem/ai-protal/pkg/db/user"
	resputil "github.com/aisystem/ai-protal/pkg/server/response"

	"golang.org/x/crypto/bcrypt"

	"github.com/aisystem/ai-protal/pkg/config"
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

	user, err := lc.LoginUsecase.GetByUserName(request.UserName)
	if err != nil {
		resputil.HttpError(c, http.StatusNotFound, "User not found with the given name", 40401)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)) != nil {
		resputil.HttpError(c, http.StatusUnauthorized, "Invalid credentials", 40101)
		return
	}

	accessToken, err := lc.LoginUsecase.CreateAccessToken(user, lc.TokenConf.AccessTokenSecret, lc.TokenConf.AccessTokenExpiryHour)
	if err != nil {
		resputil.HttpError(c, http.StatusInternalServerError, err.Error(), 50010)
		return
	}

	refreshToken, err := lc.LoginUsecase.CreateRefreshToken(user, lc.TokenConf.RefreshTokenSecret, lc.TokenConf.RefreshTokenExpiryHour)
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
