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
	"github.com/raids-lab/crater/internal/middleware"
	"github.com/raids-lab/crater/pkg/constants"
)

type Backend struct {
	R *gin.Engine
}

func Register(registerConfig *handler.RegisterConfig) *Backend {
	s := new(Backend)
	s.R = gin.Default()

	// Kubernetes health check
	s.R.GET("/v1/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	// Register custom routes
	s.RegisterService(registerConfig)

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

func (b *Backend) RegisterService(conf *handler.RegisterConfig) {
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

	managers := registerManagers(conf)

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

	for _, mgr := range managers {
		mgr.RegisterProtected(protectedRouter.Group(mgr.GetName()))
	}

	///////////////////////////////////////
	//// Admin routers, need admin role ///
	///////////////////////////////////////

	adminRouter := b.R.Group(constants.APIPrefix + "/admin")
	adminRouter.Use(middleware.AuthProtected(), middleware.AuthAdmin())

	for _, mgr := range managers {
		mgr.RegisterAdmin(adminRouter.Group(mgr.GetName()))
	}
}
