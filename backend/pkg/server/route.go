package server

import (
	"net/http"

	"github.com/aisystem/ai-protal/pkg/server/handlers"
	"github.com/gin-gonic/gin"
)

type Backend struct {
	R *gin.Engine
}

func (b *Backend) RegisterService(manager handlers.Manager) {
	manager.RegisterRoute(b.R)
}

func Register() (*Backend, error) {
	s := new(Backend)

	s.R = gin.Default()
	s.R.GET("/v1/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	return s, nil
}
