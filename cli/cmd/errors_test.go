package cmd

import (
	"errors"
	"syscall"
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

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
