package service

import (
	"encoding/json"
	"strings"

	"github.com/raids-lab/crater/internal/bizerr"
)

const (
	agentMaximumContextFields     = 64
	agentMaximumContextText       = 512
	agentMaximumContextValueBytes = 8 << 10
	agentMaximumContextTotalBytes = 32 << 10
)

var validAgentActions = map[string]struct{}{
	"choose-job-type":        {},
	"choose-template":        {},
	"assist-form":            {},
	"find-platform-resource": {},
	"recreate-job":           {},
	"diagnose-job":           {},
}

var validAgentFieldTypes = map[string]struct{}{
	"string":             {},
	"number":             {},
	"boolean":            {},
	"string-array":       {},
	"option-array":       {},
	"volume-mount-array": {},
	"forward-array":      {},
	"json":               {},
}

type AgentPageContext struct {
	Route    string                `json:"route"`
	Action   string                `json:"action"`
	Title    string                `json:"title"`
	Prompt   string                `json:"prompt"`
	Form     *AgentFormContext     `json:"form,omitempty"`
	Resource *AgentResourceContext `json:"resource,omitempty"`
}

// AgentResourceContext describes the current page resource without granting
// the runtime access to arbitrary platform data. Authorization remains in the
// dedicated backend diagnostic endpoints.
type AgentResourceContext struct {
	Kind    string `json:"kind"`
	JobName string `json:"jobName"`
	Status  string `json:"status,omitempty"`
	JobType string `json:"jobType,omitempty"`
}

type AgentFormContext struct {
	CapabilityID  string                     `json:"capabilityId"`
	SchemaVersion uint                       `json:"schemaVersion"`
	Title         string                     `json:"title"`
	Fields        []AgentFormFieldDescriptor `json:"fields"`
	CurrentValues map[string]json.RawMessage `json:"currentValues"`
}

type AgentFormFieldDescriptor struct {
	Path        string `json:"path"`
	Label       string `json:"label"`
	ValueType   string `json:"valueType"`
	Description string `json:"description,omitempty"`
}

func ValidateAgentPageContext(context *AgentPageContext, identity AgentIdentity) error {
	if context == nil {
		return nil
	}
	context.Route = strings.TrimSpace(context.Route)
	context.Action = strings.TrimSpace(context.Action)
	context.Title = strings.TrimSpace(context.Title)
	context.Prompt = strings.TrimSpace(context.Prompt)
	expectedPrefix := "/portal/"
	if identity == AgentIdentityAdmin {
		expectedPrefix = "/admin/"
	}
	if !strings.HasPrefix(context.Route, expectedPrefix) || len(context.Route) > agentMaximumContextText {
		return bizerr.BadRequest.ParameterError.New("Agent page route is invalid")
	}
	if _, valid := validAgentActions[context.Action]; !valid {
		return bizerr.BadRequest.ParameterError.New("Agent page action is invalid")
	}
	if context.Title == "" || len(context.Title) > agentMaximumContextText ||
		context.Prompt == "" || len(context.Prompt) > agentMaximumContextText {
		return bizerr.BadRequest.ParameterError.New("Agent page description is invalid")
	}
	if context.Action == "diagnose-job" && context.Resource == nil {
		return bizerr.BadRequest.ParameterError.New("Agent diagnostic resource is required")
	}
	if context.Form != nil {
		if err := validateAgentFormContext(context.Form); err != nil {
			return err
		}
	}
	if context.Resource != nil {
		return validateAgentResourceContext(context.Resource)
	}
	return nil
}

func validateAgentResourceContext(resource *AgentResourceContext) error {
	resource.Kind = strings.TrimSpace(resource.Kind)
	resource.JobName = strings.TrimSpace(resource.JobName)
	resource.Status = strings.TrimSpace(resource.Status)
	resource.JobType = strings.TrimSpace(resource.JobType)
	if resource.Kind != "job" || resource.JobName == "" || len(resource.JobName) > agentMaximumContextText ||
		len(resource.Status) > agentMaximumContextText || len(resource.JobType) > agentMaximumContextText {
		return bizerr.BadRequest.ParameterError.New("Agent page resource is invalid")
	}
	return nil
}

func validateAgentFormContext(form *AgentFormContext) error {
	form.CapabilityID = strings.TrimSpace(form.CapabilityID)
	form.Title = strings.TrimSpace(form.Title)
	if !validAgentFormHeader(form) {
		return bizerr.BadRequest.ParameterError.New("Agent form capability is invalid")
	}
	paths := make(map[string]struct{}, len(form.Fields))
	for index := range form.Fields {
		field := &form.Fields[index]
		field.Path = strings.TrimSpace(field.Path)
		field.Label = strings.TrimSpace(field.Label)
		field.Description = strings.TrimSpace(field.Description)
		if !validAgentFormField(field) {
			return bizerr.BadRequest.ParameterError.New("Agent form field is invalid")
		}
		if _, duplicate := paths[field.Path]; duplicate {
			return bizerr.BadRequest.ParameterError.New("Agent form field is duplicated")
		}
		paths[field.Path] = struct{}{}
	}
	return validateAgentCurrentValues(form.CurrentValues, paths)
}

func validAgentFormHeader(form *AgentFormContext) bool {
	return form.CapabilityID != "" && len(form.CapabilityID) <= 128 && form.SchemaVersion > 0 &&
		form.Title != "" && len(form.Title) <= agentMaximumContextText &&
		len(form.Fields) > 0 && len(form.Fields) <= agentMaximumContextFields
}

func validAgentFormField(field *AgentFormFieldDescriptor) bool {
	_, validType := validAgentFieldTypes[field.ValueType]
	return field.Path != "" && len(field.Path) <= 128 && field.Label != "" &&
		len(field.Label) <= agentMaximumContextText && len(field.Description) <= agentMaximumContextText && validType
}

func validateAgentCurrentValues(
	currentValues map[string]json.RawMessage,
	paths map[string]struct{},
) error {
	totalBytes := 0
	for path, value := range currentValues {
		totalBytes += len(value)
		if _, allowed := paths[path]; !allowed || !json.Valid(value) ||
			len(value) > agentMaximumContextValueBytes || totalBytes > agentMaximumContextTotalBytes {
			return bizerr.BadRequest.ParameterError.New("Agent form current value is invalid")
		}
	}
	return nil
}
