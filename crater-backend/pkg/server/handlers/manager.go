package handlers

import "github.com/gin-gonic/gin"

type Manager interface {
	RegisterRoute(r *gin.Engine)
}

type Handler struct {
	Method      string
	HandlerFunc []gin.HandlerFunc
	Path        string
}
