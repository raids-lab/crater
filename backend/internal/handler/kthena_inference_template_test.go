package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

func TestKthenaInferenceTemplateHandlersArePrivateToCurrentUserAndAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:kthena_inference_template_handlers?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.PrequeueConfig{}, &model.KthenaInferenceTemplate{}); err != nil {
		t.Fatal(err)
	}
	configService := service.NewConfigService(query.Use(db))
	if err := configService.SetKthenaInferenceEnabled(t.Context(), true); err != nil {
		t.Fatal(err)
	}

	currentToken := util.JWTMessage{UserID: 11, AccountID: 22, Username: "template-owner"}
	manager := &KthenaInferenceTemplateMgr{db: db, configService: configService}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		util.SetJWTContext(c, currentToken)
		c.Next()
	})
	manager.RegisterProtected(router.Group("/v1/kthena"))

	created := requestKthenaInferenceTemplate(t, router, http.MethodPost, "/v1/kthena/inference-templates", `{
		"name":"my V100 template",
		"description":"private preset",
		"config":{"backendType":"vLLM","resource":{"gpu":{"count":1,"model":"nvidia.com/gpu"}}}
	}`)
	if created.ID == 0 || created.Name != "my V100 template" || kthenaInferenceTemplateConfig(t, &created).BackendType != "vLLM" {
		t.Fatalf("created template = %+v", created)
	}

	listed := requestKthenaInferenceTemplateList(t, router)
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("owner list = %+v", listed)
	}

	// A different identity in the same browser/server process cannot read or modify it.
	currentToken = util.JWTMessage{UserID: 12, AccountID: 22, Username: "other-user"}
	if listed = requestKthenaInferenceTemplateList(t, router); len(listed) != 0 {
		t.Fatalf("foreign user received templates: %+v", listed)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodDelete,
		"/v1/kthena/inference-templates/"+formatKthenaInferenceTemplateID(created.ID), http.NoBody)
	// The endpoint must return not found for another user instead of disclosing ownership.
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("foreign delete returned %d: %s", response.Code, response.Body.String())
	}

	currentToken = util.JWTMessage{UserID: 11, AccountID: 22, Username: "template-owner"}
	updated := requestKthenaInferenceTemplate(t, router, http.MethodPut,
		"/v1/kthena/inference-templates/"+formatKthenaInferenceTemplateID(created.ID), `{
			"name":"my V100 template",
			"description":"updated private preset",
			"config":{"backendType":"vLLM","replicas":2}
		}`)
	if updated.Description != "updated private preset" || kthenaInferenceTemplateConfig(t, &updated).Replicas != 2 {
		t.Fatalf("updated template = %+v", updated)
	}
}

func TestValidateKthenaInferenceTemplateReq(t *testing.T) {
	valid := &KthenaInferenceTemplateReq{
		Name:   "  Private vLLM  ",
		Config: json.RawMessage(`{"backendType":"vLLM"}`),
	}
	if err := validateKthenaInferenceTemplateReq(valid); err != nil {
		t.Fatalf("valid request: %v", err)
	}
	if valid.Name != "Private vLLM" {
		t.Fatalf("trimmed name = %q", valid.Name)
	}
	for _, config := range []json.RawMessage{
		json.RawMessage(`[]`),
		json.RawMessage(`{"backendType":"SGLang"}`),
		json.RawMessage(`{"backendType":123}`),
	} {
		if err := validateKthenaInferenceTemplateReq(&KthenaInferenceTemplateReq{Name: "invalid", Config: config}); err == nil {
			t.Fatalf("config %s unexpectedly passed validation", config)
		}
	}
}

func requestKthenaInferenceTemplate(
	t *testing.T, router http.Handler, method, path, body string,
) KthenaInferenceTemplateResp {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s returned %d: %s", method, path, response.Code, response.Body.String())
	}
	var payload struct {
		Data KthenaInferenceTemplateResp `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Data
}

func requestKthenaInferenceTemplateList(t *testing.T, router http.Handler) []KthenaInferenceTemplateResp {
	t.Helper()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/v1/kthena/inference-templates", http.NoBody,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("GET templates returned %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data []KthenaInferenceTemplateResp `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Data
}

func formatKthenaInferenceTemplateID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

func kthenaInferenceTemplateConfig(t *testing.T, template *KthenaInferenceTemplateResp) struct {
	BackendType string `json:"backendType"`
	Replicas    int    `json:"replicas"`
} {
	t.Helper()
	var config struct {
		BackendType string `json:"backendType"`
		Replicas    int    `json:"replicas"`
	}
	if err := json.Unmarshal(template.Config, &config); err != nil {
		t.Fatal(err)
	}
	return config
}
