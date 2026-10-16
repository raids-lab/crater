package middleware

import (
	"net/http"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/resputil"
)

func AuthProtected(userBanService *service.UserBanService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken, err := protectedRequestToken(c)
		if err != nil {
			resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("Invalid token"))
			c.Abort()
			return
		}
		token, err := util.GetTokenMgr().CheckToken(authToken)
		if err != nil {
			resputil.HTTPError(c, http.StatusUnauthorized, err.Error(), resputil.TokenExpired)
			c.Abort()
			return
		}
		if userBanService != nil {
			if err := userBanService.RequireCapability(
				c.Request.Context(),
				token.UserID,
				service.UserBanCapabilityPlatformAccess,
			); err != nil {
				resputil.HandleError(c, err)
				c.Abort()
				return
			}
		}

		// If query method is not GET (e.g. POST, PUT, DELETE), validate permissions from database
		if requiresFreshPermissions(c) {
			u := query.User

			// Check platform role
			user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
			if err != nil {
				resputil.HTTPError(c, http.StatusUnauthorized, "User not found", resputil.TokenInvalid)
				c.Abort()
				return
			}
			if user.Role != token.RolePlatform {
				resputil.HTTPError(c, http.StatusUnauthorized, "Platform token not match", resputil.TokenInvalid)
				c.Abort()
				return
			}

			// Skip account-level permission check for /api/v1/auth/switch endpoint
			// This endpoint is used to refresh tokens when permissions change, so it should be allowed
			// even if the current token's account permissions are outdated
			if c.Request.URL.Path != "/api/v1/auth/switch" {
				uq := query.UserAccount
				userQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID), uq.AccountID.Eq(token.AccountID)).First()
				if err != nil {
					resputil.HTTPError(c, http.StatusUnauthorized, "UserQueue not found", resputil.TokenInvalid)
					c.Abort()
					return
				}
				if userQueue.Role != token.RoleAccount || userQueue.AccessMode != token.AccountAccessMode {
					resputil.HTTPError(c, http.StatusUnauthorized, "Queue role not match", resputil.TokenInvalid)
					c.Abort()
					return
				}
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
			resputil.HTTPError(c, http.StatusUnauthorized, "Not Admin", resputil.TokenInvalid)
			c.Abort()
			return
		}
		c.Next()
	}
}
