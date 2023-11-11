package handlers

import "github.com/gin-gonic/gin"

type Manager interface {
	RegisterRoute(r *gin.RouterGroup)
}

type Handler struct {
	Method      string
	HandlerFunc []gin.HandlerFunc
	Path        string
}
