package handlers

import (
	"net/http"

	resputil "github.com/aisystem/ai-protal/pkg/server/response"

	"github.com/aisystem/ai-protal/pkg/config"
	"github.com/aisystem/ai-protal/pkg/util"

	//"github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"

	"github.com/gin-gonic/gin"
)

type RefreshTokenMgr struct {
	//Refres
	TokenConf *config.TokenConf
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func NewRefreshTokenMgr(tokenConf *config.TokenConf) *RefreshTokenMgr {
	return &RefreshTokenMgr{
		TokenConf: tokenConf,
	}
}

func (rtc *RefreshTokenMgr) RegisterRoute(group *gin.RouterGroup) {
	//ur := repository.NewUserRepository(db, domain.CollectionUser)

	group.POST("/refresh", rtc.RefreshToken)
}

func (rtc *RefreshTokenMgr) RefreshToken(c *gin.Context) {
	var request RefreshTokenRequest

	err := c.ShouldBind(&request)
	if err != nil {
		resputil.HttpError(c, http.StatusBadRequest, err.Error(), 40003)
		return
	}

	userinfo, err := util.CheckAndGetUser(request.RefreshToken, rtc.TokenConf.RefreshTokenSecret)
	if err != nil {
		resputil.HttpError(c, http.StatusUnauthorized, "User not found", 40102)
		return
	}

	accessToken, err := util.CreateAccessToken(&userinfo, rtc.TokenConf.AccessTokenSecret, rtc.TokenConf.AccessTokenExpiryHour)
	if err != nil {
		resputil.HttpError(c, http.StatusInternalServerError, err.Error(), 50012)
		return
	}

	refreshToken, err := util.CreateRefreshToken(&userinfo, rtc.TokenConf.RefreshTokenSecret, rtc.TokenConf.RefreshTokenExpiryHour)
	if err != nil {
		resputil.HttpError(c, http.StatusInternalServerError, err.Error(), 50013)
		return
	}

	refreshTokenResponse := RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	c.JSON(http.StatusOK, refreshTokenResponse)
}
