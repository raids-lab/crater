package config

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/utils/ptr"
)

func TestUserFieldsReadConfigCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name   string
		yaml   string
		custom bool
	}{
		{name: "old configuration", yaml: "auth:\n  normal:\n    allowLogin: true\n"},
		{name: "empty overrides", yaml: "userFields: {}\n"},
		{name: "explicit overrides", custom: true, yaml: `userFields:
  teacher:
    showInTable: false
    labels:
      en: Supervisor
  identity:
    showInTable: true
    showInForm: true
  phone:
    showInForm: false
  lastLoginAt:
    showInTable: true
auth:
  ldap:
    attributeMapping:
      identity: title
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			var cfg Config
			if err := readConfig(path, &cfg); err != nil {
				t.Fatal(err)
			}
			if problems := cfg.validateUserFields(); len(problems) != 0 {
				t.Fatalf("valid YAML was rejected: %v", problems)
			}
			fields := cfg.UserFieldPresentation()
			if fields["teacher"].ShowInTable == tc.custom ||
				fields["identity"].ShowInTable != tc.custom ||
				fields["identity"].ShowInForm != tc.custom ||
				fields["phone"].ShowInForm == tc.custom ||
				fields["lastLoginAt"].ShowInTable != tc.custom {
				t.Fatal("loaded configuration did not preserve defaults or explicit false/true overrides")
			}
			if tc.custom && (fields["teacher"].Labels["en"] != "Supervisor" ||
				cfg.Auth.LDAP.AttributeMapping.Identity != "title") {
				t.Fatal("display labels or independent LDAP mapping were lost while loading YAML")
			}
		})
	}
}

func TestUserFieldsDefaultsAndAuthenticationIndependence(t *testing.T) {
	cfg := &Config{}
	cfg.UserFields = map[string]UserFieldConfig{
		"teacher":  {ShowInTable: ptr.To(false), Labels: map[string]string{"zh": "直属领导", "en": "Supervisor"}},
		"identity": {ShowInTable: ptr.To(true), ShowInForm: ptr.To(true)},
	}
	for _, ldapEnabled := range []bool{false, true} {
		cfg.Auth.LDAP.Enable = ldapEnabled
		fields := cfg.UserFieldPresentation()
		if fields["teacher"].ShowInTable || !fields["teacher"].ShowInForm || fields["teacher"].Labels["en"] != "Supervisor" {
			t.Fatal("explicit table visibility or labels were lost, or form default changed")
		}
		if !fields["identity"].ShowInTable || !fields["identity"].ShowInForm || !fields["group"].ShowInTable || !fields["email"].ShowInForm {
			t.Fatal("field defaults/overrides depend on authentication")
		}
		if fields["createdAt"].ShowInForm || fields["lastLoginAt"].ShowInForm || fields["phone"].ShowInTable {
			t.Fatal("read-only fields or existing table defaults changed")
		}
	}
	if len(cfg.validateUserFields()) != 0 {
		t.Fatal("valid configuration rejected")
	}
	cfg.UserFields["unknown"] = UserFieldConfig{}
	cfg.UserFields["lastLoginAt"] = UserFieldConfig{ShowInForm: ptr.To(true)}
	if len(cfg.validateUserFields()) != 2 {
		t.Fatal("unknown and read-only field overrides must be rejected")
	}
}
