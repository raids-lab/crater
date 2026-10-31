package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
)

const agentRuntimeConfigResponseLimit = 2 << 20

// GetAgentRuntimeConfig returns the persisted override to the Agent runtime.
func (mgr *AgentMgr) GetAgentRuntimeConfig(c *gin.Context) {
	if !isInternalAgentRequest(c) {
		return
	}
	config, found, err := mgr.runtimeConfig.Get(c.Request.Context())
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	if !found {
		resputil.Success(c, nil)
		return
	}
	resputil.Success(c, config)
}

// AdminGetAgentRuntimeConfig returns the persisted configuration or Agent defaults.
//
//	@Summary		Get Agent runtime configuration
//	@Tags			Agent
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[service.AgentRuntimeConfig]
//	@Router			/v1/admin/agent/runtime-config [get]
func (mgr *AgentMgr) AdminGetAgentRuntimeConfig(c *gin.Context) {
	config, found, err := mgr.runtimeConfig.Get(c.Request.Context())
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	if !found {
		config, err = requestDefaultAgentRuntimeConfig(c)
		if err != nil {
			resputil.HandleError(c, err)
			return
		}
	}
	resputil.Success(c, config)
}

// AdminUpdateAgentRuntimeConfig validates and stores editable Agent guidance.
//
//	@Summary		Update Agent runtime configuration
//	@Tags			Agent
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		service.AgentRuntimeConfig	true	"Agent runtime configuration"
//	@Success		200		{object}	resputil.Response[string]
//	@Router			/v1/admin/agent/runtime-config [put]
func (mgr *AgentMgr) AdminUpdateAgentRuntimeConfig(c *gin.Context) {
	var config service.AgentRuntimeConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid Agent runtime configuration"))
		return
	}
	if err := mgr.runtimeConfig.Save(c.Request.Context(), &config); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "Agent runtime configuration updated successfully")
}

func isInternalAgentRequest(c *gin.Context) bool {
	serviceToken := strings.TrimSpace(os.Getenv(service.AgentInternalTokenEnv))
	if !constantTimeTokenEqual(bearerToken(c.GetHeader("Authorization")), serviceToken) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Agent service token"})
		return false
	}
	return true
}

func requestDefaultAgentRuntimeConfig(c *gin.Context) (*service.AgentRuntimeConfig, error) {
	runtimeURL, err := buildAgentRuntimeURL(os.Getenv("CRATER_AGENT_URL"), "/runtime-config/defaults")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, runtimeURL, http.NoBody)
	if err != nil {
		return nil, bizerr.Internal.ServiceError.Wrap(err, "failed to build Agent default configuration request")
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(os.Getenv(service.AgentInternalTokenEnv)))
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, bizerr.BadGateway.BadGateway.Wrap(err, "Crater-Agent service is unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, bizerr.BadGateway.BadGateway.New("failed to load Agent default configuration")
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, agentRuntimeConfigResponseLimit))
	if err != nil {
		return nil, bizerr.BadGateway.BadGateway.Wrap(err, "failed to read Agent default configuration")
	}
	var config service.AgentRuntimeConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, bizerr.BadGateway.BadGateway.Wrap(err, "invalid Agent default configuration")
	}
	if err := service.ValidateAgentRuntimeConfig(&config); err != nil {
		return nil, err
	}
	return &config, nil
}
