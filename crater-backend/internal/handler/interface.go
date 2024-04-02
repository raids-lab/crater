package handler

import "github.com/gin-gonic/gin"

type Manager interface {
	RegisterPublic(group *gin.RouterGroup)
	RegisterProtected(group *gin.RouterGroup)
	RegisterAdmin(group *gin.RouterGroup)
}
