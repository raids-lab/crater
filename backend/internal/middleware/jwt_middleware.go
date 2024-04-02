package middleware

import (
	"net/http"
	"strings"

	usersvc "github.com/raids-lab/crater/pkg/db/user"
	"github.com/raids-lab/crater/pkg/util"

	"github.com/gin-gonic/gin"
	resputil "github.com/raids-lab/crater/pkg/server/response"
)

var (
	userDB = usersvc.NewDBService()
)

func JwtAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if gin.Mode() == gin.DebugMode {
			if username := c.Request.Header.Get("X-Debug-Username"); username != "" {
				if user, err := userDB.GetByUserName(username); err == nil {
					c.Set(util.UserNameKey, user.UserName)
					c.Set(util.UserRoleKey, user.Role)
					c.Set(util.NamespaceKey, user.NameSpace)
					c.Next()
					return
				}
			}
		}

		authHeader := c.Request.Header.Get("Authorization")
		t := strings.Split(authHeader, " ")
		if len(t) < 2 || t[0] != "Bearer" {
			resputil.HTTPError(c, http.StatusUnauthorized, "Invalid token", resputil.InvalidToken)
			c.Abort()
			return
		}

		authToken := t[1]
		user, err := util.CheckAndGetUser(authToken, secret)
		if err != nil {
			resputil.HTTPError(c, http.StatusUnauthorized, err.Error(), resputil.TokenExpired)
			c.Abort()
			return
		}

		// If request method is not GET, check user info from DB.
		if c.Request.Method != "GET" {
			user, err := userDB.GetByUserName(user.UserName)
			if err != nil {
				resputil.HTTPError(c, http.StatusUnauthorized, "User not found", resputil.UserNotFound)
				c.Abort()
				return
			}
			c.Set(util.UserNameKey, user.UserName)
			c.Set(util.UserRoleKey, user.Role)
			c.Set(util.NamespaceKey, user.NameSpace)
			c.Next()
			return
		}

		// If request method is GET, use the user info from token.
		c.Set(util.UserNameKey, user.UserName)
		c.Set(util.UserRoleKey, user.Role)
		c.Set(util.NamespaceKey, user.NameSpace)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userContext, _ := util.GetUserFromGinContext(c)
		if userContext.UserRole != "admin" {
			resputil.HTTPError(c, http.StatusUnauthorized, "Not authorized", resputil.InvalidRole)
			c.Abort()
			return
		}
		c.Next()
	}
}
