// Copyright 2026 The Crater Project Team, RAIDS-Lab

package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSSHSessionIsExplicitAndOlderResponsesRemainUsable(t *testing.T) {
	for _, enhanced := range []bool{false, true} {
		t.Run(map[bool]string{false: "address only", true: "user session"}[enhanced], func(t *testing.T) {
			t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				want := ""
				if enhanced {
					want = "user"
				}
				if r.Method != http.MethodPost || r.URL.Path != VCJobsPrefix+"/job/ssh" || r.URL.Query().Get("session") != want {
					t.Errorf("request=%s %s, want SSH with session=%q", r.Method, r.URL, want)
				}
				w.Header().Set("Content-Type", "application/json")
				// An older backend ignores the new query and returns only its address.
				_, _ = io.WriteString(w, `{"code":0,"data":{"ip":"cpu.example.test","port":"30222"}}`)
			}))
			t.Cleanup(server.Close)
			info, err := NewClient(server.URL).OpenJobSSH("job", enhanced)
			if err != nil || info.IP != "cpu.example.test" || info.Port != "30222" || info.SessionCommand != "" {
				t.Fatalf("info=%+v error=%v, want usable legacy SSH address", info, err)
			}
		})
	}
}
