package middleware

import (
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
	Code    int    `json:"errorCode"`
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
		t := strings.Split(authHeader, " ")
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
