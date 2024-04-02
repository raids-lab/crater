package handler

import "github.com/gin-gonic/gin"

type UserMgr struct {
}

func NewUserMgr() Handler {
	return &UserMgr{}
}

func (mgr *UserMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *UserMgr) RegisterProtected(_ *gin.RouterGroup) {}

func (mgr *UserMgr) RegisterAdmin(_ *gin.RouterGroup) {}
