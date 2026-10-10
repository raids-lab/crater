package service

import (
	"context"
	"testing"

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
	if err := service.SavePersonalConfig(ctx, 7, AgentIdentityUser, personal, false); err != nil {
		t.Fatalf("save personal config: %v", err)
	}

	effective, err := service.ResolveEffectiveConfig(ctx, 7, AgentIdentityUser)
	if err != nil {
		t.Fatalf("resolve personal config: %v", err)
	}
	if effective.Source != LLMConfigSourcePersonal || effective.ModelName != personal.ModelName {
		t.Fatalf("effective config = %+v, want personal config", effective)
	}
	adminEffective, err := service.ResolveEffectiveConfig(ctx, 7, AgentIdentityAdmin)
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
	if err := service.SavePersonalConfig(ctx, 7, AgentIdentityUser, personal, false); err != nil {
		t.Fatalf("save personal config: %v", err)
	}

	if err := service.DeletePersonalConfig(ctx, 7, AgentIdentityUser); err != nil {
		t.Fatalf("delete personal config: %v", err)
	}
	effective, err := service.ResolveEffectiveConfig(ctx, 7, AgentIdentityUser)
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
	status, err := service.GetStatus(ctx, 7, AgentIdentityUser)
	if err != nil {
		t.Fatalf("load platform status: %v", err)
	}
	if status.BaseURL != "https://api.platform.example/v1" || status.ModelName != "platform-model" {
		t.Fatalf("platform defaults = %q/%q, want configured endpoint and model", status.BaseURL, status.ModelName)
	}
}

func TestAgentConfigServiceRejectsUnsafePersonalURL(t *testing.T) {
	db := openAgentConfigTestDB(t)
	service := NewAgentConfigService(db, NewConfigService(query.Use(db)))
	err := service.SavePersonalConfig(context.Background(), 9, AgentIdentityUser, &LLMConfig{
		BaseURL:   "http://127.0.0.1:8080/v1",
		ModelName: "local-model",
		APIKey:    "secret",
	}, false)
	if err == nil {
		t.Fatal("expected private HTTP endpoint to be rejected")
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
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.PrequeueConfig{}, &model.UserLLMConfig{}); err != nil {
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
		if err := db.Model(&model.SystemConfig{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			t.Fatalf("seed platform config %s: %v", key, err)
		}
	}
}
