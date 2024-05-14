package util

import (
	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
)

const (
	UserIDKey   = "x-user-id"
	UsernameKey = "x-user-name"

	QueueIDKey   = "x-queue-id"
	QueueNameKey = "x-queue-name"

	RoleQueueKey    = "x-role-queue"
	RolePlatformKey = "x-role-platform"

	AccessModeKey       = "x-access-mode"
	PublicAccessModeKey = "x-public-access-mode"
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

	c.Set(QueueIDKey, msg.QueueID)
	c.Set(QueueNameKey, msg.QueueName)

	c.Set(RoleQueueKey, msg.RoleQueue)
	c.Set(RolePlatformKey, msg.RolePlatform)
	c.Set(AccessModeKey, msg.AccessMode)
	c.Set(PublicAccessModeKey, msg.PublicAccessMode)
}

func GetToken(ctx *gin.Context) JWTMessage {
	var msg JWTMessage
	msg.UserID = ctx.GetUint(UserIDKey)
	msg.Username = ctx.GetString(UsernameKey)

	msg.QueueID = ctx.GetUint(QueueIDKey)
	msg.QueueName = ctx.GetString(QueueNameKey)

	roleQueue, _ := ctx.Get(RoleQueueKey)
	msg.RoleQueue = roleQueue.(model.Role)

	rolePlatform, _ := ctx.Get(RolePlatformKey)
	msg.RolePlatform = rolePlatform.(model.Role)
	accessModeKey, _ := ctx.Get(AccessModeKey)
	msg.AccessMode = accessModeKey.(model.AccessMode)
	publicAcessModeKey, _ := ctx.Get(PublicAccessModeKey)
	msg.PublicAccessMode = publicAcessModeKey.(model.AccessMode)
	return msg
}
