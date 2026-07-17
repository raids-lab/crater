package service

import "testing"

func TestUserBanPolicyAllows(t *testing.T) {
	t.Parallel()

	policy := UserBanPolicy{
		AllowPlatformAccess:  true,
		AllowJobSubmission:   false,
		AllowImageBuild:      true,
		AllowModelDownload:   false,
		AllowDatasetDownload: true,
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
		{name: "unknown capability", capability: UserBanCapability("unknown"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := policy.Allows(tc.capability); got != tc.want {
				t.Fatalf("Allows(%q) = %v, want %v", tc.capability, got, tc.want)
			}
		})
	}
}
