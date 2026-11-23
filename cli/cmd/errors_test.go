package cmd

import (
	"errors"
	"net/http"
	"strings"
	"syscall"
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

func TestRefreshErrorsExplainRenewalAndPreserveDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		status, code int
		wantSignIn   bool
	}{
		{http.StatusNotFound, 40401, false},
		{http.StatusUnauthorized, 40102, true},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			const upstream = "original platform response"
			got := cliErrFromAPI(&api.RequestError{
				HTTPStatus: tc.status, CraterCode: tc.code, Msg: upstream, DuringRefresh: true,
			})
			if got.Code != apiCodeForHTTP(tc.status) || got.Context["http_status"] != tc.status ||
				got.Context["crater_code"] != tc.code || got.Context["msg"] != upstream {
				t.Fatalf("unexpected error diagnostics: %+v", got)
			}
			want := i18n.T("err_session_refresh", i18n.T("err_api_request", tc.status, upstream))
			if !strings.HasPrefix(got.Message, want) ||
				strings.Contains(got.Message, i18n.T("err_session_expired")) != tc.wantSignIn {
				t.Fatalf("unexpected refresh message: %q", got.Message)
			}
		})
	}
}

func TestErrSurveyOrSameInterrupt(t *testing.T) {
	t.Parallel()
	got := errSurveyOrSame(terminal.InterruptErr)
	var ce *clierror.Error
	if !errors.As(got, &ce) {
		t.Fatalf("want *clierror.Error, got %T", got)
	}
	if ce.Category != errorcodes.CategoryCancelled {
		t.Fatalf("category: got %q want %q", ce.Category, errorcodes.CategoryCancelled)
	}
	if ce.Code != errorcodes.ErrOperationCancelled {
		t.Fatalf("code: got %q want %q", ce.Code, errorcodes.ErrOperationCancelled)
	}
}

func TestErrSurveyOrSameEINTR(t *testing.T) {
	t.Parallel()
	got := errSurveyOrSame(syscall.EINTR)
	var ce *clierror.Error
	if !errors.As(got, &ce) {
		t.Fatalf("want *clierror.Error, got %T", got)
	}
	if ce.Category != errorcodes.CategoryCancelled {
		t.Fatalf("category: got %q want %q", ce.Category, errorcodes.CategoryCancelled)
	}
}

func TestErrUsageFromIssuesMultiple(t *testing.T) {
	t.Parallel()
	e := errUsageFromIssues([]usageIssue{
		{Code: errorcodes.ErrMissingRequiredFlag, Message: "a", Field: "username"},
		{Code: errorcodes.ErrMissingRequiredFlag, Message: "b", Field: "password"},
	})
	if e.Context == nil {
		t.Fatal("expected context for multiple issues")
	}
	issues, ok := e.Context["issues"].([]map[string]interface{})
	if !ok || len(issues) != 2 {
		t.Fatalf("issues: %#v", e.Context["issues"])
	}
}

func TestErrSurveyOrSamePassesThrough(t *testing.T) {
	t.Parallel()
	orig := errors.New("other")
	if got := errSurveyOrSame(orig); !errors.Is(got, orig) {
		t.Fatalf("want original error preserved")
	}
}
