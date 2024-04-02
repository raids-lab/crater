package handler

import "github.com/gin-gonic/gin"

type AITaskMgr struct {
}

func NewAITaskMgr() Handler {
	return &AITaskMgr{}
}

func (mgr *AITaskMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AITaskMgr) RegisterProtected(_ *gin.RouterGroup) {}

func (mgr *AITaskMgr) RegisterAdmin(_ *gin.RouterGroup) {}
