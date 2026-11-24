package config

import "fmt"

// UserFieldConfig enables a profile field. Disclosure to other users is opt-in.
type UserFieldConfig struct {
	ShowToAll bool              `json:"showToAll"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type UserFieldPresentation struct {
	UserFieldConfig
	Editable    bool `json:"editable"`
	LDAPManaged bool `json:"ldapManaged"`
}

func UserFieldDefaults() map[string]UserFieldPresentation {
	return map[string]UserFieldPresentation{
		"nickname": {Editable: true}, "email": {Editable: true}, "identity": {Editable: true},
		"teacher": {Editable: true}, "group": {Editable: true}, "phone": {Editable: true},
		"expiredAt": {}, "createdAt": {}, "lastLoginAt": {},
	}
}

// LDAPProfileMappings excludes login identity and filesystem identifiers.
func (c *Config) LDAPProfileMappings() map[string]string {
	m := c.Auth.LDAP.AttributeMapping
	return map[string]string{"email": m.Email, "identity": m.Identity, "teacher": m.Teacher,
		"group": m.Group, "phone": m.Phone, "expiredAt": m.ExpiredAt}
}

func (c *Config) UserFieldEnabled(key string) bool { _, ok := c.UserFields[key]; return ok }
func (c *Config) UserFieldVisible(key string, privileged bool) bool {
	f, ok := c.UserFields[key]
	return ok && (privileged || f.ShowToAll)
}

func (c *Config) UserFieldPresentation() map[string]UserFieldPresentation {
	fields := make(map[string]UserFieldPresentation)
	defaults := UserFieldDefaults()
	mappings := c.LDAPProfileMappings()
	for key, override := range c.UserFields {
		field, ok := defaults[key]
		if !ok {
			continue
		}
		field.UserFieldConfig = override
		field.LDAPManaged = c.Auth.LDAP.Enable && mappings[key] != ""
		if key == "nickname" {
			field.LDAPManaged = c.Auth.LDAP.Enable && c.Auth.LDAP.AttributeMapping.DisplayName != ""
		}
		fields[key] = field
	}
	return fields
}

func (c *Config) validateUserFields() []string {
	var problems []string
	defaults := UserFieldDefaults()
	for key := range c.UserFields {
		if _, ok := defaults[key]; !ok {
			problems = append(problems, fmt.Sprintf("userFields.%s is not a supported user field", key))
		}
	}
	for key, mapping := range c.LDAPProfileMappings() {
		if mapping != "" && !c.UserFieldEnabled(key) {
			problems = append(problems, fmt.Sprintf("auth.ldap.attributeMapping.%s requires userFields.%s to be enabled", key, key))
		}
	}
	return problems
}
