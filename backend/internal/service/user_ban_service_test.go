package service

import (
	"testing"
	"time"

	"github.com/raids-lab/crater/dao/model"
)

func TestUserBanRestrictions(t *testing.T) {
	t.Parallel()

	restrictions := model.UserBanRestrictions{
		PlatformAccess:  true,
		JobSubmission:   false,
		ImageBuild:      true,
		ModelDownload:   false,
		DatasetDownload: true,
	}
	if !restrictions.Any() {
		t.Fatal("non-empty restrictions must be detected")
	}
	if (model.UserBanRestrictions{}).Any() {
		t.Fatal("empty restrictions must not be detected")
	}

	tests := []struct {
		name       string
		capability UserBanCapability
		want       bool
	}{
		{name: "platform access", capability: UserBanCapabilityPlatformAccess, want: true},
		{name: "job submission", capability: UserBanCapabilityJobSubmission, want: false},
		{name: "image build", capability: UserBanCapabilityImageBuild, want: true},
		{name: "model download", capability: UserBanCapabilityModelDownload, want: false},
		{name: "dataset download", capability: UserBanCapabilityDatasetDownload, want: true},
		{name: "unknown capability", capability: UserBanCapability("unknown"), want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isUserBanCapabilityRestricted(restrictions, tc.capability); got != tc.want {
				t.Fatalf("isUserBanCapabilityRestricted(%q) = %v, want %v", tc.capability, got, tc.want)
			}
		})
	}
}

func TestUserBanTimestampRules(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(2 * time.Hour)
	restrictions := model.UserBanRestrictions{PlatformAccess: true}

	if isUserBannedAt(nil, now) {
		t.Fatal("nil timestamp must not be banned")
	}
	if isUserBannedAt(&past, now) {
		t.Fatal("expired timestamp must not be banned")
	}
	if !isUserBannedAt(&future, now) {
		t.Fatal("future timestamp must be banned")
	}
	if isUserBanCapabilityBlockedAt(&past, restrictions, UserBanCapabilityPlatformAccess, now) {
		t.Fatal("expired ban restrictions must not block capabilities")
	}
	if !isUserBanCapabilityBlockedAt(&future, restrictions, UserBanCapabilityPlatformAccess, now) {
		t.Fatal("active ban restriction must block its capability")
	}
	if isUserBanCapabilityBlockedAt(&future, restrictions, UserBanCapabilityJobSubmission, now) {
		t.Fatal("unselected restriction must not block its capability")
	}

	initial, action := nextUserBanTimestamp(nil, now, false, 24*time.Hour)
	if action != model.UserBanActionBan || !initial.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("initial ban = (%v, %q)", initial, action)
	}

	extended, action := nextUserBanTimestamp(&future, now, false, 3*time.Hour)
	if action != model.UserBanActionExtend || !extended.Equal(future.Add(3*time.Hour)) {
		t.Fatalf("extended ban = (%v, %q)", extended, action)
	}

	restarted, action := nextUserBanTimestamp(&past, now, false, 30*time.Minute)
	if action != model.UserBanActionBan || !restarted.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("expired ban restart = (%v, %q)", restarted, action)
	}
}

func TestEffectiveUserBanRestrictions(t *testing.T) {
	t.Parallel()

	restrictions := model.UserBanRestrictions{PlatformAccess: true}

	if EffectiveUserBanRestrictions(false, restrictions).Any() {
		t.Fatal("expired restrictions must not be exposed as active")
	}
	if got := EffectiveUserBanRestrictions(true, restrictions); got != restrictions {
		t.Fatalf("active restrictions = %+v, want %+v", got, restrictions)
	}
}
