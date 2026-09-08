package cmd

import (
	"errors"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

func loadAccessToken(st session.State, ac session.ActiveContext) (string, error) {
	token, err := session.LoadToken(st, ac)
	if err != nil {
		if errors.Is(err, session.ErrNoToken) {
			return "", &clierror.Error{
				Category: errorcodes.CategoryUsage,
				Code:     errorcodes.ErrNotFound,
				Message:  i18n.T("err_token_missing"),
			}
		}
		if errors.Is(err, session.ErrAuthInfoNotFound) {
			return "", &clierror.Error{
				Category: errorcodes.CategoryUsage,
				Code:     errorcodes.ErrNotFound,
				Message:  i18n.T("err_not_found"),
			}
		}
		return "", err
	}
	return token, nil
}

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
	token, err := loadAccessToken(st, active)
	if err != nil {
		return nil, err
	}
	return api.NewClient(active.PlatformURL).SetToken(token), nil
}
