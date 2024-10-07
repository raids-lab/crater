package handler

import "github.com/gin-gonic/gin"

type PodContainerMgr struct {
	name string
}

func NewPodContainerMgr() Manager {
	return &PodContainerMgr{
		name: "podcontainers",
	}
}

func (mgr *PodContainerMgr) GetName() string {
	return mgr.name
}

func (mgr *PodContainerMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *PodContainerMgr) RegisterProtected(_ *gin.RouterGroup) {}

func (mgr *PodContainerMgr) RegisterAdmin(_ *gin.RouterGroup) {}
