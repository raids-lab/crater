package middleware

import (
	"net/http"
	"strings"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/resputil"
)

func AuthProtected() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		t := strings.Split(authHeader, " ")
		if len(t) < 2 || t[0] != "Bearer" {
			resputil.HTTPError(c, http.StatusUnauthorized, "Invalid token", resputil.TokenExpired)
			c.Abort()
			return
		}

		authToken := t[1]
		token, err := util.GetTokenMgr().CheckToken(authToken)
		if err != nil {
			resputil.HTTPError(c, http.StatusUnauthorized, err.Error(), resputil.TokenExpired)
			c.Abort()
			return
		}

		// 如果查询方法不是 GET (e.g. POST, PUT, DELETE), 从数据库中校验权限
		if c.Request.Method != "GET" {
			u := query.User
			uq := query.UserQueue

			// check platform role
			user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
			if err != nil {
				resputil.HTTPError(c, http.StatusUnauthorized, "User not found", resputil.TokenExpired)
				c.Abort()
				return
			}
			if user.Role != token.RolePlatform || user.AccessMode != token.PublicAccessMode {
				resputil.HTTPError(c, http.StatusUnauthorized, "Platform token not match", resputil.TokenExpired)
				c.Abort()
				return
			}

			userQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID), uq.QueueID.Eq(token.QueueID)).First()
			if err != nil {
				resputil.HTTPError(c, http.StatusUnauthorized, "UserQueue not found", resputil.TokenExpired)
				c.Abort()
				return
			}
			if userQueue.Role != token.RoleQueue || userQueue.AccessMode != token.AccessMode {
				resputil.HTTPError(c, http.StatusUnauthorized, "Queue role not match", resputil.TokenExpired)
				c.Abort()
				return
			}
		}

		// If request method is GET, use the user info from token.
		util.SetJWTContext(c, token)
		c.Next()
	}
}

func AuthAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := util.GetToken(c)
		if token.RolePlatform != model.RoleAdmin {
			resputil.HTTPError(c, http.StatusUnauthorized, "Not Admin", resputil.TokenExpired)
			c.Abort()
			return
		}
		c.Next()
	}
}
