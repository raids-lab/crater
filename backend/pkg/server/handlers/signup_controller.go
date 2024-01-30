package handlers

import (
	"fmt"

	"github.com/aisystem/ai-protal/pkg/aitaskctl"
	"github.com/aisystem/ai-protal/pkg/config"
	"github.com/aisystem/ai-protal/pkg/crclient"
	"github.com/sirupsen/logrus"

	"github.com/aisystem/ai-protal/pkg/db/quota"
	"github.com/aisystem/ai-protal/pkg/db/user"
	"github.com/aisystem/ai-protal/pkg/domain"
	"github.com/aisystem/ai-protal/pkg/models"

	"github.com/gin-gonic/gin"

	"golang.org/x/crypto/bcrypt"
	"sigs.k8s.io/controller-runtime/pkg/client"

	//"awesomeProject/pkg/models"
	"net/http"
)

type SignupMgr struct {
	UserDB         user.DBService
	QuotaDB        quota.DBService
	TokenConf      *config.TokenConf
	taskController *aitaskctl.TaskController
	CL             client.Client
}
type SignupRequest struct {
	Name     string `json:"userName" binding:"required"`
	Role     string `json:"role" binding:"required"`
	Password string `json:"passWord" binding:"required"`
} //domain

func NewSignupMgr(taskController *aitaskctl.TaskController, tokenConf *config.TokenConf, cl client.Client) *SignupMgr {
	return &SignupMgr{
		UserDB:         user.NewDBService(),
		QuotaDB:        quota.NewDBService(),
		TokenConf:      tokenConf,
		taskController: taskController,
		CL:             cl,
	}
}

func (sc *SignupMgr) RegisterRoute(group *gin.RouterGroup) {
	//ur := repository.NewUserRepository(db, domain.CollectionUser)
	group.POST("/signup", sc.Signup)
}

func (sc *SignupMgr) Signup(c *gin.Context) {
	var request SignupRequest

	err := c.ShouldBind(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: err.Error(),
			Code:    40004,
		})
		return
	}

	_, err = sc.UserDB.GetByUserName(request.Name) //GetUserByEmail(c, request.Email)
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
			"error":     "role is invalid",
			"errorCode": 40005,
		})

		return
	}
	ct := &crclient.Control{Client: sc.CL}
	request.Password = string(encryptedPassword)
	err = ct.CreateUserNameSpace(request.Name)
	ct.CreateUserHomePVC(request.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     "namespace creating wrong",
			"errorCode": 50015,
		})

		return
	}
	ns := fmt.Sprintf("user-%s", request.Name)
	user := models.User{
		UserName:  request.Name,
		Role:      request.Role,
		Password:  request.Password,
		NameSpace: ns,
	}

	err = sc.UserDB.Create(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
			Code:    50016,
		})
		return
	}
	quota := models.Quota{
		// UserID:    user.ID,
		UserName:  user.UserName,
		NameSpace: ns,
		// HardQuota: models.ResourceListToJSON(v1.ResourceList{}),
		HardQuota: models.ResourceListToJSON(models.DefaultQuota),
	}
	err = sc.QuotaDB.Create(&quota)
	if err != nil {
		logrus.Infof("quota create failed: %v", err)
	}
	// 通知 TaskController 有新 Quota
	sc.taskController.AddUser(quota.UserName, quota)
	accessToken, err := sc.UserDB.CreateAccessToken(&user, sc.TokenConf.AccessTokenSecret, sc.TokenConf.AccessTokenExpiryHour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
			Code:    50017,
		})
		return
	}

	refreshToken, err := sc.UserDB.CreateRefreshToken(&user, sc.TokenConf.RefreshTokenSecret, sc.TokenConf.RefreshTokenExpiryHour)
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
		Role:         user.Role,
	}

	c.JSON(http.StatusOK, signupResponse)
}
