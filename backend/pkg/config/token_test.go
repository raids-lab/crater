// Copyright 2026 The Crater Project Team, RAIDS-Lab
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"strings"
	"testing"
)

func TestTokenSecretConfiguration(t *testing.T) {
	for _, field := range []string{"accessTokenSecret", "refreshTokenSecret"} {
		for _, secret := range []string{"", "   ", "example-access-token", "example-refresh-token", "<MUSTEDIT>", " example-refresh-token "} {
			t.Run(field+"/"+secret, func(t *testing.T) {
				var cfg Config
				cfg.Auth.Token.AccessTokenSecret = "private-access-for-test"
				cfg.Auth.Token.RefreshTokenSecret = "private-refresh-for-test"
				if field == "accessTokenSecret" {
					cfg.Auth.Token.AccessTokenSecret = secret
				} else {
					cfg.Auth.Token.RefreshTokenSecret = secret
				}
				err := cfg.ValidateConfig()
				if err == nil || !strings.Contains(err.Error(), "auth.token."+field) {
					t.Fatalf("unsafe secret was not rejected: %v", err)
				}
			})
		}
	}
	var cfg Config
	cfg.Auth.Token.AccessTokenSecret = "private-access-for-test"
	cfg.Auth.Token.RefreshTokenSecret = "private-refresh-for-test"
	if err := cfg.ValidateConfig(); err != nil && strings.Contains(err.Error(), "auth.token.") {
		t.Fatalf("private secrets rejected: %v", err)
	}
}
