package service

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/crypto"
)

func TestAgentConfigServicePersonalOverrideIsSharedAcrossIdentities(t *testing.T) {
	db := openAgentConfigTestDB(t)
	platformService := NewConfigService(query.Use(db))
	seedPlatformLLMConfig(t, db)
	service := NewAgentConfigService(db, platformService)
	ctx := context.Background()

	personal := &LLMConfig{
		BaseURL:   "https://api.personal.example/v1",
		ModelName: "personal-model",
		APIKey:    "personal-key",
	}
	if err := service.SavePersonalConfig(ctx, 7, personal, false); err != nil {
		t.Fatalf("save personal config: %v", err)
	}

	effective, err := service.ResolveEffectiveConfig(ctx, 7)
	if err != nil {
		t.Fatalf("resolve personal config: %v", err)
	}
	if effective.Source != LLMConfigSourcePersonal || effective.ModelName != personal.ModelName {
		t.Fatalf("effective config = %+v, want personal config", effective)
	}
	adminEffective, err := service.ResolveEffectiveConfig(ctx, 7)
	if err != nil {
		t.Fatalf("resolve shared personal config for admin identity: %v", err)
	}
	if adminEffective.Source != LLMConfigSourcePersonal || adminEffective.ModelName != personal.ModelName {
		t.Fatalf("admin effective config = %+v, want shared personal config", adminEffective)
	}
	var stored model.UserLLMConfig
	if err := db.First(&stored, "user_id = ?", 7).Error; err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if stored.EncryptedAPIKey == personal.APIKey {
		t.Fatal("personal API key was stored as plaintext")
	}
}

func TestAgentConfigServiceDeleteRestoresPlatformFallback(t *testing.T) {
	db := openAgentConfigTestDB(t)
	platformService := NewConfigService(query.Use(db))
	seedPlatformLLMConfig(t, db)
	service := NewAgentConfigService(db, platformService)
	ctx := context.Background()
	personal := &LLMConfig{
		BaseURL:   "https://api.personal.example/v1",
		ModelName: "personal-model",
		APIKey:    "personal-key",
	}
	if err := service.SavePersonalConfig(ctx, 7, personal, false); err != nil {
		t.Fatalf("save personal config: %v", err)
	}

	if err := service.DeletePersonalConfig(ctx, 7); err != nil {
		t.Fatalf("delete personal config: %v", err)
	}
	effective, err := service.ResolveEffectiveConfig(ctx, 7)
	if err != nil {
		t.Fatalf("resolve platform fallback: %v", err)
	}
	if effective.Source != LLMConfigSourcePlatform || effective.ModelName != "platform-model" {
		t.Fatalf("effective config = %+v, want platform fallback", effective)
	}
}

func TestAgentConfigServiceReportsPlatformDefaults(t *testing.T) {
	db := openAgentConfigTestDB(t)
	platformService := NewConfigService(query.Use(db))
	seedPlatformLLMConfig(t, db)
	service := NewAgentConfigService(db, platformService)
	ctx := context.Background()
	status, err := service.GetStatus(ctx, 7)
	if err != nil {
		t.Fatalf("load platform status: %v", err)
	}
	if status.BaseURL != "https://api.platform.example/v1" || status.ModelName != "platform-model" {
		t.Fatalf("platform defaults = %q/%q, want configured endpoint and model", status.BaseURL, status.ModelName)
	}
}

func TestAgentConfigServiceUnconfiguredStatusDoesNotRequireEncryptionKey(t *testing.T) {
	db := openAgentConfigTestDB(t)
	t.Setenv(AgentEncryptionKeyEnv, "")
	service := NewAgentConfigService(db, NewConfigService(query.Use(db)))
	ctx := context.Background()

	status, err := service.GetStatus(ctx, 7)
	if err != nil {
		t.Fatalf("load unconfigured status: %v", err)
	}
	if status.PersonalConfigured || status.PlatformConfigured || status.EffectiveSource != "" {
		t.Fatalf("status = %+v, want no configured model", status)
	}
	if _, err := service.ResolveEffectiveConfig(ctx, 7); err == nil {
		t.Fatal("expected model use to reject missing configuration")
	}
	if err := db.Create(&model.UserLLMConfig{
		UserID:    7,
		BaseURL:   "https://api.personal.example/v1",
		ModelName: "personal-model",
	}).Error; err != nil {
		t.Fatalf("seed configuration without an API key: %v", err)
	}
	status, err = service.GetStatus(ctx, 7)
	if err != nil || status.PersonalConfigured {
		t.Fatalf("load configuration without an API key: status=%+v, err=%v", status, err)
	}
	if _, err := service.ResolveEffectiveConfig(ctx, 7); err == nil ||
		!strings.Contains(err.Error(), "LLM is not configured") {
		t.Fatalf("model use error = %v, want missing model configuration", err)
	}
}

