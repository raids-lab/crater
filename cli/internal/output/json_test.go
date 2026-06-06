package output

import (
	"strings"
	"testing"
)

func TestMarshalJSONPrettyIndented(t *testing.T) {
	t.Parallel()
	b, err := MarshalJSONPretty(map[string]string{"status": "OK"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "{\n") {
		t.Fatalf("want indented object, got %q", s)
	}
	if !strings.HasSuffix(s, "\n") {
		t.Fatalf("want trailing newline, got %q", s)
	}
}
