package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func WrapResponse(c *gin.Context, status bool, msg string, data interface{}, er_code int) {
	if status {
		c.JSON(http.StatusOK, gin.H{
			"status": status,
			"error":  msg,
			"data":   data,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":    status,
			"error":     msg,
			"errorCode": er_code,
		})
	}
}

func WrapSuccessResponse(c *gin.Context, data interface{}) {
	WrapResponse(c, true, "", data, 1)
}

func WrapFailedResponse(c *gin.Context, msg string, er_code int) {
	WrapResponse(c, false, msg, nil, er_code)
}
