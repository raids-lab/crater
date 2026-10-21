package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/output"
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

func TestLegacyReadErrorsKeepCLIClassificationAndJSONFacts(t *testing.T) {
	tests := []struct {
		name       string
		httpStatus int
		craterCode int
		msg        string
		path       string
		wantCode   string
	}{
		{name: "image build lookup", httpStatus: http.StatusBadRequest, craterCode: 40001, msg: "fetch kaniko by name failed, err record not found", path: api.ImagesPrefix + "/getbyname", wantCode: errorcodes.ErrClient4XX},
		{name: "dataset association lookup", httpStatus: http.StatusInternalServerError, craterCode: 99999, msg: "Can't get mydatasets", path: api.DatasetPrefix + "/mydataset", wantCode: errorcodes.ErrServerInternal5XX},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CRATER_TEST_SANDBOX", "1")
			t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != tt.path {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.httpStatus)
				if encodeErr := json.NewEncoder(w).Encode(api.Response[any]{Code: tt.craterCode, Message: tt.msg}); encodeErr != nil {
					t.Errorf("encode loopback fixture: %v", encodeErr)
				}
			}))
			t.Cleanup(server.Close)
			client := api.NewClient(server.URL)
			var apiErr error
			if tt.name == "image build lookup" {
				_, apiErr = client.GetKanikoByName("missing")
			} else {
				_, apiErr = client.GetRaw(tt.path, nil)
			}
			if apiErr == nil {
				t.Fatal("expected API read failure")
			}
			err := cliErrFromAPI(apiErr)
			if err.Category != errorcodes.CategoryAPI || err.Code != tt.wantCode || errorcodes.ExitCodeForCategory(err.Category) != 4 {
				t.Fatalf("unexpected CLI classification: %+v", err)
			}
			var stderr bytes.Buffer
			output.WriteError(&stderr, true, err)
			var result struct {
				Category string `json:"category"`
				Code     string `json:"code"`
				Context  struct {
					HTTPStatus int    `json:"http_status"`
					CraterCode int    `json:"crater_code"`
					Msg        string `json:"msg"`
				} `json:"context"`
			}
			if decodeErr := json.Unmarshal(stderr.Bytes(), &result); decodeErr != nil {
				t.Fatalf("decode CLI JSON stderr: %v", decodeErr)
			}
			if result.Category != string(errorcodes.CategoryAPI) || result.Code != tt.wantCode ||
				result.Context.HTTPStatus != tt.httpStatus || result.Context.CraterCode != tt.craterCode || result.Context.Msg != tt.msg {
				t.Fatalf("CLI JSON lost legacy response facts: %s", stderr.String())
			}
		})
	}
}
