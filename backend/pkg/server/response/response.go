package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func WrapResponse(c *gin.Context, status bool, msg string, data interface{}) {
	if status {
		c.JSON(http.StatusOK, gin.H{
			"status": status,
			"error":  msg,
			"data":   data,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"status": status,
			"error":  msg,
		})
	}
}

func WrapSuccessResponse(c *gin.Context, data interface{}) {
	WrapResponse(c, true, "", data)
}

func WrapFailedResponse(c *gin.Context, msg string) {
	WrapResponse(c, false, msg, nil)
}
