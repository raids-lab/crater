package read_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/snaptest"
)

func TestDatasetSharedUsersArrayContract(t *testing.T) {
	bin := snaptest.CraterExecutable(t)
	for _, tc := range []struct {
		name   string
		count  int
		status int
	}{
		{name: "complete list", count: 201, status: http.StatusOK},
		{name: "empty list", status: http.StatusOK},
		{name: "read failure", status: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != api.DatasetPrefix+"/1/usersIn" || r.URL.RawQuery != "" {
					t.Errorf("unexpected dataset users request: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				if tc.status != http.StatusOK {
					w.WriteHeader(tc.status)
					_ = json.NewEncoder(w).Encode(api.Response[any]{Code: 50002, Message: "failed to get dataset shared users"})
					return
				}
				users := make([]map[string]any, tc.count)
				for index := range users {
					users[index] = map[string]any{"id": index + 1, "name": fmt.Sprintf("user-%d", index+1), "isowner": index == 0}
				}
				if err := json.NewEncoder(w).Encode(api.Response[[]map[string]any]{Data: users}); err != nil {
					t.Errorf("encode shared-user fixture: %v", err)
				}
			}))
			defer server.Close()
			env := append(snaptest.EnvMinimal(t.TempDir(), "en"),
				"CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL="+server.URL)
			result, err := snaptest.Run(bin, env, []string{"dataset", "users", "1", "--json", "--no-interactive"})
			if err != nil {
				t.Fatal(err)
			}
			if requests.Load() != 1 {
				t.Fatalf("dataset users made %d requests, want one complete-array read", requests.Load())
			}
			if tc.status != http.StatusOK {
				if result.ExitCode != 4 || result.Stdout != "" || !strings.Contains(result.Stderr, "failed to get dataset shared users") {
					t.Fatalf("read error was not preserved: %+v", result)
				}
				return
			}
			if result.ExitCode != 0 || result.Stderr != "" {
				t.Fatalf("shared-user read failed: %+v", result)
			}
			var response struct {
				Data struct {
					Users []struct {
						ID      int  `json:"id"`
						IsOwner bool `json:"isowner"`
					} `json:"users"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
				t.Fatal(err)
			}
			if response.Data.Users == nil || len(response.Data.Users) != tc.count {
				t.Fatalf("incomplete shared-user output: %s", result.Stdout)
			}
			for index, user := range response.Data.Users {
				if user.ID != index+1 || user.IsOwner != (index == 0) {
					t.Fatalf("shared-user output changed: %+v", user)
				}
			}
		})
	}
}
