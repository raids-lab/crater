package alert

import (
	"context"
	"errors"
	"testing"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/config"
)

func TestUserFieldsDisabledEmailStopsExistingJobNotification(t *testing.T) {
	cfg := config.GetConfig()
	old := cfg.UserFields
	t.Cleanup(func() { cfg.UserFields = old })
	cfg.UserFields = map[string]config.UserFieldConfig{}
	unavailable := errors.New("sender must not be called")
	mgr := &alertMgr{err: unavailable}
	if err := mgr.sendJobNotification(context.Background(), "old-job", "subject", model.AlertType(0), nil, nil); err != nil {
		t.Fatal(err)
	}
	cfg.UserFields = map[string]config.UserFieldConfig{"email": {}}
	if err := mgr.sendJobNotification(context.Background(), "old-job", "subject", model.AlertType(0), nil, nil); !errors.Is(err, unavailable) {
		t.Fatal("private enabled email must still allow notifications")
	}
}
