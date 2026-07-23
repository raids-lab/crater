package prompts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	bytesPerKiB                    = 1 << 10
	streamScannerInitialBufferSize = 64 * bytesPerKiB
	streamScannerMaxBufferSize     = bytesPerKiB * bytesPerKiB
	llmErrorBodyReadLimitBytes     = 4 * bytesPerKiB
)

// --- LLM 通用结构体 ---

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponseFormat struct {
	Type string `json:"type"` // e.g., "json_object"
}

type LLMRequestPayload struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
}

func executeLLMRequest(
	httpClient *http.Client,
	ctx context.Context,
	apiURL string,
	apiKey string,
	payload LLMRequestPayload,
) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal LLM request payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create LLM request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute LLM request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API returned non-200 status code: %d", resp.StatusCode)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode LLM API response wrapper: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("LLM response contained no choices")
	}

	return strings.TrimSpace(apiResp.Choices[0].Message.Content), nil
}

// CallLLMAPI 是一个通用的 LLM API 调用泛型函数
func CallLLMAPI[T any](
	httpClient *http.Client,
	ctx context.Context,
	apiURL string,
	apiKey string,
	modelName string,
	systemPrompt, userPrompt string,
) (*T, error) {
	payload := LLMRequestPayload{
		Model: modelName,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	rawContent, err := executeLLMRequest(httpClient, ctx, apiURL, apiKey, payload)
	if err != nil {
		return nil, err
	}
	cleanedContent := CleanLLMJSONOutput(rawContent)

	var result T
	if err := json.Unmarshal([]byte(cleanedContent), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content JSON: %w (raw: %s)", err, rawContent)
	}

	return &result, nil
}

func CallLLMText(
	httpClient *http.Client,
	ctx context.Context,
	apiURL string,
	apiKey string,
	modelName string,
	systemPrompt, userPrompt string,
) (string, error) {
	payload := LLMRequestPayload{
		Model: modelName,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	return executeLLMRequest(httpClient, ctx, apiURL, apiKey, payload)
}

func CheckLLMAvailable(
	httpClient *http.Client,
	ctx context.Context,
	apiURL string,
	apiKey string,
	modelName string,
) error {
	payload := LLMRequestPayload{
		Model: modelName,
		Messages: []Message{
			{Role: "user", Content: "ping"},
		},
		MaxTokens: 1,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal LLM health check payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create LLM health check request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute LLM health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LLM API returned non-200 status code: %d%s", resp.StatusCode, readLLMErrorBody(resp))
	}
	return nil
}

func CallLLMTextStream(
	httpClient *http.Client,
	ctx context.Context,
	apiURL string,
	apiKey string,
	modelName string,
	systemPrompt, userPrompt string,
	onDelta func(string) error,
) (string, error) {
	req, err := newLLMStreamRequest(ctx, apiURL, apiKey, modelName, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute LLM stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API returned non-200 status code: %d%s", resp.StatusCode, readLLMErrorBody(resp))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, streamScannerInitialBufferSize), streamScannerMaxBufferSize)
	for scanner.Scan() {
		data, ok := parseStreamDataLine(scanner.Text())
		if !ok {
			continue
		}
		if data == "[DONE]" {
			break
		}
		if err := appendLLMStreamDelta(&full, data, onDelta); err != nil {
			return full.String(), err
		}
	}
	if err := scanner.Err(); err != nil {
		return full.String(), fmt.Errorf("failed to read LLM stream: %w", err)
	}
	return strings.TrimSpace(full.String()), nil
}

func readLLMErrorBody(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, llmErrorBodyReadLimitBytes))
	if err != nil {
		return ""
	}
	body := strings.TrimSpace(string(bodyBytes))
	if body == "" {
		return ""
	}
	return ": " + body
}

func newLLMStreamRequest(
	ctx context.Context,
	apiURL string,
	apiKey string,
	modelName string,
	systemPrompt string,
	userPrompt string,
) (*http.Request, error) {
	payload := LLMRequestPayload{
		Model: modelName,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: true,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LLM stream request payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	return req, nil
}

func parseStreamDataLine(rawLine string) (string, bool) {
	line := strings.TrimSpace(rawLine)
	if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(line, "data:")), true
}

func appendLLMStreamDelta(full *strings.Builder, data string, onDelta func(string) error) error {
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return fmt.Errorf("failed to decode LLM stream chunk: %w", err)
	}
	if len(chunk.Choices) == 0 {
		return nil
	}
	delta := chunk.Choices[0].Delta.Content
	if delta == "" {
		return nil
	}
	full.WriteString(delta)
	if onDelta != nil {
		return onDelta(delta)
	}
	return nil
}

// CleanLLMJSONOutput 移除可能存在的 Markdown 代码块标记
func CleanLLMJSONOutput(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}
