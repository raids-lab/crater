package util

import (
	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
)

const (
	UserIDKey   = "x-user-id"
	UsernameKey = "x-user-name"

	AccountIDKey   = "x-queue-id"
	AccountNameKey = "x-queue-name"

	RoleAccountKey  = "x-role-queue"
	RolePlatformKey = "x-role-platform"

	AccountAccessModeKey = "x-access-mode"
	PublicAccessModeKey  = "x-public-access-mode"
)

const (
	QueueNameNull = ""
	QueueIDNull   = 0
)

func SetJWTContext(
	c *gin.Context,
	msg JWTMessage,
) {
	c.Set(UserIDKey, msg.UserID)
	c.Set(UsernameKey, msg.Username)

	c.Set(AccountIDKey, msg.AccountID)
	c.Set(AccountNameKey, msg.AccountName)

	c.Set(RoleAccountKey, msg.RoleAccount)
	c.Set(RolePlatformKey, msg.RolePlatform)
	c.Set(AccountAccessModeKey, msg.AccountAccessMode)
	c.Set(PublicAccessModeKey, msg.PublicAccessMode)
}

func GetToken(ctx *gin.Context) JWTMessage {
	var msg JWTMessage
	msg.UserID = ctx.GetUint(UserIDKey)
	msg.Username = ctx.GetString(UsernameKey)

	msg.AccountID = ctx.GetUint(AccountIDKey)
	msg.AccountName = ctx.GetString(AccountNameKey)

	roleQueue, _ := ctx.Get(RoleAccountKey)
	msg.RoleAccount = roleQueue.(model.Role)

	rolePlatform, _ := ctx.Get(RolePlatformKey)
	msg.RolePlatform = rolePlatform.(model.Role)
	accessModeKey, _ := ctx.Get(AccountAccessModeKey)
	msg.AccountAccessMode = accessModeKey.(model.AccessMode)
	publicAcessModeKey, _ := ctx.Get(PublicAccessModeKey)
	msg.PublicAccessMode = publicAcessModeKey.(model.AccessMode)
	return msg
}
