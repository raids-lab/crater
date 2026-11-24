package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
)

func TestUserFieldSemanticFallback(t *testing.T) {
	field := map[string]interface{}{"labels": map[string]interface{}{"zh": "毕业年份", "en-US": "Graduation year", "ja": " "}}
	if got := userFieldLabel(field, "ja", "group"); got != "Graduation year" {
		t.Fatalf("semantic fallback = %s", got)
	}
	if got := userFieldLabel(field, "zh-CN", "group"); got != "毕业年份" {
		t.Fatal(got)
	}
}

func TestUserJSONPreservesNewServerFields(t *testing.T) {
	var user api.AdminUser
	const payload = `{"id":4,"name":"alice","lastLoginAt":"2026-11-24T00:00:00Z","newField":{"x":1},"attributes":{"identity":"intern"}}`
	if err := json.Unmarshal([]byte(payload), &user); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(user)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"newField":{"x":1}`) || user.Field("identity") != "intern" {
		t.Fatalf("lost response fields: %s", raw)
	}
}
