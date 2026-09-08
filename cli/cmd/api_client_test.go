package cmd

import (
	"errors"
	"testing"

	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/state"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

func TestLoadAccessTokenMissingTokenMapsUsageNotFound(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX", "")
	t.Setenv("CRATER_TEST_SANDBOX_SESSION", "")

	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	active := state.ActiveContext{
		PlatformURL: "https://example.invalid",
		Username:    "alice",
		Method:      "ldap",
	}
	st := state.State{
		AuthInfos: []state.AuthInfo{{
			PlatformURL: active.PlatformURL,
			Username:    active.Username,
			Method:      active.Method,
		}},
		ActiveContext: active,
	}
	_, err := loadAccessToken(st, active)
	var ce *clierror.Error
	if !errors.As(err, &ce) {
		t.Fatalf("want *clierror.Error, got %T (%v)", err, err)
	}
	if ce.Category != errorcodes.CategoryUsage {
		t.Fatalf("category = %q, want %q", ce.Category, errorcodes.CategoryUsage)
	}
	if ce.Code != errorcodes.ErrNotFound {
		t.Fatalf("code = %q, want %q", ce.Code, errorcodes.ErrNotFound)
	}
	want := i18n.T("err_token_missing")
	if ce.Message != want {
		t.Fatalf("message = %q, want %q", ce.Message, want)
	}
}

func TestLoadAccessTokenMissingAuthInfoMapsUsageNotFound(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX", "")
	t.Setenv("CRATER_TEST_SANDBOX_SESSION", "")

	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	active := state.ActiveContext{
		PlatformURL: "https://example.invalid",
		Username:    "alice",
		Method:      "ldap",
	}
	_, err := loadAccessToken(state.State{ActiveContext: active}, active)
	var ce *clierror.Error
	if !errors.As(err, &ce) {
		t.Fatalf("want *clierror.Error, got %T (%v)", err, err)
	}
	if ce.Category != errorcodes.CategoryUsage {
		t.Fatalf("category = %q, want %q", ce.Category, errorcodes.CategoryUsage)
	}
	if ce.Code != errorcodes.ErrNotFound {
		t.Fatalf("code = %q, want %q", ce.Code, errorcodes.ErrNotFound)
	}
	want := i18n.T("err_not_found")
	if ce.Message != want {
		t.Fatalf("message = %q, want %q", ce.Message, want)
	}
}
