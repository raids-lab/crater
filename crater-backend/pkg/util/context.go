package util

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

const (
	UserIDKey    = "x-user-id"
	UserNameKey  = "x-user-name"
	UserRoleKey  = "x-user-role"
	NamespaceKey = "x-namespace"
)

// return type
type UserContext struct {
	UserID    string
	UserName  string
	UserRole  string
	Namespace string
}

// GetUserFromGinContext retrieves user information from the given Gin context.
// It expects the following keys to be set in the context:
// - UserIDKey: the key for the user ID
// - UserNameKey: the key for the user name
// - NamespaceKey: the key for the namespace
// - UserRoleKey: the key for the user role
// If any of the keys are missing, an error is returned.
// The function returns a UserContext struct containing the retrieved user information.
func GetUserFromGinContext(ctx *gin.Context) (UserContext, error) {
	userID, exists := ctx.Get(UserIDKey)
	if !exists {
		return UserContext{}, fmt.Errorf("user id not found in context")
	}
	userName, exists := ctx.Get(UserNameKey)
	if !exists {
		return UserContext{}, fmt.Errorf("user name not found in context")
	}
	namespace, exists := ctx.Get(NamespaceKey)
	if !exists {
		return UserContext{}, fmt.Errorf("namespace not found in context")
	}
	userRole, exists := ctx.Get(UserRoleKey)
	if !exists {
		return UserContext{}, fmt.Errorf("user role not found in context")
	}
	return UserContext{
		UserID:    userID.(string),
		UserName:  userName.(string),
		UserRole:  userRole.(string),
		Namespace: namespace.(string),
	}, nil
}
