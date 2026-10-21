package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
)

type agentJobListQuery struct {
	Page     int      `form:"page" binding:"omitempty,min=1"`
	PageSize int      `form:"pageSize" binding:"omitempty,min=1,max=50"`
	Search   string   `form:"search"`
	Username string   `form:"username"`
	Statuses []string `form:"status" binding:"omitempty,max=20,dive,required"`
}

// getInternalAgentClaims verifies that a request came from the Agent runtime
// and that its short-lived context was issued by this backend instance.
func getInternalAgentClaims(c *gin.Context) (*service.AgentContextClaims, bool) {
	serviceToken := strings.TrimSpace(os.Getenv(service.AgentInternalTokenEnv))
	if !constantTimeTokenEqual(bearerToken(c.GetHeader("Authorization")), serviceToken) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Agent service token"})
		return nil, false
	}
	claims, err := service.ParseAgentContext(
		c.GetHeader("X-Crater-Agent-Context"),
		strings.TrimSpace(os.Getenv(service.AgentContextSigningKeyEnv)),
	)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Agent context"})
		return nil, false
	}
	return claims, true
}

func (mgr *AgentMgr) GetAgentJobDiagnostic(c *gin.Context) {
	claims, ok := getInternalAgentClaims(c)
	if !ok {
		return
	}
	snapshot, err := mgr.diagnostics.GetSnapshot(
		c.Request.Context(), claims.UserID, claims.AccountID, claims.IdentityScope, c.Param("name"),
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, snapshot)
}

func (mgr *AgentMgr) GetAgentJobSchedulingDiagnosis(c *gin.Context) {
	claims, ok := getInternalAgentClaims(c)
	if !ok {
		return
	}
	diagnosis, err := mgr.diagnostics.GetSchedulingDiagnosis(
		c.Request.Context(), claims.UserID, claims.AccountID, claims.IdentityScope, c.Param("name"),
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, diagnosis)
}

func (mgr *AgentMgr) ListAgentJobs(c *gin.Context) {
	claims, ok := getInternalAgentClaims(c)
	if !ok {
		return
	}
	var request agentJobListQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid Agent query"))
		return
	}
	page, err := mgr.platformRead.ListJobs(c.Request.Context(), &service.AgentJobListOptions{
		Page:      request.Page,
		PageSize:  request.PageSize,
		Search:    request.Search,
		Username:  request.Username,
		Statuses:  request.Statuses,
		Identity:  claims.IdentityScope,
		UserID:    claims.UserID,
		AccountID: claims.AccountID,
	})
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, page)
}

func (mgr *AgentMgr) ListAgentNodes(c *gin.Context) {
	if _, ok := getInternalAgentClaims(c); !ok {
		return
	}
	nodes, err := mgr.platformRead.ListNodes(c.Request.Context())
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, nodes)
}

func (mgr *AgentMgr) GetAgentNodeCapacity(c *gin.Context) {
	if _, ok := getInternalAgentClaims(c); !ok {
		return
	}
	node, err := mgr.platformRead.GetNodeCapacity(c.Request.Context(), c.Param("name"))
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, node)
}

func (mgr *AgentMgr) GetAgentClusterCapacity(c *gin.Context) {
	if _, ok := getInternalAgentClaims(c); !ok {
		return
	}
	capacity, err := mgr.platformRead.GetClusterCapacity(c.Request.Context())
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, capacity)
}
