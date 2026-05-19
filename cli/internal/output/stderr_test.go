package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

func TestWriteErrorJSONReplacesUnencodableContext(t *testing.T) {
	err := &clierror.Error{
		Category: errorcodes.CategoryAPI,
		Code:     errorcodes.ErrAPIOther,
		Message:  "request failed",
		Context: map[string]interface{}{
			"bad": make(chan int),
		},
	}

	var buf bytes.Buffer
	WriteError(&buf, true, err)

	var got errorResponse
	if unmarshalErr := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &got); unmarshalErr != nil {
		t.Fatalf("expected valid JSON error output, got %q: %v", buf.String(), unmarshalErr)
	}
	if got.Category != errorcodes.CategoryAPI {
		t.Fatalf("category = %q, want %q", got.Category, errorcodes.CategoryAPI)
	}
	if got.Code != errorcodes.ErrAPIOther {
		t.Fatalf("code = %q, want %q", got.Code, errorcodes.ErrAPIOther)
	}
	if got.Message != "request failed" {
		t.Fatalf("message = %q, want request failed", got.Message)
	}
	msg, ok := got.Context["msg"].(string)
	if !ok || !strings.Contains(msg, "context JSON 化失败") {
		t.Fatalf("context msg = %#v, want JSON encode failure diagnostic", got.Context["msg"])
	}
	if _, ok := got.Context["encode_error"].(string); !ok {
		t.Fatalf("context encode_error = %#v, want string", got.Context["encode_error"])
	}
}
