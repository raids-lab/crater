package handlers

import (
	"github.com/aisystem/ai-protal/pkg/aitaskctl"
	"github.com/aisystem/ai-protal/pkg/config"
	"github.com/aisystem/ai-protal/pkg/crclient"
	resputil "github.com/aisystem/ai-protal/pkg/server/response"
	"github.com/aisystem/ai-protal/pkg/util"
	"github.com/sirupsen/logrus"

	"github.com/aisystem/ai-protal/pkg/db/quota"
	"github.com/aisystem/ai-protal/pkg/db/user"
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
	pvcClient      *crclient.PVCClient
	CL             client.Client
}
type SignupRequest struct {
	Name     string `json:"userName" binding:"required"`
	Role     string `json:"role" binding:"required"`
	Password string `json:"passWord" binding:"required"`
}

type SignupResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Role         string `json:"role"`
}

func NewSignupMgr(taskController *aitaskctl.TaskController, tokenConf *config.TokenConf, pvcClient *crclient.PVCClient, cl client.Client) *SignupMgr {
	return &SignupMgr{
		UserDB:         user.NewDBService(),
		QuotaDB:        quota.NewDBService(),
		TokenConf:      tokenConf,
		taskController: taskController,
		pvcClient:      pvcClient,
		CL:             cl,
	}
}

func (sc *SignupMgr) RegisterRoute(group *gin.RouterGroup) {
	// group.POST("/signup", sc.Signup)
	// group.POST("/migrate", sc.Migrate)
}

func (sc *SignupMgr) Signup(c *gin.Context) {
	var request SignupRequest

	err := c.ShouldBind(&request)
	if err != nil {
		resputil.HttpError(c, http.StatusBadRequest, err.Error(), 40004)
		return
	}

	// Bug(TODO): 无法区分用户未存在和查询错误（如数据库未连接）
	_, err = sc.UserDB.GetByUserName(request.Name) //GetUserByEmail(c, request.Email)
	if err == nil {
		resputil.HttpError(c, http.StatusConflict, "User already exists with the given Name", 40901)
		return
	}

	encryptedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(request.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		resputil.HttpError(c, http.StatusInternalServerError, err.Error(), 50014)
		return
	}
	if request.Role != "user" && request.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "role is invalid",
			"errorCode": 40005,
		})

		return
	}
	request.Password = string(encryptedPassword)
	sc.pvcClient.CreateUserHomePVC(request.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     "namespace creating wrong",
			"errorCode": 50015,
		})

		return
	}
	user := models.User{
		UserName:  request.Name,
		Role:      request.Role,
		Password:  request.Password,
		NameSpace: crclient.NameSpace,
	}

	err = sc.UserDB.Create(&user)
	if err != nil {
		resputil.HttpError(c, http.StatusInternalServerError, err.Error(), 50016)
		return
	}
	quota := models.Quota{
		// UserID:    user.ID,
		UserName:  user.UserName,
		NameSpace: crclient.NameSpace,
		// HardQuota: models.ResourceListToJSON(v1.ResourceList{}),
		HardQuota: models.ResourceListToJSON(models.DefaultQuota),
	}
	err = sc.QuotaDB.Create(&quota)
	if err != nil {
		logrus.Infof("quota create failed: %v", err)
	}
	// 通知 TaskController 有新 Quota
	sc.taskController.AddUser(quota.UserName, quota)
	accessToken, err := util.CreateAccessToken(&user, sc.TokenConf.AccessTokenSecret, sc.TokenConf.AccessTokenExpiryHour)
	if err != nil {
		resputil.HttpError(c, http.StatusInternalServerError, err.Error(), 50017)
		return
	}

	refreshToken, err := util.CreateRefreshToken(&user, sc.TokenConf.RefreshTokenSecret, sc.TokenConf.RefreshTokenExpiryHour)
	if err != nil {
		resputil.HttpError(c, http.StatusInternalServerError, err.Error(), 50018)
		return
	}

	signupResponse := SignupResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Role:         user.Role,
	}

	c.JSON(http.StatusOK, signupResponse)
}
