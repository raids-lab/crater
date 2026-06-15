//nolint:dupl // Chat and completion requests intentionally share nearly identical HTTP transport flow.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/crypto"
)

type ProviderConfig struct {
	BaseURL   string
	APIKey    string
	ModelName string
}

func (c ProviderConfig) CompletionURL() string {
	baseURL := strings.TrimSuffix(strings.TrimSpace(c.BaseURL), "/")
	if baseURL == "" {
		return ""
	}
	if strings.HasSuffix(baseURL, "/completions") {
		return baseURL
	}
	return baseURL + "/completions"
}

func (c ProviderConfig) ChatCompletionURL() string {
	baseURL := strings.TrimSuffix(strings.TrimSpace(c.BaseURL), "/")
	if baseURL == "" {
		return ""
	}
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}

func loadRuntimeLLMConfig(ctx context.Context) (*ProviderConfig, error) {
	cfg := &ProviderConfig{
		BaseURL:   strings.TrimSpace(os.Getenv("LLM_API_BASE_URL")),
		APIKey:    strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		ModelName: strings.TrimSpace(os.Getenv("LLM_MODEL_NAME")),
	}

	var rows []model.SystemConfig
	err := query.GetDB().WithContext(ctx).
		Where("key IN ?", []string{
			model.ConfigKeyLLMBaseURL,
			model.ConfigKeyLLMAPIKey,
			model.ConfigKeyLLMModelName,
		}).
		Find(&rows).Error
	if err != nil {
		if cfg.BaseURL == "" || cfg.ModelName == "" {
			return nil, fmt.Errorf("failed to load llm config from database: %w", err)
		}
		return cfg, nil
	}

	configMap := make(map[string]string, len(rows))
	for _, row := range rows {
		configMap[row.Key] = strings.TrimSpace(row.Value)
	}

	if baseURL := configMap[model.ConfigKeyLLMBaseURL]; baseURL != "" {
		cfg.BaseURL = baseURL
	}
	if modelName := configMap[model.ConfigKeyLLMModelName]; modelName != "" {
		cfg.ModelName = modelName
	}
	if encryptedKey := configMap[model.ConfigKeyLLMAPIKey]; encryptedKey != "" {
		plainKey, decryptErr := crypto.Decrypt(encryptedKey)
		if decryptErr != nil {
			klog.Warningf("loadRuntimeLLMConfig: failed to decrypt llm api key, using raw value: %v", decryptErr)
			cfg.APIKey = encryptedKey
		} else {
			cfg.APIKey = plainKey
		}
	}

	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("llm base url is not configured")
	}
	if cfg.ModelName == "" {
		return nil, fmt.Errorf("llm model name is not configured")
	}

	return cfg, nil
}

type completionRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
}

type completionResponse struct {
	Choices []struct {
		Text         string `json:"text"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func callConfiguredLLM(ctx context.Context, cfg ProviderConfig, req dsRequest) (*dsResponse, error) {
	req.Model = cfg.ModelName

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	chatCompletionURL := cfg.ChatCompletionURL()
	if chatCompletionURL == "" {
		return nil, fmt.Errorf("llm chat completion url is empty")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 120 * time.Second}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer httpResp.Body.Close()

	var apiResp dsResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析 API 响应失败: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("HTTP %d", httpResp.StatusCode)
		if apiResp.Error != nil {
			errMsg = apiResp.Error.Message
		}
		return nil, fmt.Errorf("API 错误: %s", errMsg)
	}

	return &apiResp, nil
}

func callConfiguredCompletion(ctx context.Context, cfg ProviderConfig, req completionRequest) (*completionResponse, error) {
	req.Model = cfg.ModelName

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal completion request: %w", err)
	}

	completionURL := cfg.CompletionURL()
	if completionURL == "" {
		return nil, fmt.Errorf("llm completion url is empty")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, completionURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create completion request: %w", err)
	}
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 120 * time.Second}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("completion request failed: %w", err)
	}
	defer httpResp.Body.Close()

	var apiResp completionResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode completion response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("HTTP %d", httpResp.StatusCode)
		if apiResp.Error != nil {
			errMsg = apiResp.Error.Message
		}
		return nil, fmt.Errorf("completion API error: %s", errMsg)
	}

	return &apiResp, nil
}
