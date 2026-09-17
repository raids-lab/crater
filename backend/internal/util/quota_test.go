package util

import (
	"path/filepath"
	"testing"

	"github.com/raids-lab/crater/pkg/config"
)

func TestCheckStorageQuotaDisabledSkipsDatabase(t *testing.T) {
	configPath, err := filepath.Abs("../../etc/debug-config.yaml")
	if err != nil {
		t.Fatalf("resolve debug config path: %v", err)
	}
	t.Setenv("CRATER_DEBUG_CONFIG_PATH", configPath)

	cfg := config.GetConfig()
	originalEnabled := cfg.Storage.Quota.Enabled
	disabled := false
	cfg.Storage.Quota.Enabled = &disabled
	t.Cleanup(func() {
		cfg.Storage.Quota.Enabled = originalEnabled
	})

	if err := CheckStorageQuota("database-is-not-initialized", nil, nil); err != nil {
		t.Fatalf("CheckStorageQuota() with quota disabled returned error: %v", err)
	}
}
