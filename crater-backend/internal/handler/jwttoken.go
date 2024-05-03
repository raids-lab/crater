package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

type JWTTokenMgr struct {
}

func NewJWTTokenMgr() Manager {
	return &JWTTokenMgr{}
}
func (mgr *JWTTokenMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *JWTTokenMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/verify", mgr.VerifyToken)
}

func (mgr *JWTTokenMgr) RegisterAdmin(_ *gin.RouterGroup) {}

type TokenReq struct {
	UserID     uint           `json:"userId"`
	RootPath   string         `json:"rootPath"`
	Permission FilePermission `json:"permission"`
}
type FilePermission int

const (
	_ FilePermission = iota
	NotAllowed
	ReadOnly
	ReadWrite
)

// VerifyToken godoc
// @Summary 通过token鉴权
// @Description 读取header的auth进行鉴权
// @Tags Token
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[TokenReq] "Token 鉴权"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router  /v1/token/verify [get]
func (mgr *JWTTokenMgr) VerifyToken(c *gin.Context) {
	token := util.GetToken(c)
	uq := query.UserQueue
	q := query.Queue
	userQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(token.UserID), uq.QueueID.Eq(token.QueueID)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	queue, err := q.WithContext(c).Where(q.ID.Eq(token.QueueID)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	data := TokenReq{
		UserID:     token.UserID,
		Permission: FilePermission(userQueue.Role),
		RootPath:   queue.Space,
	}
	resputil.Success(c, data)
}
