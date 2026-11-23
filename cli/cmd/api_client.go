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
	client, _, err := activeAPIClientWithContext()
	return client, err
}

func activeAPIClientWithContext() (*api.Client, session.ActiveContext, error) {
	st, err := session.LoadState()
	if err != nil {
		return nil, session.ActiveContext{}, &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  i18n.T("err_config_write", err.Error()),
		}
	}
	active := st.ActiveContext
	if active.PlatformURL == "" {
		return nil, active, &clierror.Error{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrNotFound,
			Message:  i18n.T("err_no_active"),
		}
	}
	token, err := loadAccessToken(st, active)
	if err != nil {
		return nil, active, err
	}
	info, _ := session.AuthInfoFor(st, active)
	tokens := api.Tokens{AccessToken: token, RefreshToken: info.RefreshToken}
	check := func(current api.Tokens) error {
		return session.CheckTokens(active, current.AccessToken, current.RefreshToken)
	}
	save := func(previous, next api.Tokens, requireActive bool) error {
		return session.ReplaceTokens(active, previous.AccessToken, previous.RefreshToken, next.AccessToken, next.RefreshToken, requireActive)
	}
	return api.NewClient(active.PlatformURL).SetSession(tokens, check, save), active, nil
}
