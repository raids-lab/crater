package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/raids-lab/crater/internal/bizerr"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/resputil"
)

func AuthProtected(userBanService *service.UserBanService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var authToken string
		// if request type is websocket, read token from string
		if c.Request.Header.Get("Upgrade") == "websocket" {
			authToken = c.Query("token")
		} else {
			authHeader := c.Request.Header.Get("Authorization")
			t := strings.Fields(authHeader)
			if len(t) != 2 || !strings.EqualFold(t[0], "Bearer") {
				resputil.HTTPError(c, http.StatusUnauthorized, "Invalid token", resputil.TokenInvalid)
				c.Abort()
				return
			}
			authToken = t[1]
		}
		token, err := util.GetTokenMgr().CheckToken(authToken)
		if err != nil {
			resputil.HandleError(c, util.TokenValidationError(err))
			c.Abort()
			return
		}

		// Reload status and permissions for every request, including reads.
		u := query.User
		user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = bizerr.Auth.TokenInvalid.Wrap(err, "Account no longer exists. Sign in again.")
		} else if err != nil {
			err = bizerr.Internal.DatabaseError.Wrap(err, "Account lookup is unavailable. Retry later.")
		} else {
			err = util.ValidatePlatformUser(user)
		}
		if err != nil {
			resputil.HandleError(c, err)
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
		token.RolePlatform = user.Role
		token.Username = user.Name
		if c.Request.URL.Path != "/api/v1/auth/switch" {
			if err := refreshProjectPermissions(c, user, &token); err != nil {
				resputil.HandleError(c, err)
				c.Abort()
				return
			}
		}

		util.SetJWTContext(c, token)
		c.Next()
	}
}

func AuthAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := util.GetToken(c)
		if token.RolePlatform != model.RoleAdmin {
			resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("Administrator permission is required."))
			c.Abort()
			return
		}
		c.Next()
	}
}

func refreshProjectPermissions(c *gin.Context, user *model.User, token *util.JWTMessage) error {
	uq := query.UserAccount
	association, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID), uq.AccountID.Eq(token.AccountID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = bizerr.Forbidden.NoProjectAccess.Wrap(err, "Project access is unavailable. Contact the platform administrator.")
	} else if err != nil {
		err = bizerr.Internal.DatabaseError.Wrap(err, "Project lookup is unavailable. Retry later.")
	} else if association.AccessMode == model.AccessModeNA {
		err = bizerr.Forbidden.NoProjectAccess.New("Project access is disabled. Contact the platform administrator.")
	}
	if err != nil {
		return err
	}
	q := query.Account
	project, err := q.WithContext(c).Where(q.ID.Eq(token.AccountID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = bizerr.Forbidden.NoProjectAccess.Wrap(err, "Project no longer exists. Contact the platform administrator.")
	} else if err != nil {
		err = bizerr.Internal.DatabaseError.Wrap(err, "Project lookup is unavailable. Retry later.")
	}
	if err != nil {
		return err
	}
	token.AccountName = project.Name
	token.RoleAccount, token.AccountAccessMode = association.Role, association.AccessMode
	token.PublicAccessMode = model.AccessModeNA
	public, err := uq.WithContext(c).Where(uq.UserID.Eq(user.ID), uq.AccountID.Eq(model.DefaultAccountID)).First()
	if err == nil {
		token.PublicAccessMode = public.AccessMode
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerr.Internal.DatabaseError.Wrap(err, "Public permissions could not be loaded. Retry later.")
	}
	return nil
}
