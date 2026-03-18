package resputil

import (
	"errors"
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
	switch {
	case errors.Is(err, bizerr.BadRequest.Base):
		return http.StatusBadRequest
	case errors.Is(err, bizerr.Auth.Base):
		return http.StatusUnauthorized
	case errors.Is(err, bizerr.Forbidden.Base):
		return http.StatusForbidden
	case errors.Is(err, bizerr.MethodNotAllowed.Base):
		return http.StatusMethodNotAllowed
	case errors.Is(err, bizerr.NotFound.Base):
		return http.StatusNotFound
	case errors.Is(err, bizerr.Conflict.Base):
		return http.StatusConflict
	case errors.Is(err, bizerr.PayloadTooLarge.Base):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, bizerr.RateLimit.Base):
		return http.StatusTooManyRequests
	case errors.Is(err, bizerr.Internal.Base):
		return http.StatusInternalServerError
	case errors.Is(err, bizerr.BadGateway.Base):
		return http.StatusBadGateway
	case errors.Is(err, bizerr.ServiceError.Base):
		return http.StatusServiceUnavailable
	case errors.Is(err, bizerr.GatewayTimeout.Base):
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
