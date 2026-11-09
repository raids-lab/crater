// Copyright 2026 The Crater Project Team, RAIDS-Lab

package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenJobSSHRequestsNativeAddress(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != VCJobsPrefix+"/job/ssh" || r.URL.RawQuery != "" {
			t.Errorf("request=%s %s, want POST to the existing SSH address endpoint", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"ip":"cpu.example.test","port":"30222"}}`)
	}))
	t.Cleanup(server.Close)
	info, err := NewClient(server.URL).OpenJobSSH("job")
	if err != nil || info.IP != "cpu.example.test" || info.Port != "30222" {
		t.Fatalf("info=%+v error=%v, want usable SSH address", info, err)
	}
}