func TestAgentConfigServiceStatusDoesNotDecryptPersonalCredentials(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  string
	}{
		{name: "missing encryption key", key: ""},
		{name: "different encryption key", key: "abcdef0123456789abcdef0123456789"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := openAgentConfigTestDB(t)
			encrypted, err := crypto.EncryptWithKey("personal-key", "0123456789abcdef0123456789abcdef")
			if err != nil {
				t.Fatalf("encrypt personal key: %v", err)
			}
			record := &model.UserLLMConfig{
				UserID:          7,
				BaseURL:         "https://api.personal.example/v1",
				ModelName:       "personal-model",
				EncryptedAPIKey: encrypted,
			}
			if err := db.Create(record).Error; err != nil {
				t.Fatalf("seed personal configuration: %v", err)
			}
			t.Setenv(AgentEncryptionKeyEnv, tc.key)
			t.Setenv(AgentPreviousEncryptionKeysEnv, "")
			service := NewAgentConfigService(db, NewConfigService(query.Use(db)))
			ctx := context.Background()
			status, err := service.GetStatus(ctx, 7)
			if err != nil {
				t.Fatalf("load saved configuration status: %v", err)
			}
			if !status.PersonalConfigured || status.EffectiveSource != LLMConfigSourcePersonal ||
				status.BaseURL != record.BaseURL || status.ModelName != record.ModelName {
				t.Fatalf("status = %+v, want saved personal configuration metadata", status)
			}
			if _, err := service.ResolveEffectiveConfig(ctx, 7); err == nil {
				t.Fatal("expected model use to reject unreadable credentials")
			}
		})
	}
}

func TestAgentConfigServiceRejectsUnsafePersonalURL(t *testing.T) {
	db := openAgentConfigTestDB(t)
	service := NewAgentConfigService(db, NewConfigService(query.Use(db)))
	err := service.SavePersonalConfig(context.Background(), 9, &LLMConfig{
		BaseURL:   "http://127.0.0.1:8080/v1",
		ModelName: "local-model",
		APIKey:    "secret",
	}, false)
	if err == nil {
		t.Fatal("expected private HTTP endpoint to be rejected")
	}
}

func TestPersonalLLMHTTPClientDisablesEnvironmentProxy(t *testing.T) {
	client := NewLLMHTTPClient(LLMConfigSourcePersonal, time.Second)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("personal LLM transport must not use environment proxies")
	}
}

func TestAgentContextRoundTrip(t *testing.T) {
	const secret = "this-is-a-test-secret-with-at-least-32-bytes"
	token, err := SignAgentContext(42, AgentIdentityAdmin, secret)
	if err != nil {
		t.Fatalf("sign context: %v", err)
	}
	claims, err := ParseAgentContext(token, secret)
	if err != nil {
		t.Fatalf("parse context: %v", err)
	}
	if claims.UserID != 42 {
		t.Fatalf("user ID = %d, want 42", claims.UserID)
	}
	if claims.IdentityScope != AgentIdentityAdmin {
		t.Fatalf("identity = %q, want %q", claims.IdentityScope, AgentIdentityAdmin)
	}
	if _, err := ParseAgentContext(token, secret+"-wrong"); err == nil {
		t.Fatal("expected signature validation failure")
	}
}

func openAgentConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv(AgentEncryptionKeyEnv, "0123456789abcdef0123456789abcdef")
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.UserLLMConfig{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

func seedPlatformLLMConfig(t *testing.T, db *gorm.DB) {
	t.Helper()
	encrypted, err := crypto.Encrypt("platform-key")
	if err != nil {
		t.Fatalf("encrypt platform key: %v", err)
	}
	values := map[string]string{
		model.ConfigKeyLLMBaseURL:   "https://api.platform.example/v1",
		model.ConfigKeyLLMAPIKey:    encrypted,
		model.ConfigKeyLLMModelName: "platform-model",
	}
	for key, value := range values {
		if err := db.Save(&model.SystemConfig{Key: key, Value: value}).Error; err != nil {
			t.Fatalf("seed platform config %s: %v", key, err)
		}
	}
}
