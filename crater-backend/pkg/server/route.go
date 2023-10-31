package server

import (
	"net/http"

	"github.com/aisystem/ai-protal/pkg/bootstrap"
	"github.com/aisystem/ai-protal/pkg/constants"
	"github.com/aisystem/ai-protal/pkg/db/user"
	"github.com/aisystem/ai-protal/pkg/middleware"
	"github.com/aisystem/ai-protal/pkg/server/handlers"
	"github.com/aisystem/ai-protal/pkg/util"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Backend struct {
	R *gin.Engine
}

func (b *Backend) RegisterService(manager handlers.Manager, cl client.Client) {
	Env := bootstrap.NewEnv()
	b.R.Use(middleware.Cors())
	publicRouter := b.R.Group("")
	db := user.NewDBService()
	//timeout := time.Duration(Env.ContextTimeout) * time.Second
	lc := &handlers.LoginController{
		LoginUsecase: db,
		Env:          Env,
	}
	sc := &handlers.SignupController{
		SignupUsecase: db,
		Env:           Env,
		CL:            cl,
	}
	rtc := &handlers.RefreshTokenController{

		Env: Env,
	}
	sc.NewSignupRouter(publicRouter)
	lc.NewLoginRouter(publicRouter)
	rtc.NewRefreshTokenRouter(publicRouter)
	protectedRouter := b.R.Group(constants.APIPrefix + "/task")
	protectedRouter.Use(middleware.JwtAuthMiddleware(Env.AccessTokenSecret))
	manager.RegisterRoute(protectedRouter)
}

func Register(taskUpdateChan <-chan util.TaskUpdateChan, cl client.Client) (*Backend, error) {
	s := new(Backend)

	s.R = gin.Default()

	s.R.GET("/v1/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	s.RegisterService(handlers.NewTaskMgr(taskUpdateChan), cl)
	s.R.Run(":8078")
	return s, nil
}
