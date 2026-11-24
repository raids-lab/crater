package config

import (
	"strings"
	"testing"
)

func TestUserFieldsEnablementAndMappingValidation(t *testing.T) {
	cfg := &Config{}
	if len(cfg.UserFieldPresentation()) != 0 {
		t.Fatal("empty config must disable all fields")
	}
	cfg.UserFields = map[string]UserFieldConfig{"phone": {}, "group": {ShowToAll: true}, "createdAt": {}}
	fields := cfg.UserFieldPresentation()
	if !fields["phone"].Editable || fields["createdAt"].Editable || fields["phone"].ShowToAll {
		t.Fatal("incorrect field capabilities")
	}
	if !cfg.UserFieldVisible("phone", true) || cfg.UserFieldVisible("phone", false) || !cfg.UserFieldVisible("group", false) {
		t.Fatal("privacy must depend on enablement and audience")
	}
	cfg.Auth.LDAP.AttributeMapping.Phone = "telephoneNumber"
	if len(cfg.validateUserFields()) != 0 {
		t.Fatal("enabled mapped phone rejected")
	}
	cfg.Auth.LDAP.AttributeMapping.Identity = "title"
	problems := strings.Join(cfg.validateUserFields(), " ")
	if !strings.Contains(problems, "userFields.identity") {
		t.Fatal("missing field-specific mapping error")
	}
	cfg.UserFields["identity"] = UserFieldConfig{}
	cfg.Auth.LDAP.Enable = true
	if !cfg.UserFieldPresentation()["identity"].LDAPManaged {
		t.Fatal("missing LDAP edit warning")
	}
	cfg.UserFields["unknown"] = UserFieldConfig{}
	if len(cfg.validateUserFields()) != 1 {
		t.Fatal("unknown fields must fail validation")
	}
}
