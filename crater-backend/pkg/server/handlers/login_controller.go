package handlers

import (
	"net/http"

	"github.com/aisystem/ai-protal/pkg/db/user"

	"golang.org/x/crypto/bcrypt"

	"github.com/aisystem/ai-protal/pkg/config"
	//"github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"
	"github.com/gin-gonic/gin"
)

type LoginController struct {
	LoginUsecase user.DBService //domain.LoginUsecase
	Env          *config.TokenConf
}
type LoginRequest struct {
	UserName string `json:"userName" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}
type ErrorResponse struct {
	Message string `json:"error"`
	Code    int    `json:"error_code"`
}

func (lc *LoginController) NewLoginRouter(group *gin.RouterGroup) {
	//ur := repository.NewUserRepository(db, domain.CollectionUser)

	group.POST("/login", lc.Login)
}

func (lc *LoginController) Login(c *gin.Context) {
	var request LoginRequest //domain.LoginRequest

	err := c.ShouldBind(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: err.Error(),
			Code:    40002,
		})
		return
	}

	user, err := lc.LoginUsecase.GetByUserName(request.UserName)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Message: "User not found with the given name",
			Code:    40401,
		})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)) != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Message: "Invalid credentials",
			Code:    40101,
		})
		return
	}

	accessToken, err := lc.LoginUsecase.CreateAccessToken(user, lc.Env.AccessTokenSecret, lc.Env.AccessTokenExpiryHour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
			Code:    50010,
		})
		return
	}

	refreshToken, err := lc.LoginUsecase.CreateRefreshToken(user, lc.Env.RefreshTokenSecret, lc.Env.RefreshTokenExpiryHour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
			Code:    50011,
		})
		return
	}

	loginResponse := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	c.JSON(http.StatusOK, loginResponse)
}
