// TODO(huangsy): This file is too long; consider splitting it into smaller files.
package image

import (
	"math"

	"github.com/gin-gonic/gin"
	imrocreq "github.com/imroc/req/v3"

	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/packer"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	handler.Registers = append(handler.Registers, NewImagePackMgr)
}

type ImagePackMgr struct {
	name            string
	imagepackClient *crclient.ImagePackController
	imagePacker     packer.ImagePackerInterface
	imageRegistry   imageregistry.ImageRegistryInterface
	req             *imrocreq.Client
}

func (mgr *ImagePackMgr) GetName() string { return mgr.name }

func (mgr *ImagePackMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ImagePackMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/kaniko", mgr.UserListKaniko)
	g.POST("/kaniko", mgr.UserCreateByPipApt)
	g.POST("/dockerfile", mgr.UserCreateByDockerfile)
	g.POST("/envd", mgr.UserCreateByEnvd)
	g.POST("/remove", mgr.UserRemoveKanikoByID)

	g.GET("/image", mgr.UserListImage)
	g.POST("/image", mgr.UserUploadImage)
	g.DELETE("/image/:id", mgr.DeleteImageByID)

	g.GET("/available", mgr.ListAvailableImages)
	g.GET("/getbyname", mgr.GetKanikoByImagePackName)
	g.POST("/quota", mgr.UpdateProjectQuota)

	g.GET("/podname", mgr.GetImagepackPodName)
	g.POST("/credential", mgr.UserGetProjectCredential)
	g.GET("/quota", mgr.UserGetProjectDetail)
	g.POST("/valid", mgr.CheckLinkValidity)
	g.POST("/description", mgr.UserChangeImageDescription)
	g.POST("/deleteimage", mgr.UserDeleteImageByIDList)
	g.POST("/type", mgr.UserChangeImageTaskType)
	g.GET("/harbor", mgr.GetHarborIP)
	g.POST("/tags", mgr.UserChangeImageTags)
	g.GET("/template", mgr.GetKanikoTemplateByImagePackName)

	g.POST("/share", mgr.UserShareImage)
	g.DELETE("/share", mgr.UserCancelShareImage)
	g.GET("/share", mgr.GetImageGrantedUserOrAccount)
	g.GET("/user", mgr.UserSearchUngrantedUsers)
	g.GET("/account", mgr.UserGetImageUngrantedAccounts)
	g.GET("/cudabaseimage", mgr.UserGetCudaBaseImages)
	g.POST("/cudabaseimage", mgr.UserAddCudaBaseImage)
	g.DELETE("/cudabaseimage/:id", mgr.UserDeleteCudaBaseImage)
	g.POST("/arch", mgr.UserUpdateImageArch)
}

func (mgr *ImagePackMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/kaniko", mgr.AdminListKaniko)
	g.GET("/image", mgr.AdminListImage)
	g.POST("/deleteimage", mgr.AdminDeleteImageByIDList)
	g.POST("/remove", mgr.AdminRemoveKanikoByIDList)
	g.POST("/type", mgr.AdminChangeImageTaskType)
	g.POST("/change/:id", mgr.AdminUpdateImagePublicStatus)
	g.POST("/description", mgr.AdminChangeImageDescription)
	g.POST("/tags", mgr.AdminChangeImageTags)
	g.POST("/arch", mgr.AdminUpdateImageArch)
}

func NewImagePackMgr(conf *handler.RegisterConfig) handler.Manager {
	return &ImagePackMgr{
		name:            "images",
		imagepackClient: &crclient.ImagePackController{Client: conf.Client},
		imagePacker:     conf.ImagePacker,
		imageRegistry:   conf.ImageRegistry,
		req:             imrocreq.C(),
	}
}

var (
	UserNameSpace   = config.GetConfig().Namespaces.Image
	ProjectIsPublic = true
	//nolint:mnd // default project quota: 20GB
	GBit = int64(math.Pow(2, 30))
)
