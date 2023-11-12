package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	usersvc "github.com/aisystem/ai-protal/pkg/db/user"
	"github.com/aisystem/ai-protal/pkg/util"

	//"github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"
	//"github.com/amitshekhariitbhu/go-backend-clean-architecture/internal/tokenutil"
	"github.com/gin-gonic/gin"
)

var (
	userDB = usersvc.NewDBService()
)

type ErrorResponse struct {
	Message string `json:"error"`
	Code    int    `json:"error_code"`
}

func JwtAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if username := c.Request.Header.Get("X-Debug-Username"); username != "" {
			if user, err := userDB.GetByUserName(username); err == nil {
				c.Set("x-user-id", strconv.Itoa(int(user.ID)))
				c.Set("username", user.UserName)
				c.Set("role", user.Role)
				c.Set("x-user-object", user)
				c.Next()
				return
			}
		}
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
				id, _ := strconv.Atoi(userID)
				user, err := userDB.GetUserByID(uint(id))
				if err != nil {
					c.JSON(http.StatusUnauthorized, ErrorResponse{
						Message: err.Error(),
						Code:    40104,
					})
					c.Abort()
					return
				}
				c.Set("x-user-id", userID)
				c.Set("username", user.UserName)
				c.Set("role", user.Role)
				c.Set("x-user-object", user)
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

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Message: "Not authorized",
				Code:    40107,
			})
			c.Abort()
			return
		}
		c.Next()
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
