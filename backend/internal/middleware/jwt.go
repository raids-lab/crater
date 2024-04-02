package middleware

import (
	"net/http"
	"strings"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/logutils"
	resputil "github.com/raids-lab/crater/pkg/server/response"
)

func AuthProtected() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := query.User
		up := query.UserProject
		p := query.Project
		if gin.Mode() == gin.DebugMode {
			if username := c.Request.Header.Get("X-Debug-Username"); username != "" {
				// Debug 模式下，可通过 X-Debug-Username 设置用户，此时将使用默认的个人项目
				user, err := u.WithContext(c).Where(u.Name.Eq(username)).Take()
				if err != nil {
					resputil.HTTPError(c, http.StatusUnauthorized, "User not found", resputil.UserNotFound)
					c.Abort()
					return
				}
				// 获取用户的默认个人项目
				var projects []payload.ProjectResp
				err = up.WithContext(c).Where(up.UserID.Eq(user.ID)).
					Select(p.ID, p.Name, up.Role, p.IsPersonal).Join(p, p.ID.EqCol(up.ProjectID)).
					Where(p.Status.Eq(uint8(model.StatusActive)), p.IsPersonal.Is(true)).Scan(&projects)
				if err != nil {
					logutils.Log.Error(err)
					resputil.HTTPError(c, http.StatusUnauthorized, "DB query", resputil.UserNotFound)
					c.Abort()
					return
				}
				if len(projects) != 1 {
					resputil.HTTPError(c, http.StatusUnauthorized, "No matched personal project", resputil.UserNotFound)
					c.Abort()
					return
				}
				util.SetJWTContext(c, user.ID, projects[0].ID, projects[0].Role, user.Role)
				c.Next()
				return
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
		token, err := util.GetTokenMgr().CheckToken(authToken)
		if err != nil {
			resputil.HTTPError(c, http.StatusUnauthorized, err.Error(), resputil.TokenExpired)
			c.Abort()
			return
		}

		// 如果查询方法不是 GET (e.g. POST, PUT, DELETE), 从数据库中校验权限
		if c.Request.Method != "GET" {
			user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
			if err != nil {
				resputil.HTTPError(c, http.StatusUnauthorized, "User not found", resputil.UserNotFound)
				c.Abort()
				return
			}
			if user.Role != token.PlatformRole {
				resputil.HTTPError(c, http.StatusUnauthorized, "Platform role not match", resputil.InvalidRole)
				c.Abort()
				return
			}
			// 获取用户当前项目
			var projects []payload.ProjectResp
			err = up.WithContext(c).Where(up.UserID.Eq(user.ID)).
				Select(p.ID, p.Name, up.Role, p.IsPersonal).Join(p, p.ID.EqCol(up.ProjectID)).
				Where(p.Status.Eq(uint8(model.StatusActive)), p.ID.Eq(token.ProjectID)).Scan(&projects)
			if err != nil {
				logutils.Log.Error(err)
				resputil.HTTPError(c, http.StatusUnauthorized, "DB query", resputil.UserNotFound)
				c.Abort()
				return
			}
			if len(projects) != 1 {
				resputil.HTTPError(c, http.StatusUnauthorized, "No matched project", resputil.UserNotFound)
				c.Abort()
				return
			}
			if projects[0].Role != token.ProjectRole {
				resputil.HTTPError(c, http.StatusUnauthorized, "Project role not match", resputil.InvalidRole)
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// If request method is GET, use the user info from token.
		util.SetJWTContext(c, token.UserID, token.ProjectID, token.ProjectRole, token.PlatformRole)
		c.Next()
	}
}

func AuthAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := util.GetToken(c)
		if token.ClusterRole != model.RoleAdmin {
			resputil.HTTPError(c, http.StatusUnauthorized, "Not authorized", resputil.InvalidRole)
			c.Abort()
			return
		}
		c.Next()
	}
}
