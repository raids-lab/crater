package handlers

import (
	"net/http"
	"strconv"

	"github.com/aisystem/ai-protal/pkg/domain"

	"github.com/aisystem/ai-protal/pkg/config"
	"github.com/aisystem/ai-protal/pkg/db/user"
	"github.com/aisystem/ai-protal/pkg/util"

	//"github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"

	"github.com/gin-gonic/gin"
)

type RefreshTokenMgr struct {
	//Refres
	TokenConf *config.TokenConf
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
	var request domain.RefreshTokenRequest

	err := c.ShouldBind(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: err.Error(),
			Code:    40003,
		})
		return
	}

	id, err := util.ExtractIDFromToken(request.RefreshToken, rtc.TokenConf.RefreshTokenSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Message: "User not found",
			Code:    40102,
		})
		return
	}
	intNum, _ := strconv.Atoi(id)
	//uint64Num := uint64(intNum)
	user, err := user.NewDBService().GetUserByID(uint(intNum))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Message: "User not found",
			Code:    40103,
		})
		return
	}

	accessToken, err := util.CreateAccessToken(user, rtc.TokenConf.AccessTokenSecret, rtc.TokenConf.AccessTokenExpiryHour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
			Code:    50012,
		})
		return
	}

	refreshToken, err := util.CreateRefreshToken(user, rtc.TokenConf.RefreshTokenSecret, rtc.TokenConf.RefreshTokenExpiryHour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
			Code:    50013,
		})
		return
	}

	refreshTokenResponse := domain.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	c.JSON(http.StatusOK, refreshTokenResponse)
}
