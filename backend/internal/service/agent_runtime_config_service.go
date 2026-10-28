package service

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

const (
	agentRuntimeConfigVersion = 1
	maxAgentRuntimeFieldBytes = 100_000
	maxAgentRuntimeRegexBytes = 8_192
	maxAgentRuntimeJSONBytes  = 1 << 20
)

type AgentLocaleRuntimeConfig struct {
	BasePrompt                    string `json:"basePrompt"`
	PlatformGuidePrompt           string `json:"platformGuidePrompt"`
	PlatformKnowledge             string `json:"platformKnowledge"`
	FormAssistantPrompt           string `json:"formAssistantPrompt"`
	JobDiagnosisPrompt            string `json:"jobDiagnosisPrompt"`
	JobDiagnosisUnavailablePrompt string `json:"jobDiagnosisUnavailablePrompt"`
	TitlePrompt                   string `json:"titlePrompt"`
	ScopeRefusalMessage           string `json:"scopeRefusalMessage"`
	PromptOverrideRegex           string `json:"promptOverrideRegex"`
}

type AgentRuntimeLocales struct {
	ZhCN AgentLocaleRuntimeConfig `json:"zhCN"`
	EnUS AgentLocaleRuntimeConfig `json:"enUS"`
}

type AgentRuntimeConfig struct {
	Version int                 `json:"version"`
	Locales AgentRuntimeLocales `json:"locales"`
}

type AgentRuntimeConfigService struct {
	db *gorm.DB
}

func NewAgentRuntimeConfigService(db *gorm.DB) *AgentRuntimeConfigService {
	return &AgentRuntimeConfigService{db: db}
}

func (s *AgentRuntimeConfigService) Get(ctx context.Context) (*AgentRuntimeConfig, bool, error) {
	var stored model.SystemConfig
	err := s.db.WithContext(ctx).Where("key = ?", model.ConfigKeyAgentRuntime).First(&stored).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, bizerr.Internal.DatabaseError.Wrap(err, "failed to load Agent runtime configuration")
	}
	var config AgentRuntimeConfig
	if err := json.Unmarshal([]byte(stored.Value), &config); err != nil {
		return nil, false, bizerr.Internal.DatabaseError.Wrap(err, "invalid stored Agent runtime configuration")
	}
	if err := ValidateAgentRuntimeConfig(&config); err != nil {
		return nil, false, err
	}
	return &config, true, nil
}

func (s *AgentRuntimeConfigService) Save(ctx context.Context, config *AgentRuntimeConfig) error {
	if err := ValidateAgentRuntimeConfig(config); err != nil {
		return err
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, "invalid Agent runtime configuration")
	}
	if len(raw) > maxAgentRuntimeJSONBytes {
		return bizerr.BadRequest.ParameterError.New("Agent runtime configuration is too large")
	}
	stored := model.SystemConfig{Key: model.ConfigKeyAgentRuntime, Value: string(raw)}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&stored).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to save Agent runtime configuration")
	}
	return nil
}

func ValidateAgentRuntimeConfig(config *AgentRuntimeConfig) error {
	if config == nil || config.Version != agentRuntimeConfigVersion {
		return bizerr.BadRequest.ParameterError.New("unsupported Agent runtime configuration version")
	}
	if err := validateAgentLocaleRuntimeConfig("zhCN", &config.Locales.ZhCN); err != nil {
		return err
	}
	return validateAgentLocaleRuntimeConfig("enUS", &config.Locales.EnUS)
}

func validateAgentLocaleRuntimeConfig(locale string, config *AgentLocaleRuntimeConfig) error {
	fields := map[string]string{
		"basePrompt":                    config.BasePrompt,
		"platformGuidePrompt":           config.PlatformGuidePrompt,
		"platformKnowledge":             config.PlatformKnowledge,
		"formAssistantPrompt":           config.FormAssistantPrompt,
		"jobDiagnosisPrompt":            config.JobDiagnosisPrompt,
		"jobDiagnosisUnavailablePrompt": config.JobDiagnosisUnavailablePrompt,
		"titlePrompt":                   config.TitlePrompt,
		"scopeRefusalMessage":           config.ScopeRefusalMessage,
		"promptOverrideRegex":           config.PromptOverrideRegex,
	}
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			return bizerr.BadRequest.ParameterError.New(locale + "." + name + " cannot be empty")
		}
		if len(value) > maxAgentRuntimeFieldBytes {
			return bizerr.BadRequest.ParameterError.New(locale + "." + name + " is too large")
		}
	}
	if len(config.PromptOverrideRegex) > maxAgentRuntimeRegexBytes {
		return bizerr.BadRequest.ParameterError.New(locale + ".promptOverrideRegex is too large")
	}
	if _, err := regexp.Compile(config.PromptOverrideRegex); err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, locale+".promptOverrideRegex is invalid")
	}
	return nil
}
