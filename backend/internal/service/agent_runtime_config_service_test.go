package service

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

func TestAgentRuntimeConfigServiceSaveAndGet(t *testing.T) {
	db := openAgentRuntimeConfigTestDB(t)
	svc := NewAgentRuntimeConfigService(db)

	if config, found, err := svc.Get(context.Background()); err != nil || found || config != nil {
		t.Fatalf("empty config = %#v/%t/%v, want nil/false/nil", config, found, err)
	}

	config := validAgentRuntimeConfig()
	if err := svc.Save(context.Background(), config); err != nil {
		t.Fatalf("save config: %v", err)
	}
	config.Locales.ZhCN.BasePrompt = "updated base prompt"
	if err := svc.Save(context.Background(), config); err != nil {
		t.Fatalf("update config: %v", err)
	}

	stored, found, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if !found || stored.Locales.ZhCN.BasePrompt != "updated base prompt" {
		t.Fatalf("stored config = %#v/%t", stored, found)
	}
	var count int64
	if err := db.Model(&model.SystemConfig{}).
		Where("key = ?", model.ConfigKeyAgentRuntime).
		Count(&count).Error; err != nil {
		t.Fatalf("count stored config: %v", err)
	}
	if count != 1 {
		t.Fatalf("stored config count = %d, want 1", count)
	}
}

func TestValidateAgentRuntimeConfigRejectsInvalidRegex(t *testing.T) {
	config := validAgentRuntimeConfig()
	config.Locales.ZhCN.PromptOverrideRegex = "["
	if err := ValidateAgentRuntimeConfig(config); err == nil {
		t.Fatal("expected invalid regex to fail validation")
	}

	config = validAgentRuntimeConfig()
	config.Locales.ZhCN.PromptOverrideRegex = strings.Repeat("a", maxAgentRuntimeRegexBytes+1)
	if err := ValidateAgentRuntimeConfig(config); err == nil {
		t.Fatal("expected oversized regex to fail validation")
	}
}

func openAgentRuntimeConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

func validAgentRuntimeConfig() *AgentRuntimeConfig {
	locale := AgentLocaleRuntimeConfig{
		BasePrompt:                    "base prompt",
		PlatformGuidePrompt:           "platform guide",
		PlatformKnowledge:             "platform knowledge",
		FormAssistantPrompt:           "form assistant",
		JobDiagnosisPrompt:            "job diagnosis",
		JobDiagnosisUnavailablePrompt: "diagnosis unavailable",
		TitlePrompt:                   "title prompt",
		ScopeRefusalMessage:           "scope refusal",
		PromptOverrideRegex:           "ignore|reveal",
	}
	return &AgentRuntimeConfig{
		Version: agentRuntimeConfigVersion,
		Locales: AgentRuntimeLocales{
			ZhCN: locale,
			EnUS: locale,
		},
	}
}
