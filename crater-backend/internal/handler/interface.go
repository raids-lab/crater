package handler

import "github.com/gin-gonic/gin"

type Handler interface {
	RegisterPublic(group *gin.RouterGroup)
	RegisterProtected(group *gin.RouterGroup)
	RegisterAdmin(group *gin.RouterGroup)
}
