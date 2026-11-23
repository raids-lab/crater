package config

import "fmt"

// UserFieldConfig overrides presentation only; it never changes authentication,
// attribute synchronization, authorization, or which data the API returns.
type UserFieldConfig struct {
	ShowInTable *bool             `json:"showInTable,omitempty"`
	ShowInForm  *bool             `json:"showInForm,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type UserFieldPresentation struct {
	ShowInTable bool              `json:"showInTable"`
	ShowInForm  bool              `json:"showInForm"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// UserFieldDefaults preserves existing columns and editable fields. New optional
// fields must be enabled explicitly, independently of the authentication source.
func UserFieldDefaults() map[string]UserFieldPresentation {
	return map[string]UserFieldPresentation{
		"nickname":    {ShowInForm: true},
		"email":       {ShowInForm: true},
		"identity":    {},
		"teacher":     {ShowInTable: true, ShowInForm: true},
		"group":       {ShowInTable: true, ShowInForm: true},
		"phone":       {ShowInForm: true},
		"expiredAt":   {},
		"createdAt":   {},
		"lastLoginAt": {},
	}
}

func (c *Config) UserFieldPresentation() map[string]UserFieldPresentation {
	fields := UserFieldDefaults()
	for key, override := range c.UserFields {
		field, ok := fields[key]
		if !ok {
			continue
		}
		if override.ShowInTable != nil {
			field.ShowInTable = *override.ShowInTable
		}
		if override.ShowInForm != nil {
			field.ShowInForm = *override.ShowInForm
		}
		field.Labels = override.Labels
		fields[key] = field
	}
	return fields
}

func (c *Config) validateUserFields() []string {
	var problems []string
	defaults := UserFieldDefaults()
	for key, field := range c.UserFields {
		if _, ok := defaults[key]; !ok {
			problems = append(problems, fmt.Sprintf("userFields.%s is not a supported user field", key))
		}
		if (key == "createdAt" || key == "lastLoginAt" || key == "expiredAt") &&
			field.ShowInForm != nil && *field.ShowInForm {
			problems = append(problems, fmt.Sprintf("userFields.%s.showInForm cannot be true for a read-only field", key))
		}
	}
	return problems
}
