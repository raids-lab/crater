package cmd

import (
	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

func activeAPIClient() (*api.Client, error) {
	st, err := session.LoadState()
	if err != nil {
		return nil, &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  i18n.T("err_config_write", err.Error()),
		}
	}
	active := st.ActiveContext
	if active.PlatformURL == "" {
		return nil, &clierror.Error{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrNotFound,
			Message:  i18n.T("err_no_active"),
		}
	}
	token, err := session.LoadToken(active)
	if err != nil {
		return nil, &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrSecureStorageError,
			Message:  i18n.T("err_token_load_failed", err.Error()),
		}
	}
	return api.NewClient(active.PlatformURL).SetToken(token), nil
}
