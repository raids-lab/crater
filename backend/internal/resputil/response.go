package resputil

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
)

// Response 用于 swagger 生成文档
type Response[T any] struct {
	Code    bizerr.BizCode `json:"code"` // 依然保持 int (ErrorCode) 类型
	Data    T              `json:"data"`
	Message string         `json:"msg"`
}

// emit 内部统一发送方法，接收基础 ErrorCode
func emit(c *gin.Context, httpCode int, bizCode bizerr.BizCode, msg string, data any) {
	c.JSON(httpCode, Response[any]{
		Code:    bizCode,
		Data:    data,
		Message: msg,
	})
}

const OK bizerr.BizCode = 0

// Success 200 OK - 常规查询、修改成功
func Success(c *gin.Context, data any) {
	emit(c, http.StatusOK, OK, "success", data)
}

func respond(c *gin.Context, httpCode int, bErr *bizerr.BizError) {
	emit(c, httpCode, bErr.Code, bErr.Message, nil)
}

// =============================================================================
// 兼容旧版本 / 逃生舱
// =============================================================================

type AnyErrorCode interface {
	~int
}

// Deprecated: 请使用 HandleError 方法配合 bizerr 包的错误类型来处理错误响应
func Error[T AnyErrorCode](c *gin.Context, msg string, errorCode T) {
	// 统一转为 500 处理，并将泛型 T 强转回 ErrorCode
	emit(c, http.StatusInternalServerError, bizerr.BizCode(errorCode), msg, nil)
}

// Deprecated: 请使用 HandleError 方法配合 bizerr 包的错误类型来处理错误响应
func HTTPError[T AnyErrorCode](c *gin.Context, httpCode int, err string, errorCode T) {
	emit(c, httpCode, bizerr.BizCode(errorCode), err, nil)
}

// Deprecated: 请使用 HandleError 方法配合 bizerr 包的错误类型来处理错误响应
func BadRequestError(c *gin.Context, err string) {
	// 这里硬编码使用 InvalidRequest，因为它已经是 BizCode400 了
	respond(c, http.StatusBadRequest, &bizerr.BizError{
		Code:    InvalidRequest,
		Message: err,
	})
}
