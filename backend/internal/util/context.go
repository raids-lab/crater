package util

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
)

const (
	UserIDKey       = "x-user-id"
	ProjectIDKey    = "x-project-id"
	ProjectRoleKey  = "x-project-role"
	PlatformRoleKey = "x-platform-role"
)

type ReqContext struct {
	UserID       uint
	ProjectID    uint
	ProjectRole  model.Role
	PlatformRole model.Role
}

func SetJWTContext(
	c *gin.Context,
	userID uint,
	projectID uint,
	projectRole model.Role,
	platformRole model.Role,
) {
	c.Set(UserIDKey, userID)
	c.Set(ProjectIDKey, projectID)
	c.Set(ProjectRoleKey, projectRole)
	c.Set(PlatformRoleKey, platformRole)
}

func GetToken(ctx *gin.Context) (ReqContext, error) {
	userID, ok := ctx.Get(UserIDKey)
	if !ok {
		return ReqContext{}, fmt.Errorf("user id not found")
	}

	projectID, ok := ctx.Get(ProjectIDKey)
	if !ok {
		return ReqContext{}, fmt.Errorf("project id not found")
	}

	projectRole, ok := ctx.Get(ProjectRoleKey)
	if !ok {
		return ReqContext{}, fmt.Errorf("project role not found")
	}

	platformRole, ok := ctx.Get(PlatformRoleKey)
	if !ok {
		return ReqContext{}, fmt.Errorf("platform role not found")
	}

	return ReqContext{
		UserID:       userID.(uint),
		ProjectID:    projectID.(uint),
		ProjectRole:  projectRole.(model.Role),
		PlatformRole: platformRole.(model.Role),
	}, nil
}

// 当需要通过 Restful API 传递数据库 ID 时，使用此函数解析参数.
// 例如：`/aijobs/1`，解析出的 ID 为 uint 类型的 `1`.
func GetParamID(c *gin.Context, key string) (uint, error) {
	param := c.Param(key)
	paramUint, err := strconv.ParseUint(param, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s to uint: %w", key, err)
	}
	return uint(paramUint), nil
}
