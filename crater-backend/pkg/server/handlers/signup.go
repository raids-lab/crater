package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/db/quota"
	"github.com/raids-lab/crater/pkg/db/user"
	"github.com/raids-lab/crater/pkg/models"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	"github.com/raids-lab/crater/pkg/util"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func NewSignupMgr(taskController *aitaskctl.TaskController, tokenConf *config.TokenConf,
	pvcClient *crclient.PVCClient, cl client.Client) *SignupMgr {
	return &SignupMgr{
		UserDB:         user.NewDBService(),
		QuotaDB:        quota.NewDBService(),
		TokenConf:      tokenConf,
		taskController: taskController,
		pvcClient:      pvcClient,
		CL:             cl,
	}
}

func (sc *SignupMgr) RegisterRoute(_ *gin.RouterGroup) {
}

func (sc *SignupMgr) Signup(c *gin.Context) {
	var request SignupRequest

	err := c.ShouldBind(&request)
	if err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.NotSpecified)
		return
	}

	// Bug(TODO): 无法区分用户未存在和查询错误（如数据库未连接）
	_, err = sc.UserDB.GetByUserName(request.Name)
	if err == nil {
		resputil.HTTPError(c, http.StatusConflict, "User already exists with the given Name", resputil.NotSpecified)
		return
	}

	encryptedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(request.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
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
	err = sc.pvcClient.CreateUserHomePVC(request.Name)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	userModel := models.User{
		UserName:  request.Name,
		Role:      request.Role,
		Password:  request.Password,
		NameSpace: crclient.NameSpace,
	}

	err = sc.UserDB.Create(&userModel)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}
	userQuota := models.Quota{
		// UserID:    user.ID,
		UserName:  userModel.UserName,
		NameSpace: crclient.NameSpace,
		// HardQuota: models.ResourceListToJSON(v1.ResourceList{}),
		HardQuota: models.ResourceListToJSON(models.DefaultQuota),
	}
	err = sc.QuotaDB.Create(&userQuota)
	if err != nil {
		logrus.Infof("quota create failed: %v", err)
	}
	// 通知 TaskController 有新 Quota
	sc.taskController.AddUser(userQuota.UserName, &userQuota)
	accessToken, err := util.CreateAccessToken(&userModel, sc.TokenConf.AccessTokenSecret, sc.TokenConf.AccessTokenExpiryHour)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}

	refreshToken, err := util.CreateRefreshToken(&userModel, sc.TokenConf.RefreshTokenSecret, sc.TokenConf.RefreshTokenExpiryHour)
	if err != nil {
		resputil.HTTPError(c, http.StatusInternalServerError, err.Error(), resputil.NotSpecified)
		return
	}

	signupResponse := SignupResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Role:         userModel.Role,
	}

	c.JSON(http.StatusOK, signupResponse)
}
