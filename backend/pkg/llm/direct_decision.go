package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/crypto"
)

const (
	StorageDecisionModeEnv              = "CRATER_STORAGE_DECISION_MODE"
	StorageDecisionModeAgent            = "agent"
	StorageDecisionModeDirect           = "direct"
	StorageDecisionConfigSourceEnv      = "CRATER_STORAGE_DECISION_CONFIG_SOURCE"
	StorageDecisionConfigSourcePlatform = "platform"
	StorageDecisionConfigSourceCustom   = "custom"

	DirectModelBaseURLEnv     = "CRATER_STORAGE_DIRECT_MODEL_BASE_URL"
	DirectModelAPIKeyEnv      = "CRATER_STORAGE_DIRECT_MODEL_API_KEY"
	DirectModelNameEnv        = "CRATER_STORAGE_DIRECT_MODEL_NAME"
	DefaultDirectModelBaseURL = "http://192.168.5.68:30186/v1"

	directDecisionInstruction = `你是面向 AI 集群的存储治理决策模型。请根据输入的结构化存储治理快照，输出一个 JSON 决策对象。
输出要求：
1. 只输出 JSON，不要附加解释文本。
2. JSON 字段固定为，且顺序必须如下：
   - reason: string
   - allow_expand: boolean
   - expand_bytes: integer
   - freeze_new_jobs: boolean
3. reason 需要简洁说明决策依据，优先引用 usage_ratio、growth_rate_bytes_per_hour、平台容量和 Prometheus 相关字段。
4. reason 必须与 allow_expand、expand_bytes、freeze_new_jobs 严格一致：
   - allow_expand=false 时，reason 不得写成“建议扩容”“应扩容”或任何支持扩容的表述
   - allow_expand=true 时，expand_bytes 必须为正数，reason 需要明确支持扩容
   - freeze_new_jobs=true 时，reason 必须明确说明需要冻结新作业
   - freeze_new_jobs=false 时，reason 不得写成“建议冻结新作业”或任何支持冻结的表述
5. expand_bytes 必须使用字节数。
6. 不要重复输入，不要输出 markdown，不要输出额外内容。`

	directDecisionRefinementInstruction = `你是存储治理决策的一致性修正器。你会看到：
1. 输入 snapshot
2. 首轮 JSON 决策
3. 外部一致性验证器指出的冲突点

请你重写一个新的最终 JSON 决策对象，并修复所有冲突。
要求：
1. 只输出 JSON，不要解释修改过程。
2. reason 必须与 allow_expand、expand_bytes、freeze_new_jobs 严格一致。
3. allow_expand=false 时 expand_bytes 必须为 0。
4. allow_expand=true 时 expand_bytes 必须为正数。
5. freeze_new_jobs=true 时 reason 必须明确写出冻结新作业的原因；freeze_new_jobs=false 时 reason 不得写成建议冻结。
6. 直接基于 snapshot 的证据重写，不要保留首轮输出里自相矛盾的描述。`
)

type storageDecisionRuntimeConfig struct {
	Mode         string
	ConfigSource string
	BaseURL      string
	APIKey       string
	ModelName    string
}

type directDecisionAttempt struct {
	Decision *LLMDecisionResponse
	RawJSON  string
	RawText  string
}

type directDecisionValidationIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GetStorageDecisionMode returns the active storage decision mode.
// Database config is used by default, while environment variables can still
// temporarily override it for local testing.
func GetStorageDecisionMode(ctx context.Context) string {
	cfg, err := loadStorageDecisionRuntimeConfig(ctx)
	if err != nil {
		klog.Warningf("GetStorageDecisionMode: failed to load runtime config, fallback to agent: %v", err)
		return StorageDecisionModeAgent
	}
	return normalizeStorageDecisionMode(cfg.Mode)
}

