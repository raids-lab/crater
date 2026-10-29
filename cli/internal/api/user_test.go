package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imroc/req/v3"
)

func TestListAdminUsersUsesTypedEndpoints(t *testing.T) {
	for _, test := range []struct {
		name string
		base bool
		path string
	}{
		{name: "full", path: AdminUsersPrefix},
		{name: "base", base: true, path: AdminUsersPrefix + "/baseinfo"},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient("https://example.invalid")
			client.httpClient.GetTransport().WrapRoundTripFunc(
				func(_ http.RoundTripper) req.HttpRoundTripFunc {
					return func(request *http.Request) (*http.Response, error) {
						if request.URL.Path != test.path {
							t.Fatalf("path = %q, want %q", request.URL.Path, test.path)
						}
						recorder := httptest.NewRecorder()
						recorder.Header().Set("Content-Type", "application/json")
						users := []AdminUser{{ID: 7, Name: "alice", Attributes: &UserAttribute{Nickname: "Alice"}}}
						var data interface{} = users
						if !test.base {
							data = Page[AdminUser]{Items: users, Total: 1, Page: 1, PageSize: 200}
						}
						_ = json.NewEncoder(recorder).Encode(Response[interface{}]{Data: data})
						return recorder.Result(), nil
					}
				},
			)

			users, err := client.ListAdminUsers(test.base)
			if err != nil {
				t.Fatalf("ListAdminUsers: %v", err)
			}
			if len(users) != 1 || users[0].Name != "alice" ||
				users[0].Attributes == nil || users[0].Attributes.Nickname != "Alice" {
				t.Fatalf("unexpected users: %#v", users)
			}
		})
	}
}

func TestAdminUserJSONOmitsFieldsAbsentFromBaseEndpoint(t *testing.T) {
	raw, err := json.Marshal(AdminUser{
		Name: "alice", Nickname: "Alice", Space: "users/alice",
	})
	if err != nil {
		t.Fatalf("marshal base user: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode base user: %v", err)
	}
	for _, field := range []string{"id", "role", "status", "extraBalance", "attributes"} {
		if _, ok := payload[field]; ok {
			t.Fatalf("base user unexpectedly contains %q: %s", field, raw)
		}
	}

	zero := 0.0
	raw, err = json.Marshal(AdminUser{
		ID: 7, Name: "alice", Role: 2, Status: 2, ExtraBalance: &zero,
		Attributes: &UserAttribute{Nickname: "Alice"},
	})
	if err != nil {
		t.Fatalf("marshal full user: %v", err)
	}
	payload = nil
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode full user: %v", err)
	}
	if value, ok := payload["extraBalance"]; !ok || value != float64(0) {
		t.Fatalf("full user did not preserve zero extra balance: %s", raw)
	}
}

func TestAdminUserAttributesPreserveBackendFields(t *testing.T) {
	raw := []byte(`{
		"id":7,
		"name":"alice",
		"role":2,
		"status":2,
		"extraBalance":0,
		"attributes":{
			"id":7,
			"name":"alice",
			"nickname":"Alice",
			"expiredAt":"2030-01-01",
			"avatar":"avatar.png",
			"uid":"1001",
			"gid":"1002"
		}
	}`)
	var user AdminUser
	if err := json.Unmarshal(raw, &user); err != nil {
		t.Fatalf("unmarshal admin user: %v", err)
	}
	encoded, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal admin user: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("decode round trip: %v", err)
	}
	attributes, ok := payload["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("attributes missing after round trip: %s", encoded)
	}
	for _, field := range []string{"expiredAt", "avatar", "uid", "gid"} {
		if _, ok := attributes[field]; !ok {
			t.Fatalf("attribute %q missing after round trip: %s", field, encoded)
		}
	}
	for _, field := range []string{"email", "phone", "teacher", "group"} {
		if _, ok := attributes[field]; ok {
			t.Fatalf("absent attribute %q was injected after round trip: %s", field, encoded)
		}
	}
}
