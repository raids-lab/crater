package server

import (
	"net/http"

	"github.com/aisystem/ai-protal/pkg/aitaskctl"
	"github.com/aisystem/ai-protal/pkg/config"
	"github.com/aisystem/ai-protal/pkg/constants"
	"github.com/aisystem/ai-protal/pkg/crclient"
	"github.com/aisystem/ai-protal/pkg/db/user"
	"github.com/aisystem/ai-protal/pkg/server/handlers"
	"github.com/aisystem/ai-protal/pkg/server/middleware"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Backend struct {
	R *gin.Engine
}

func (b *Backend) RegisterService(aitaskCtrl *aitaskctl.TaskController, cl client.Client) {
	// enable cors in debug mode
	if gin.Mode() == gin.DebugMode {
		b.R.Use(middleware.Cors())
	}

	// timeout := time.Duration(Env.ContextTimeout) * time.Second
	// public routers
	publicRouter := b.R.Group("")
	tokenConf := config.NewTokenConf()

	loginMgr := handlers.NewLoginMgr(tokenConf)
	signupMgr := handlers.NewSignupMgr(tokenConf, cl)
	tokenMgr := handlers.NewRefreshTokenMgr(tokenConf)

	loginMgr.RegisterRoute(publicRouter)
	signupMgr.RegisterRoute(publicRouter)
	tokenMgr.RegisterRoute(publicRouter)

	// protected routers, need login

	protectedRouter := b.R.Group(constants.APIPrefix)
	protectedRouter.Use(middleware.JwtAuthMiddleware(tokenConf.AccessTokenSecret))

	shareDirMgr := handlers.NewShareDirMgr()
	shareDirMgr.RegisterRoute(protectedRouter.Group("/sharedir"))

	aitaskMgr := handlers.NewAITaskMgr(aitaskCtrl, &crclient.PVCClient{Client: cl})
	recommenddljobMgr := handlers.NewRecommendDLJobMgr(user.NewDBService(), cl)
	datasetMgr := handlers.NewDataSetMgr(user.NewDBService(), cl)

	aitaskMgr.RegisterRoute(protectedRouter.Group("/aitask"))
	recommenddljobMgr.RegisterRoute(protectedRouter.Group("/recommenddljob"))
	datasetMgr.RegisterRoute(protectedRouter.Group("/dataset"))

	adminRouter := b.R.Group(constants.APIPrefix + "/admin")
	adminRouter.Use(middleware.JwtAuthMiddleware(tokenConf.AccessTokenSecret), middleware.AdminMiddleware())
	adminMgr := handlers.NewAdminMgr(aitaskCtrl)
	adminMgr.RegisterRoute(adminRouter)
}

func Register(aitaskCtrl *aitaskctl.TaskController, cl client.Client) (*Backend, error) {
	s := new(Backend)

	s.R = gin.Default()

	s.R.GET("/v1/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	s.RegisterService(aitaskCtrl, cl)
	return s, nil
}