// AskDirectDecision sends a precomputed storage snapshot to a specialized model.
// It performs a first-pass generation, validates reason/decision consistency, and
// when conflicts are found, feeds them back to the model for one rewrite.
func AskDirectDecision(ctx context.Context, snapshotJSON string) (*LLMDecisionResponse, error) {
	llmConfig, err := loadDirectDecisionConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("加载运行时 LLM 配置失败: %w", err)
	}

	firstAttempt, err := askDirectDecisionOnce(ctx, *llmConfig, snapshotJSON)
	if err != nil {
		return nil, err
	}

	firstIssues := validateDirectDecisionConsistency(*firstAttempt.Decision)
	if len(firstIssues) == 0 {
		return firstAttempt.Decision, nil
	}

	klog.Warningf(
		"AskDirectDecision: first-pass direct decision has %d consistency issue(s): %s",
		len(firstIssues),
		joinValidationIssueMessages(firstIssues),
	)

	refinedAttempt, err := refineDirectDecision(
		ctx,
		*llmConfig,
		snapshotJSON,
		firstAttempt.RawJSON,
		firstIssues,
	)
	if err != nil {
		klog.Warningf("AskDirectDecision: refinement failed, keeping first-pass decision: %v", err)
		return firstAttempt.Decision, nil
	}

	refinedIssues := validateDirectDecisionConsistency(*refinedAttempt.Decision)
	switch {
	case len(refinedIssues) == 0:
		return refinedAttempt.Decision, nil
	case len(refinedIssues) < len(firstIssues):
		klog.Warningf(
			"AskDirectDecision: refined decision still has %d issue(s), but improved from %d: %s",
			len(refinedIssues),
			len(firstIssues),
			joinValidationIssueMessages(refinedIssues),
		)
		return refinedAttempt.Decision, nil
	default:
		klog.Warningf(
			"AskDirectDecision: refinement did not improve consistency, keeping first-pass decision. first=%d refined=%d refinedIssues=%s",
			len(firstIssues),
			len(refinedIssues),
			joinValidationIssueMessages(refinedIssues),
		)
		return firstAttempt.Decision, nil
	}
}

