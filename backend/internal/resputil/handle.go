package resputil

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
)

const firstServerErrorGroup = 500

func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// 1. 尝试提取 BizError
	bErr, isBizErr := bizerr.FromError(err)

	if !isBizErr {
		handleUnexpectedError(c, err)
		return
	}

	logServerError(err, bErr)
	respond(c, httpStatusFromError(err), bErr)
}

func handleUnexpectedError(c *gin.Context, err error) {
	// 如果完全不是业务错误，记录详细堆栈到日志（%+v 是关键）
	fmt.Printf("Unexpected Error: %+v\n", err)
	respond(c, http.StatusInternalServerError, &bizerr.BizError{
		Code:    50000,
		Message: "Internal Server Error",
	})
}

func logServerError(err error, bErr *bizerr.BizError) {
	if bErr.Code.Group() >= firstServerErrorGroup {
		fmt.Printf("Business 5xx Error: %+v\n", err)
	}
}

func httpStatusFromError(err error) int {
	bErr, ok := bizerr.FromError(err)
	if !ok {
		return http.StatusInternalServerError
	}
	switch bErr.Code.Group() {
	case http.StatusBadRequest:
		return http.StatusBadRequest
	case http.StatusUnauthorized:
		return http.StatusUnauthorized
	case http.StatusForbidden:
		return http.StatusForbidden
	case http.StatusMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case http.StatusNotFound:
		return http.StatusNotFound
	case http.StatusConflict:
		return http.StatusConflict
	case http.StatusRequestEntityTooLarge:
		return http.StatusRequestEntityTooLarge
	case http.StatusTooManyRequests:
		return http.StatusTooManyRequests
	case http.StatusInternalServerError:
		return http.StatusInternalServerError
	case http.StatusBadGateway:
		return http.StatusBadGateway
	case http.StatusServiceUnavailable:
		return http.StatusServiceUnavailable
	case http.StatusGatewayTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
