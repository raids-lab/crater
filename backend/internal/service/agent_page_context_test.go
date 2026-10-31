package service

import "testing"

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

func TestValidateAgentPageContextArrayFieldTypes(t *testing.T) {
	t.Parallel()

	context := &AgentPageContext{
		Route:  "/portal/jobs/new/jupyter-job",
		Action: "assist-form",
		Title:  "Agent",
		Prompt: "help",
		Form: &AgentFormContext{
			CapabilityID:  "jupyter-job",
			SchemaVersion: 1,
			Title:         "Jupyter job",
			Fields: []AgentFormFieldDescriptor{
				{Path: "tags", Label: "Tags", ValueType: "option-array"},
				{Path: "volumeMounts", Label: "Volume mounts", ValueType: "volume-mount-array"},
				{Path: "forwards", Label: "Port forwards", ValueType: "forward-array"},
			},
		},
	}

	if err := ValidateAgentPageContext(context, AgentIdentityUser); err != nil {
		t.Fatalf("ValidateAgentPageContext() error = %v", err)
	}
}
