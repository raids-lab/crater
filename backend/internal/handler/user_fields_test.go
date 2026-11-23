package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/config"
	"k8s.io/utils/ptr"
)

func TestUserFieldsEndpoint(t *testing.T) {
	cfg := config.GetConfig()
	previous := *cfg
	t.Cleanup(func() { *cfg = previous })
	cfg.UserFields = map[string]config.UserFieldConfig{
		"identity": {ShowInTable: ptr.To(true), Labels: map[string]string{"en": "Position"}},
	}
	cfg.Auth.LDAP.Server.BindPassword = "must-not-appear-in-presentation"
	recorder := httptest.NewRecorder()
	router := gin.New()
	(&SystemConfigMgr{}).RegisterAdmin(router.Group("/v1/admin/system-config"))
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/v1/admin/system-config/user-fields", nil))
	var response resputil.Response[map[string]config.UserFieldPresentation]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != 200 || response.Code != 0 || !response.Data["identity"].ShowInTable || response.Data["identity"].Labels["en"] != "Position" {
		t.Fatalf("unexpected field configuration: %s", recorder.Body.String())
	}
	if len(response.Data) != len(config.UserFieldDefaults()) {
		t.Fatal("endpoint must return only supported presentation fields")
	}
}
