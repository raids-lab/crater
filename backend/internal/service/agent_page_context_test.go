package service

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateAgentPageContextIdentityRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		identity AgentIdentity
		route    string
		wantErr  bool
	}{
		{name: "user portal", identity: AgentIdentityUser, route: "/portal/jobs/inter"},
		{name: "user admin denied", identity: AgentIdentityUser, route: "/admin/jobs", wantErr: true},
		{name: "admin admin", identity: AgentIdentityAdmin, route: "/admin/jobs"},
		{name: "admin portal denied", identity: AgentIdentityAdmin, route: "/portal/jobs/inter", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			context := &AgentPageContext{
				Route:  test.route,
				Action: "assist-form",
				Title:  "Agent",
				Prompt: "help",
			}
			err := ValidateAgentPageContext(context, test.identity)
			if test.wantErr && err == nil {
				t.Fatal("ValidateAgentPageContext() error = nil, want error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("ValidateAgentPageContext() error = %v", err)
			}
		})
	}
}

func TestValidateAgentPageContextFormOptions(t *testing.T) {
	t.Parallel()
	context := &AgentPageContext{
		Route: "/portal/jobs/new/jupyter-job", Action: "assist-form", Title: "Jupyter", Prompt: "help",
		Form: &AgentFormContext{
			InstanceID: uuid.NewString(), CapabilityID: "job.jupyter", SchemaVersion: 1, Title: "Jupyter",
			Fields: []AgentFormFieldDescriptor{{Path: "task.resource.gpu.model", Label: "GPU", ValueType: "string"}},
			Options: map[string]AgentFormOptionSet{
				"task.resource.gpu.model": {
					Items: []AgentFormOption{{Value: "nvidia.com/v100-32gb", Label: "NVIDIA V100 32GB"}},
					Total: 1,
				},
			},
		},
	}
	if err := ValidateAgentPageContext(context, AgentIdentityUser); err != nil {
		t.Fatalf("valid options rejected: %v", err)
	}
	context.Form.Options["task.resource.gpu.model"] = AgentFormOptionSet{
		Items: []AgentFormOption{{Value: "nvidia.com/v100-32gb", Label: "V100"}}, Total: 0,
	}
	if err := ValidateAgentPageContext(context, AgentIdentityUser); err == nil {
		t.Fatal("truncated option total smaller than item count was accepted")
	}
}
