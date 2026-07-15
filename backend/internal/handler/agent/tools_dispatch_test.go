package agent

import (
	"testing"

	"gorm.io/datatypes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
)

func TestAuthorizeAgentToolForSessionRequiresAdminSurface(t *testing.T) {
	adminToken := util.JWTMessage{RolePlatform: model.RoleAdmin}
	userSurfaceSession := &model.AgentSession{
		Source:      "chat",
		PageContext: datatypes.JSON([]byte(`{"route":"/portal/jobs/detail/jpt-demo"}`)),
	}

	err := authorizeAgentToolForSession(userSurfaceSession, adminToken, agentToolCordonNode)
	if err == nil {
		t.Fatalf("expected admin tool to be denied from user surface session")
	}
}

func TestEffectiveAgentSessionTokenDowngradesAdminOnUserSurface(t *testing.T) {
	adminToken := util.JWTMessage{RolePlatform: model.RoleAdmin}
	userSurfaceSession := &model.AgentSession{
		Source:      "chat",
		PageContext: datatypes.JSON([]byte(`{"route":"/portal/jobs/detail/jpt-demo"}`)),
	}

	effectiveToken := effectiveAgentSessionToken(userSurfaceSession, adminToken)
	if effectiveToken.RolePlatform != model.RoleUser {
		t.Fatalf("expected admin token to be downgraded on user surface, got %v", effectiveToken.RolePlatform)
	}
}

func TestEffectiveAgentSessionTokenKeepsAdminOnAdminSurface(t *testing.T) {
	adminToken := util.JWTMessage{RolePlatform: model.RoleAdmin}
	adminSurfaceSession := &model.AgentSession{
		Source:      "chat",
		PageContext: datatypes.JSON([]byte(`{"url":"https://example.test/admin/nodes"}`)),
	}

	effectiveToken := effectiveAgentSessionToken(adminSurfaceSession, adminToken)
	if effectiveToken.RolePlatform != model.RoleAdmin {
		t.Fatalf("expected admin token to remain admin on admin surface, got %v", effectiveToken.RolePlatform)
	}
}

func TestAuthorizeAgentToolForSessionAllowsAdminSurface(t *testing.T) {
	adminToken := util.JWTMessage{RolePlatform: model.RoleAdmin}
	adminSurfaceSession := &model.AgentSession{
		Source:      "chat",
		PageContext: datatypes.JSON([]byte(`{"url":"https://example.test/admin/nodes"}`)),
	}

	if err := authorizeAgentToolForSession(adminSurfaceSession, adminToken, agentToolCordonNode); err != nil {
		t.Fatalf("expected admin tool to be allowed from admin surface session: %v", err)
	}
}

func TestAuthorizeAgentToolForSessionStillRequiresAdminRole(t *testing.T) {
	userToken := util.JWTMessage{RolePlatform: model.RoleUser}
	adminSurfaceSession := &model.AgentSession{
		Source:      "chat",
		PageContext: datatypes.JSON([]byte(`{"route":"/admin/nodes"}`)),
	}

	err := authorizeAgentToolForSession(adminSurfaceSession, userToken, agentToolCordonNode)
	if err == nil {
		t.Fatalf("expected admin tool to be denied without admin platform role")
	}
}

func TestAuthorizeAgentToolForSessionAllowsUserJobTools(t *testing.T) {
	userToken := util.JWTMessage{RolePlatform: model.RoleUser}
	userSurfaceSession := &model.AgentSession{
		Source:      "chat",
		PageContext: datatypes.JSON([]byte(`{"route":"/portal/jobs/detail/jpt-demo"}`)),
	}

	if err := authorizeAgentToolForSession(userSurfaceSession, userToken, agentToolStopJob); err != nil {
		t.Fatalf("expected user job tool to be allowed before per-job ownership check: %v", err)
	}
}

func TestOwnedJobMutationToolSet(t *testing.T) {
	for _, toolName := range []string{agentToolDeleteJob, agentToolStopJob, agentToolResubmitJob} {
		if !isAgentOwnedJobMutationTool(toolName) {
			t.Fatalf("expected %s to require owned-job preflight", toolName)
		}
	}
	for _, toolName := range []string{agentToolCreateJupyter, agentToolDrainNode, agentToolListUserJobs} {
		if isAgentOwnedJobMutationTool(toolName) {
			t.Fatalf("expected %s not to use owned-job preflight", toolName)
		}
	}
}
