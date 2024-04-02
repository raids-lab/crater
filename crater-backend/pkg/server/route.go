package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	docs "github.com/raids-lab/crater/docs"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/constants"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/db/imagepack"
	"github.com/raids-lab/crater/pkg/db/user"
	"github.com/raids-lab/crater/pkg/server/handlers"
	"github.com/raids-lab/crater/pkg/server/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Backend struct {
	R *gin.Engine
}

func (b *Backend) RegisterService(aitaskCtrl *aitaskctl.TaskController, cl client.Client, cs *kubernetes.Clientset) {
	// Enable CORS for http://localhost:5173 in debug mode
	if gin.Mode() == gin.DebugMode {
		b.R.Use(middleware.Cors())
	}

	// Init Clients and Configs
	pvcClient := crclient.PVCClient{Client: cl}
	err := pvcClient.InitShareDir()
	if err != nil {
		panic(err)
	}
	logClient := crclient.LogClient{Client: cl, KubeClient: cs}
	nodeClient := crclient.NodeClient{Client: cl, KubeClient: cs}
	imagepackClient := crclient.ImagePackController{Client: cl}
	tokenConf := config.NewTokenConf()

	///////////////////////////////////////
	//// Public routers, no need login ////
	///////////////////////////////////////

	oldPublicRouter := b.R.Group("")
	oldAuthMgr := handlers.NewAuthMgr(aitaskCtrl, tokenConf, &pvcClient)
	oldAuthMgr.RegisterRoute(oldPublicRouter)

	///////////////////////////////////////
	//// Protected routers, need login ////
	///////////////////////////////////////

	protectedRouter := b.R.Group(constants.APIPrefix)
	protectedRouter.Use(middleware.JwtAuthMiddleware(tokenConf.AccessTokenSecret))

	shareDirMgr := handlers.NewShareDirMgr()
	aitaskMgr := handlers.NewAITaskMgr(aitaskCtrl, &pvcClient, &logClient)
	jupyterMgr := handlers.NewJupyterMgr(aitaskCtrl, &pvcClient, &logClient)
	recommenddljobMgr := handlers.NewRecommendDLJobMgr(user.NewDBService(), cl)
	datasetMgr := handlers.NewDataSetMgr(user.NewDBService(), cl)
	imagepackMgr := handlers.NewImagePackMgr(imagepack.NewDBService(), &logClient, &crclient.ImagePackController{Client: cl})
	clusterMgr := handlers.NewClusterMgr()
	shareDirMgr.RegisterRoute(protectedRouter.Group("/sharedir"))
	aitaskMgr.RegisterRoute(protectedRouter.Group("/aitask"))
	jupyterMgr.RegisterRoute(protectedRouter.Group("/jupyter"))
	recommenddljobMgr.RegisterRoute(protectedRouter.Group("/recommenddljob"))
	datasetMgr.RegisterRoute(protectedRouter.Group("/dataset"))
	clusterMgr.RegisterRoute(protectedRouter.Group("/cluster"))
	imagepackMgr.RegisterRoute(protectedRouter.Group("/image"))

	///////////////////////////////////////
	//// Admin routers, need admin role ////
	///////////////////////////////////////

	adminRouter := b.R.Group(constants.APIPrefix + "/admin")
	adminRouter.Use(middleware.JwtAuthMiddleware(tokenConf.AccessTokenSecret), middleware.AdminMiddleware())

	adminMgr := handlers.NewAdminMgr(aitaskCtrl, &nodeClient, &imagepackClient)
	adminMgr.RegisterRoute(adminRouter)
}

func Register(aitaskCtrl *aitaskctl.TaskController, cl client.Client, cs *kubernetes.Clientset) (*Backend, error) {
	s := new(Backend)

	s.R = gin.Default()

	s.R.GET("/v1/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	s.RegisterService(aitaskCtrl, cl, cs)

	// todo: DisablingWrapHandler https://github.com/swaggo/gin-swagger/blob/master/swagger.go#L205
	docs.SwaggerInfo.BasePath = "/"
	s.R.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return s, nil
}
