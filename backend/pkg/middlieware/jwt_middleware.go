package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/aisystem/ai-protal/pkg/util"
	//"github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"
	//"github.com/amitshekhariitbhu/go-backend-clean-architecture/internal/tokenutil"
	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Message string `json:"error"`
	Code    int    `json:"error_code"`
}

func JwtAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		fmt.Println(authHeader)
		t := strings.Split(authHeader, " ")
		fmt.Println(len(t))
		if len(t) == 2 {
			authToken := t[1]
			authorized, err := util.IsAuthorized(authToken, secret)
			if authorized {
				userID, err := util.ExtractIDFromToken(authToken, secret)
				if err != nil {
					c.JSON(http.StatusUnauthorized, ErrorResponse{
						Message: err.Error(),
						Code:    40104,
					})
					c.Abort()
					return
				}
				c.Set("x-user-id", userID)
				c.Next()
				return
			}
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Message: err.Error(),
				Code:    40105,
			})
			c.Abort()
			return
		}
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Message: "Not authorized",
			Code:    40106,
		})
		c.Abort()
	}
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", "*") // 可将将 * 替换为指定的域名
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}
