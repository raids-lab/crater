package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// wrapResponse wraps the response data and sends it back to the client.
// It takes in a Gin context, a message string, data any, and an ErrorCode.
// The function sets the appropriate HTTP status code based on the ErrorCode.
// It then serializes the response data into JSON format and sends it back to the client.
func wrapResponse(c *gin.Context, msg string, data any, code ErrorCode) {
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

// Success sends a successful response to the client with the provided data.
// It wraps the response using the wrapResponse function and sets the HTTP status code to OK.
func Success(c *gin.Context, data any) {
	wrapResponse(c, "", data, OK)
}

// Error sends an error response to the client with the specified message and error code.
func Error(c *gin.Context, msg string, errorCode ErrorCode) {
	wrapResponse(c, msg, nil, errorCode)
}

// HttpError sends an HTTP error response with the specified HTTP code, error message, and error code.
func HttpError(c *gin.Context, httpCode int, msg string, errorCode ErrorCode) {
	c.JSON(httpCode, gin.H{
		"code": errorCode,
		"data": nil,
		"msg":  msg,
	})
}