func askDirectDecisionOnce(ctx context.Context, cfg ProviderConfig, snapshotJSON string) (*directDecisionAttempt, error) {
	temperature := 0.0
	prompt := directDecisionInstruction + "\n\n输入:\n" + snapshotJSON + "\n\n输出:\n"
	resp, err := callConfiguredCompletion(ctx, cfg, completionRequest{
		Prompt:      prompt,
		MaxTokens:   256,
		Temperature: &temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("调用直连决策模型失败: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("直连决策模型未返回任何候选结果")
	}

	return decodeDirectDecisionAttempt(resp.Choices[0].Text)
}

func refineDirectDecision(
	ctx context.Context,
	cfg ProviderConfig,
	snapshotJSON string,
	firstDecisionJSON string,
	issues []directDecisionValidationIssue,
) (*directDecisionAttempt, error) {
	feedbackJSON, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal direct decision validation feedback: %w", err)
	}

	temperature := 0.0
	prompt := directDecisionRefinementInstruction +
		"\n\nsnapshot:\n" + snapshotJSON +
		"\n\n首轮决策:\n" + firstDecisionJSON +
		"\n\n一致性验证反馈:\n" + string(feedbackJSON) +
		"\n\n请输出修复后的 JSON 决策:\n"

	resp, err := callConfiguredCompletion(ctx, cfg, completionRequest{
		Prompt:      prompt,
		MaxTokens:   256,
		Temperature: &temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to refine direct decision: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("direct decision refinement returned no choices")
	}

	return decodeDirectDecisionAttempt(resp.Choices[0].Text)
}

func decodeDirectDecisionAttempt(rawText string) (*directDecisionAttempt, error) {
	rawJSON := extractFirstJSONObject(rawText)

	var decision LLMDecisionResponse
	if err := json.Unmarshal([]byte(rawJSON), &decision); err != nil {
		return nil, fmt.Errorf("解析直连决策 JSON 失败: %w\n原始响应: %s", err, rawText)
	}

	return &directDecisionAttempt{
		Decision: &decision,
		RawJSON:  rawJSON,
		RawText:  rawText,
	}, nil
}

func loadDirectDecisionConfig(ctx context.Context) (*ProviderConfig, error) {
	return GetStorageDecisionProviderConfig(ctx)
}

func GetStorageDecisionProviderConfig(ctx context.Context) (*ProviderConfig, error) {
	storageCfg, err := loadStorageDecisionRuntimeConfig(ctx)
	if err != nil {
		return nil, err
	}

	if storageCfg.ConfigSource == StorageDecisionConfigSourceCustom {
		if strings.TrimSpace(storageCfg.BaseURL) == "" {
			return nil, fmt.Errorf("storage decision base url is not configured")
		}
		if strings.TrimSpace(storageCfg.ModelName) == "" {
			return nil, fmt.Errorf("storage decision model name is not configured")
		}

		return &ProviderConfig{
			BaseURL:   strings.TrimSpace(storageCfg.BaseURL),
			APIKey:    strings.TrimSpace(storageCfg.APIKey),
			ModelName: strings.TrimSpace(storageCfg.ModelName),
		}, nil
	}

	return loadRuntimeLLMConfig(ctx)
}

func loadStorageDecisionRuntimeConfig(ctx context.Context) (*storageDecisionRuntimeConfig, error) {
	cfg := &storageDecisionRuntimeConfig{}

	var rows []model.SystemConfig
	err := query.GetDB().WithContext(ctx).
		Where("key IN ?", []string{
			model.ConfigKeyStorageDecisionMode,
			model.ConfigKeyStorageDecisionConfigSource,
			model.ConfigKeyStorageDirectModelBaseURL,
			model.ConfigKeyStorageDirectModelAPIKey,
			model.ConfigKeyStorageDirectModelName,
		}).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load storage decision config from database: %w", err)
	}

	configMap := make(map[string]string, len(rows))
	for _, row := range rows {
		configMap[row.Key] = strings.TrimSpace(row.Value)
	}

	cfg.Mode = normalizeStorageDecisionMode(configMap[model.ConfigKeyStorageDecisionMode])
	cfg.ConfigSource = normalizeStorageDecisionConfigSource(
		configMap[model.ConfigKeyStorageDecisionConfigSource],
		configMap[model.ConfigKeyStorageDirectModelName],
	)
	cfg.BaseURL = configMap[model.ConfigKeyStorageDirectModelBaseURL]
	cfg.ModelName = configMap[model.ConfigKeyStorageDirectModelName]

	if encryptedKey := configMap[model.ConfigKeyStorageDirectModelAPIKey]; encryptedKey != "" {
		plainKey, decryptErr := crypto.Decrypt(encryptedKey)
		if decryptErr != nil {
			klog.Warningf("loadStorageDecisionRuntimeConfig: failed to decrypt direct model api key, using raw value: %v", decryptErr)
			cfg.APIKey = encryptedKey
		} else {
			cfg.APIKey = plainKey
		}
	}

	if envMode := strings.TrimSpace(os.Getenv(StorageDecisionModeEnv)); envMode != "" {
		cfg.Mode = normalizeStorageDecisionMode(envMode)
	}
	if envSource := strings.TrimSpace(os.Getenv(StorageDecisionConfigSourceEnv)); envSource != "" {
		cfg.ConfigSource = normalizeStorageDecisionConfigSource(envSource, cfg.ModelName)
	}
	if envBaseURL := strings.TrimSpace(os.Getenv(DirectModelBaseURLEnv)); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}
	if envAPIKey := strings.TrimSpace(os.Getenv(DirectModelAPIKeyEnv)); envAPIKey != "" {
		cfg.APIKey = envAPIKey
	}
	if envModelName := strings.TrimSpace(os.Getenv(DirectModelNameEnv)); envModelName != "" {
		cfg.ModelName = envModelName
	}
	if cfg.ConfigSource == StorageDecisionConfigSourceCustom && strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = DefaultDirectModelBaseURL
	}

	return cfg, nil
}

func normalizeStorageDecisionMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case StorageDecisionModeDirect:
		return StorageDecisionModeDirect
	default:
		return StorageDecisionModeAgent
	}
}

func normalizeStorageDecisionConfigSource(source, modelName string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case StorageDecisionConfigSourceCustom:
		return StorageDecisionConfigSourceCustom
	case StorageDecisionConfigSourcePlatform:
		return StorageDecisionConfigSourcePlatform
	default:
		if strings.TrimSpace(modelName) != "" {
			return StorageDecisionConfigSourceCustom
		}
		return StorageDecisionConfigSourcePlatform
	}
}

func extractFirstJSONObject(text string) string {
	start := strings.Index(text, "{")
	if start == -1 {
		return strings.TrimSpace(text)
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		ch := text[i]

		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(text[start : i+1])
			}
		}
	}

	return strings.TrimSpace(text[start:])
}

