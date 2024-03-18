package middleware

import (
	"net/http"
	"strconv"
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
					c.Set("x-user-id", strconv.Itoa(int(user.ID)))
					c.Set("x-user-name", user.UserName)
					c.Set("x-user-role", user.Role)
					c.Set("x-namespace", user.NameSpace)
					c.Next()
					return
				}
			}
		}

		authHeader := c.Request.Header.Get("Authorization")
		t := strings.Split(authHeader, " ")
		if len(t) == 2 {
			authToken := t[1]
			user, err := util.CheckAndGetUser(authToken, secret)
			if err != nil {
				resputil.HttpError(c, http.StatusUnauthorized, err.Error(), resputil.TokenExpired)
				c.Abort()
				return
			}
			c.Set("x-user-id", user.ID)
			c.Set("x-user-name", user.UserName)
			c.Set("x-user-role", user.Role)
			c.Set("x-namespace", user.NameSpace)
			c.Next()
			return
		}
		resputil.HttpError(c, http.StatusUnauthorized, "Not authorized", 40106)
		c.Abort()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("x-user-role")
		if role != "admin" {
			resputil.HttpError(c, http.StatusUnauthorized, "Not authorized", resputil.NotAdmin)
			c.Abort()
			return
		}
		c.Next()
	}
}
