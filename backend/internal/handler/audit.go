package handler

import (
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

// RecordOperationLog 记录操作日志的辅助函数
func RecordOperationLog(c *gin.Context, opType, target, status, message string, details map[string]interface{}) {
	operator, role := GetOperatorInfo(c)

	// Convert map to datatypes.JSON
	jsonDetails := datatypes.JSON("{}")
	if len(details) > 0 {
		importJson, err := json.Marshal(details)
		if err == nil {
			jsonDetails = datatypes.JSON(importJson)
		} else {
			klog.Errorf("Failed to marshal details for operation log: %v", err)
		}
	}

	err := service.OpLog.Create(c, operator, role, opType, target, jsonDetails, status, message)
	if err != nil {
		klog.Errorf("Failed to create operation log: %v", err)
	}
}

// GetOperatorInfo 从 Context 中提取操作人和角色
func GetOperatorInfo(c *gin.Context) (string, string) {
	claims := util.GetToken(c)
	ctx := c.Request.Context()

	role := "unknown"
	if claims.RolePlatform != 0 {
		role = claims.RolePlatform.String()
	}

	operator := "unknown"

	if claims.UserID != 0 {
		u := query.User
		user, err := u.WithContext(ctx).
			Select(u.Nickname, u.Name).
			Where(u.ID.Eq(claims.UserID)).
			First()
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				klog.Errorf("failed to fetch operator profile: %v", err)
			}
		} else {
			if user.Nickname != "" {
				operator = user.Nickname
			} else if user.Name != "" {
				operator = user.Name
			}
		}
	}

	if operator == "unknown" && claims.Username != "" {
		operator = claims.Username
	}

	return operator, role
}