func validateDirectDecisionConsistency(decision LLMDecisionResponse) []directDecisionValidationIssue {
	issues := make([]directDecisionValidationIssue, 0)
	reason := strings.TrimSpace(strings.ToLower(decision.Reason))

	if reason == "" {
		issues = append(issues, directDecisionValidationIssue{
			Code:    "empty_reason",
			Message: "reason is empty and does not explain the decision fields",
		})
	}

	if decision.AllowExpand && decision.ExpandBytes <= 0 {
		issues = append(issues, directDecisionValidationIssue{
			Code: "expand_bytes_missing",
			Message: fmt.Sprintf(
				"allow_expand=true but expand_bytes=%d; expansion decisions must use a positive expand_bytes",
				decision.ExpandBytes,
			),
		})
	}
	if !decision.AllowExpand && decision.ExpandBytes != 0 {
		issues = append(issues, directDecisionValidationIssue{
			Code:    "expand_bytes_should_be_zero",
			Message: fmt.Sprintf("allow_expand=false but expand_bytes=%d; expand_bytes must be 0 when expansion is disabled", decision.ExpandBytes),
		})
	}

	if reason == "" {
		return issues
	}

	hasExpandNegative := containsReasonPhrase(reason, directExpandNegativePhrases)
	hasExpandPositive := containsReasonPhrase(
		removeReasonPhrases(reason, directExpandNegativePhrases),
		directExpandPositivePhrases,
	)
	hasFreezeNegative := containsReasonPhrase(reason, directFreezeNegativePhrases)
	hasFreezePositive := containsReasonPhrase(
		removeReasonPhrases(reason, directFreezeNegativePhrases),
		directFreezePositivePhrases,
	)

	if decision.AllowExpand && hasExpandNegative {
		issues = append(issues, directDecisionValidationIssue{
			Code:    "reason_blocks_expansion",
			Message: "allow_expand=true but reason describes expansion as unnecessary, blocked, or forbidden",
		})
	}
	if !decision.AllowExpand && hasExpandPositive {
		issues = append(issues, directDecisionValidationIssue{
			Code:    "reason_supports_expansion",
			Message: "allow_expand=false but reason still recommends or supports expansion",
		})
	}
	if decision.FreezeNewJobs && hasFreezeNegative {
		issues = append(issues, directDecisionValidationIssue{
			Code:    "reason_blocks_freeze",
			Message: "freeze_new_jobs=true but reason says freezing new jobs is unnecessary or should not happen",
		})
	}
	if !decision.FreezeNewJobs && hasFreezePositive {
		issues = append(issues, directDecisionValidationIssue{
			Code:    "reason_supports_freeze",
			Message: "freeze_new_jobs=false but reason still recommends freezing or pausing new jobs",
		})
	}

	return issues
}

func joinValidationIssueMessages(issues []directDecisionValidationIssue) string {
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, issue.Code+": "+issue.Message)
	}
	return strings.Join(parts, "; ")
}

func containsReasonPhrase(reason string, phrases []string) bool {
	for _, phrase := range phrases {
		if strings.Contains(reason, phrase) {
			return true
		}
	}
	return false
}

func removeReasonPhrases(reason string, phrases []string) string {
	cleaned := reason
	for _, phrase := range phrases {
		cleaned = strings.ReplaceAll(cleaned, phrase, " ")
	}
	return cleaned
}

var directExpandPositivePhrases = []string{
	"建议扩容",
	"需要扩容",
	"应当扩容",
	"应该扩容",
	"可以扩容",
	"允许扩容",
	"优先扩容",
	"保守扩容",
	"临时扩容",
	"扩容保护",
	"expand",
	"allow expand",
}

var directExpandNegativePhrases = []string{
	"无需扩容",
	"不需要扩容",
	"不应扩容",
	"不应该扩容",
	"不建议扩容",
	"不能扩容",
	"不可扩容",
	"不允许扩容",
	"禁止扩容",
	"无需临时扩容",
	"no expand",
	"do not expand",
	"deny expansion",
}

var directFreezePositivePhrases = []string{
	"冻结新作业",
	"冻结新任务",
	"暂停新作业",
	"暂停新任务",
	"禁止新作业",
	"禁止新任务",
	"停止新作业",
	"停止新任务",
	"freeze new jobs",
	"freeze new job",
	"freeze jobs",
}

var directFreezeNegativePhrases = []string{
	"不冻结新作业",
	"不冻结新任务",
	"无需冻结",
	"不需要冻结",
	"不必冻结",
	"继续接受新作业",
	"继续提交新作业",
	"no freeze",
	"do not freeze",
}
