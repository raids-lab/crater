package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	imagepackv1 "github.com/raids-lab/crater/pkg/apis/imagepack/v1"
	"github.com/raids-lab/crater/pkg/crclient"
	imagepacksvc "github.com/raids-lab/crater/pkg/db/imagepack"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	payload "github.com/raids-lab/crater/pkg/server/payload"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	"github.com/raids-lab/crater/pkg/util"
	uuid "github.com/satori/go.uuid"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ImagePackMgr struct {
	imagepackService imagepacksvc.DBService
	logClient        *crclient.LogClient
	imagepackClient  *crclient.ImagePackController
}

type ImagePackCreateRequest struct {
	GitRepository   string `json:"gitRepository"`
	AccessToken     string `json:"accessToken"`
	RegistryServer  string `json:"registryServer"`
	RegistryUser    string `json:"registryUser"`
	RegistryPass    string `json:"registryPass"`
	RegistryProject string `json:"registryProject"`
	ImageName       string `json:"imageName"`
	ImageTag        string `json:"imageTag"`
}

type ImagePackDeleteRequest struct {
	ImagePackName string `json:"imagepackname"`
}

func (mgr *ImagePackMgr) RegisterRoute(g *gin.RouterGroup) {
	g.GET("/list", mgr.ListAll)
	g.POST("/create", mgr.Create)
	g.POST("/delete", mgr.Delete)
	g.GET("/available", mgr.ListAvailableImages)
}

func NewImagePackMgr(
	imagepackService imagepacksvc.DBService, logClient *crclient.LogClient, imagepackClient *crclient.ImagePackController,
) *ImagePackMgr {
	return &ImagePackMgr{
		imagepackService: imagepackService,
		logClient:        logClient,
		imagepackClient:  imagepackClient,
	}
}

