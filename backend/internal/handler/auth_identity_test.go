// Copyright 2026 The Crater Project Team, RAIDS-Lab
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"testing"

	ldap "github.com/go-ldap/ldap/v3"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/config"
)

func TestAuthenticationIdentityMapping(t *testing.T) {
	mgr := &AuthMgr{}
	for _, tc := range []struct {
		name, mapping string
		values        map[string][]string
		want          *string
	}{
		{"unconfigured", "", map[string][]string{"title": {"PhD student"}}, nil},
		{"configured", "title", map[string][]string{"title": {"PhD student"}}, ptr.To("PhD student")},
		{"empty", "title", map[string][]string{"title": {""}}, ptr.To("")},
		{"missing", "title", nil, ptr.To("")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Auth.LDAP.AttributeMapping.Identity = tc.mapping
			requested := false
			for _, attr := range mgr.prepareLDAPAttributes(cfg) {
				requested = requested || attr == "title"
			}
			if requested != (tc.mapping != "") {
				t.Fatal("LDAP search did not respect the identity mapping")
			}
			var profile model.UserAttribute
			mgr.populateUserAttributes(cfg, "test-user", ldap.NewEntry("cn=test-user", tc.values), &profile)
			if (profile.Identity == nil) != (tc.want == nil) ||
				(tc.want != nil && *profile.Identity != *tc.want) {
				t.Fatalf("identity = %v, want %v", profile.Identity, tc.want)
			}
		})
	}
}

func TestAuthenticationIdentityPersistence(t *testing.T) {
	mgr, db := newFirstLoginHarness(t, false, false)
	profile := &model.UserAttribute{Identity: ptr.To("Master student")}
	user, err := mgr.createUser(t.Context(), "identity-test-user", nil, profile)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		incoming *string
		want     string
	}{
		{"initial creation", nil, "Master student"},
		{"update", ptr.To("PhD student"), "PhD student"},
		{"unchanged", ptr.To("PhD student"), "PhD student"},
		{"unconfigured preserves value", nil, "PhD student"},
		{"configured empty clears value", ptr.To(""), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := mgr.updateUserIfNeeded(t.Context(), user, &model.UserAttribute{Identity: tc.incoming}); err != nil {
				t.Fatal(err)
			}
			var stored model.User
			if err := db.First(&stored, user.ID).Error; err != nil {
				t.Fatal(err)
			}
			identity := stored.Attributes.Data().Identity
			if identity == nil || *identity != tc.want {
				t.Fatalf("persisted identity = %v, want %q", identity, tc.want)
			}
			if stored.Role != model.RoleUser {
				t.Fatal("profile identity changed the permission role")
			}
		})
	}
}
