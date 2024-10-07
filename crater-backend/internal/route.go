package internal

import (
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	docs "github.com/raids-lab/crater/docs"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/handler/aijob"
	_ "github.com/raids-lab/crater/internal/handler/operations"
	"github.com/raids-lab/crater/internal/handler/spjob"
	_ "github.com/raids-lab/crater/internal/handler/tool"
	"github.com/raids-lab/crater/internal/middleware"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/constants"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/logutils"
)

type Backend struct {
	R *gin.Engine
}

func Register(registerConfig handler.RegisterConfig, aitaskCtrl aitaskctl.TaskControllerInterface) *Backend {
	s := new(Backend)
	s.R = gin.Default()

	// Kubernetes health check
	s.R.GET("/v1/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	// Register custom routes
	s.RegisterService(registerConfig, aitaskCtrl)

	// Swagger
	// todo: DisablingWrapHandler https://github.com/swaggo/gin-swagger/blob/master/swagger.go#L205
	if gin.Mode() == gin.DebugMode {
		docs.SwaggerInfo.BasePath = "/"
	} else {
		docs.SwaggerInfo.BasePath = "/api"
	}

	s.R.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return s
}

func (b *Backend) RegisterService(
	conf handler.RegisterConfig,
	aitaskCtrl aitaskctl.TaskControllerInterface,
) {
	// Enable CORS for http://localhost:XXXX in debug mode
	if gin.Mode() == gin.DebugMode {
		fe := os.Getenv("CRATER_FE_PORT")
		if fe != "" {
			url := "http://localhost:" + fe
			corsConf := cors.DefaultConfig()
			corsConf.AllowOrigins = []string{url}
			corsConf.AllowCredentials = true
			corsConf.AllowHeaders = []string{"Authorization", "Origin", "Content-Length", "Content-Type"}
			b.R.Use(cors.New(corsConf))
		}
	}

	// Init Clients and Configs
	logClient := crclient.LogClient{Client: conf.Client, KubeClient: conf.KubeClient}
	harborClient := crclient.NewHarborClient()

	// Init Handlers
	imagepackMgr := handler.NewImagePackMgr(&logClient, &crclient.ImagePackController{Client: conf.Client}, &harborClient)
	aijobMgr := aijob.NewAITaskMgr(aitaskCtrl, conf.Client, conf.KubeClient, &logClient)
	sparseMgr := spjob.NewSparseJobMgr(conf.Client, &logClient)

	var managers = []handler.Manager{}

	for _, register := range handler.Registers {
		manager := register(conf)
		managers = append(managers, manager)
		logutils.Log.Infof("Registering %s", manager.GetName())
	}
	///////////////////////////////////////
	//// Public routers, no need login ////
	///////////////////////////////////////

	publicRouter := b.R.Group("")

	for _, mgr := range managers {
		mgr.RegisterPublic(publicRouter.Group(mgr.GetName()))
	}

	///////////////////////////////////////
	//// Protected routers, need login ////
	///////////////////////////////////////

	protectedRouter := b.R.Group(constants.APIPrefix)
	protectedRouter.Use(middleware.AuthProtected())

	imagepackMgr.RegisterProtected(protectedRouter.Group("/images"))
	aijobMgr.RegisterProtected(protectedRouter.Group("/aijobs"))
	sparseMgr.RegisterProtected(protectedRouter.Group("/spjobs"))

	for _, mgr := range managers {
		mgr.RegisterProtected(protectedRouter.Group(mgr.GetName()))
	}

	///////////////////////////////////////
	//// Admin routers, need admin role ///
	///////////////////////////////////////

	adminRouter := b.R.Group(constants.APIPrefix + "/admin")
	adminRouter.Use(middleware.AuthProtected(), middleware.AuthAdmin())

	imagepackMgr.RegisterAdmin(adminRouter.Group("/images"))
	aijobMgr.RegisterAdmin(adminRouter.Group("/aijobs"))
	sparseMgr.RegisterAdmin(adminRouter.Group("/spjobs"))

	for _, mgr := range managers {
		mgr.RegisterAdmin(adminRouter.Group(mgr.GetName()))
	}
}
