package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func wrapResponse(c *gin.Context, msg string, data interface{}, code ErrorCode) {
	httpCode := http.StatusOK
	if code != OK {
		httpCode = http.StatusInternalServerError
	}
	c.JSON(httpCode, gin.H{
		"code": code,
		"data": data,
		"msg":  msg,
	})
}

func Success(c *gin.Context, data interface{}) {
	wrapResponse(c, "", data, OK)
}

func Error(c *gin.Context, msg string, errorCode ErrorCode) {
	wrapResponse(c, msg, nil, errorCode)
}
