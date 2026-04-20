package handler

import (
	"testing"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
)

func TestShouldIssueInitialAccountBalance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		featureEnabled bool
		active         bool
		adminCount     int
		want           bool
	}{
		{
			name:           "issues only when feature is active and admins exist",
			featureEnabled: true,
			active:         true,
			adminCount:     1,
			want:           true,
		},
		{
			name:           "skips before activation",
			featureEnabled: true,
			active:         false,
			adminCount:     1,
			want:           false,
		},
		{
			name:           "skips without admins",
			featureEnabled: true,
			active:         true,
			adminCount:     0,
			want:           false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldIssueInitialAccountBalance(tc.featureEnabled, tc.active, tc.adminCount)
			if got != tc.want {
				t.Fatalf("shouldIssueInitialAccountBalance() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCanViewUserExtraBalance(t *testing.T) {
	t.Parallel()

	target := &model.User{Model: gorm.Model{ID: 42}}
	tests := []struct {
		name  string
		token util.JWTMessage
		user  *model.User
		want  bool
	}{
		{
			name:  "allows self",
			token: util.JWTMessage{UserID: 42, RolePlatform: model.RoleUser},
			user:  target,
			want:  true,
		},
		{
			name:  "allows platform admin",
			token: util.JWTMessage{UserID: 7, RolePlatform: model.RoleAdmin},
			user:  target,
			want:  true,
		},
		{
			name:  "blocks other regular users",
			token: util.JWTMessage{UserID: 7, RolePlatform: model.RoleUser},
			user:  target,
			want:  false,
		},
		{
			name:  "blocks nil target user",
			token: util.JWTMessage{UserID: 42, RolePlatform: model.RoleAdmin},
			user:  nil,
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := canViewUserExtraBalance(tc.token, tc.user)
			if got != tc.want {
				t.Fatalf("canViewUserExtraBalance() = %v, want %v", got, tc.want)
			}
		})
	}
}