func (mgr *ImagePackMgr) Create(c *gin.Context) {
	logutils.Log.Infof("ImagePack Create, url: %s", c.Request.URL)
	req := &ImagePackCreateRequest{}
	userContext, _ := util.GetUserFromGinContext(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	mgr.requestDefaultValue(req)
	mgr.createImagePack(c, req, userContext)
	resputil.Success(c, "")
}

func (mgr *ImagePackMgr) requestDefaultValue(req *ImagePackCreateRequest) {
	if req.RegistryServer == "" {
		req.RegistryServer = "***REMOVED***"
		req.RegistryUser = "***REMOVED***"
		req.RegistryPass = "***REMOVED***"
		req.RegistryProject = "crater-images"
	}
}

func (mgr *ImagePackMgr) createImagePack(ctx *gin.Context, req *ImagePackCreateRequest, userContext util.UserContext) {
	imagepackName := fmt.Sprintf("%s-%s", userContext.UserName, uuid.NewV4().String())
	// create ImagePack CRD
	imagepackCRD := &imagepackv1.ImagePack{
		ObjectMeta: v1.ObjectMeta{
			Name:      imagepackName,
			Namespace: userContext.Namespace,
		},
		Spec: imagepackv1.ImagePackSpec{
			GitRepository:   req.GitRepository,
			AccessToken:     req.AccessToken,
			RegistryServer:  req.RegistryServer,
			RegistryUser:    req.RegistryUser,
			RegistryPass:    req.RegistryPass,
			RegistryProject: req.RegistryProject,
			UserName:        userContext.UserName,
			ImageName:       req.ImageName,
			ImageTag:        req.ImageTag,
		},
	}
	if err := mgr.imagepackClient.CreateImagePack(ctx, imagepackCRD); err != nil {
		logutils.Log.Errorf("create imagepack CRD failed, params: %+v err:%v", imagepackCRD, err)
		return
	}

	// create ImagePack DataBase entity
	imageLink := fmt.Sprintf("%s/%s/%s:%s", req.RegistryServer, req.RegistryProject, req.ImageName, req.ImageTag)

	imagepackEntity := &models.ImagePack{
		ImagePackName: imagepackName,
		ImageLink:     imageLink,
		UserName:      userContext.UserName,
		NameSpace:     userContext.Namespace,
		Status:        string(imagepackv1.PackJobInitial),
		NameTag:       fmt.Sprintf("%s:%s", req.ImageName, req.ImageTag),
	}
	if err := mgr.imagepackService.Create(imagepackEntity); err != nil {
		logutils.Log.Errorf("create imagepack entity failed, params: %+v", imagepackEntity)
	}
}

func (mgr *ImagePackMgr) ListAll(c *gin.Context) {
	logutils.Log.Infof("ImagePack Create, url: %s", c.Request.URL)
	userContext, _ := util.GetUserFromGinContext(c)
	imagepacks, err := mgr.imagepackService.ListAll(userContext.UserName)
	if err != nil {
		logutils.Log.Errorf("fetch imagepacks entity failed, err:%v", err)
	}
	for i := range imagepacks {
		imagepack := &imagepacks[i]
		if imagepack.Status != string(imagepackv1.PackJobFinished) && imagepack.Status != string(imagepackv1.PackJobFailed) {
			mgr.updateImagePackStatus(c, userContext, imagepack)
		}
	}
	resputil.Success(c, imagepacks)
}

func (mgr *ImagePackMgr) updateImagePackStatus(ctx *gin.Context, userContext util.UserContext, imagepack *models.ImagePack) {
	imagepackCRD, err := mgr.imagepackClient.GetImagePack(ctx, imagepack.ImagePackName, userContext.Namespace)
	if err != nil {
		logutils.Log.Errorf("fetch imagepack CRD failed, err:%v", err)
	}
	logutils.Log.Infof("current stage:%s ----- new stage: %s", imagepack.Status, string(imagepackCRD.Status.Stage))
	if err := mgr.imagepackService.UpdateStatusByEntity(imagepack, string(imagepackCRD.Status.Stage)); err != nil {
		logutils.Log.Errorf("save imagepack status failed, err:%v status:%v", err, *imagepack)
	}
}

func (mgr *ImagePackMgr) Delete(c *gin.Context) {
	logutils.Log.Infof("ImagePack Delete, url: %s", c.Request.URL)
	userContext, _ := util.GetUserFromGinContext(c)
	imagePackDeleteRequest := &ImagePackDeleteRequest{}
	if err := c.ShouldBindJSON(imagePackDeleteRequest); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		logutils.Log.Errorf(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	logutils.Log.Infof("imagepackName %s", imagePackDeleteRequest.ImagePackName)
	imagepackName := imagePackDeleteRequest.ImagePackName
	if imagepackName == "default" {
		resputil.Error(c, "please parse param imagepackName", resputil.NotSpecified)
		return
	}
	flag := true
	if err := mgr.imagepackClient.DeleteImagePack(c, imagepackName, userContext.Namespace); err != nil {
		flag = false
		logutils.Log.Errorf("delete imagepack CRD failed! err:%v", err)
	}
	if err := mgr.imagepackService.DeleteByName(imagepackName); err != nil {
		flag = false
		logutils.Log.Errorf("delete imagepack entity failed! err:%v", err)
	}
	if flag {
		resputil.Success(c, "")
	} else {
		resputil.Error(c, "failed to find imagepack or entity", resputil.NotSpecified)
	}
}

func (mgr *ImagePackMgr) ListAvailableImages(c *gin.Context) {
	userContext, _ := util.GetUserFromGinContext(c)
	imagepacks, err := mgr.imagepackService.ListAvailable(userContext.UserName)
	if err != nil {
		logutils.Log.Errorf("fetch available imagepack failed, err:%v", err)
		resputil.Error(c, "fetch available imagepack failed", resputil.NotSpecified)
	}
	imageLinks := make([]string, len(imagepacks))
	for i := range imagepacks {
		imageLinks[i] = imagepacks[i].ImageLink
	}

	// manually add public imagelink
	imageLinks = append(imageLinks, "jupyter/base-notebook:ubuntu-22.04")

	resp := payload.GetImagesResp{Images: imageLinks}
	resputil.Success(c, resp)
}
