package handlers

import (
	"github.com/aisystem/ai-protal/pkg/config"
	"github.com/aisystem/ai-protal/pkg/crclient"

	"github.com/aisystem/ai-protal/pkg/db/user"
	"github.com/aisystem/ai-protal/pkg/domain"
	"github.com/aisystem/ai-protal/pkg/models"

	"github.com/gin-gonic/gin"

	"golang.org/x/crypto/bcrypt"
	"sigs.k8s.io/controller-runtime/pkg/client"

	//"awesomeProject/pkg/models"
	"net/http"
)

type SignupController struct {
	SignupUsecase user.DBService
	//contextTimeout time.Duration
	Env *config.TokenConf

	CL client.Client
	//ID             int    `json:"_id"`
	//Name           string `json:"name"`
	//Email          string `json:"email"`
	//Password       string `json:"password"`
}
type SignupRequest struct {
	//Id       int    `json:"_id"`
	Name     string `json:"userName" binding:"required"`
	Role     string `json:"role" binding:"required"`
	Password string `json:"password" binding:"required"`
} //domain
/*type ErrorResponse struct {
	Message string `json:"message"`
}*/

func (sc *SignupController) NewSignupRouter(group *gin.RouterGroup) {
	//ur := repository.NewUserRepository(db, domain.CollectionUser)

	group.POST("/signup", sc.Signup)
}

func (sc *SignupController) Signup(c *gin.Context) {
	var request SignupRequest

	err := c.ShouldBind(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: err.Error(),
			Code:    40004,
		})
		return
	}

	_, err = sc.SignupUsecase.GetByUserName(request.Name) //GetUserByEmail(c, request.Email)
	if err == nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Message: "User already exists with the given Name",
			Code:    40901,
		})
		return
	}

	encryptedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(request.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
			Code:    50014,
		})
		return
	}
	if request.Role != "user" && request.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "role is invalid",
			"error_code": 40005,
		})

		return
	}
	var ct *crclient.Control
	ct = &crclient.Control{Client: sc.CL}
	request.Password = string(encryptedPassword)
	namespace := "user-" + request.Name
	err = ct.CreateUserNameSpace(namespace)
	pvcname := "home-" + request.Name + "-pvc"
	ct.CreateUserHomePVC(namespace, pvcname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "namespace creating wrong",
			"error_code": 50015,
		})

		return
	}
	user := models.User{
		UserName:  request.Name,
		Role:      request.Role,
		Password:  request.Password,
		NameSpace: namespace,
	}

	err = sc.SignupUsecase.Create(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
			Code:    50016,
		})
		return
	}

	accessToken, err := sc.SignupUsecase.CreateAccessToken(&user, sc.Env.AccessTokenSecret, sc.Env.AccessTokenExpiryHour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
			Code:    50017,
		})
		return
	}

	refreshToken, err := sc.SignupUsecase.CreateRefreshToken(&user, sc.Env.RefreshTokenSecret, sc.Env.RefreshTokenExpiryHour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
			Code:    50018,
		})
		return
	}

	signupResponse := domain.SignupResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	c.JSON(http.StatusOK, signupResponse)
}
