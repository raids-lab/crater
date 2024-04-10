package internal

import (
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	docs "github.com/raids-lab/crater/docs"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/middleware"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/constants"
	"github.com/raids-lab/crater/pkg/crclient"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Backend struct {
	R *gin.Engine
}

func Register(aitaskCtrl *aitaskctl.TaskController, cl client.Client, cs *kubernetes.Clientset) *Backend {
	s := new(Backend)
	s.R = gin.Default()

	// Kubernetes health check
	s.R.GET("/v1/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	// Register custom routes
	s.RegisterService(aitaskCtrl, cl, cs)

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

func (b *Backend) RegisterService(aitaskCtrl *aitaskctl.TaskController, cl client.Client, cs *kubernetes.Clientset) {
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
	pvcClient := crclient.PVCClient{Client: cl}
	err := pvcClient.InitShareDir()
	if err != nil {
		panic(err)
	}
	httpClient := http.Client{}
	logClient := crclient.LogClient{Client: cl, KubeClient: cs}
	nodeClient := crclient.NodeClient{Client: cl, KubeClient: cs}

	// Init Handlers
	authMgr := handler.NewAuthMgr(aitaskCtrl, &httpClient)
	aijobMgr := handler.NewAIJobMgr(aitaskCtrl, &pvcClient, &logClient)
	labelMgr := handler.NewLabelMgr()
	projectMgr := handler.NewProjectMgr()
	nodeMgr := handler.NewNodeMgr(&nodeClient)
	userMgr := handler.NewUserMgr(aitaskCtrl)
	contextMgr := handler.NewContextMgr()

	///////////////////////////////////////
	//// Public routers, no need login ////
	///////////////////////////////////////

	publicRouter := b.R.Group("")

	authMgr.RegisterPublic(publicRouter)

	///////////////////////////////////////
	//// Protected routers, need login ////
	///////////////////////////////////////

	protectedRouter := b.R.Group(constants.APIPrefix)
	protectedRouter.Use(middleware.AuthProtected())

	authMgr.RegisterProtected(protectedRouter.Group("/switch"))
	aijobMgr.RegisterProtected(protectedRouter.Group("/aijobs"))
	labelMgr.RegisterProtected(protectedRouter.Group("/labels"))
	projectMgr.RegisterProtected(protectedRouter.Group("/projects"))
	nodeMgr.RegisterProtected(protectedRouter.Group("/nodes"))
	contextMgr.RegisterProtected(protectedRouter.Group("/context"))

	///////////////////////////////////////
	//// Admin routers, need admin role ///
	///////////////////////////////////////

	adminRouter := b.R.Group(constants.APIPrefix + "/admin")
	adminRouter.Use(middleware.AuthProtected(), middleware.AuthAdmin())

	aijobMgr.RegisterAdmin(adminRouter.Group("/aijobs"))
	labelMgr.RegisterAdmin(adminRouter.Group("/labels"))
	projectMgr.RegisterAdmin(adminRouter.Group("/projects"))
	nodeMgr.RegisterAdmin(adminRouter.Group("/nodes"))
	userMgr.RegisterAdmin(adminRouter.Group("/users"))
}
