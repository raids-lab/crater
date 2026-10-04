package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/bizerr"
)

func wantsKthenaStream(c *gin.Context) bool {
	return strings.Contains(c.GetHeader("Accept"), "text/event-stream")
}

func writeKthenaEvent(c *gin.Context, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func (mgr *KthenaMgr) streamKthenaTurn(ctx context.Context, body []byte, delta func(string) error) ([]byte, error) {
	var request map[string]any
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, err
	}
	request["stream"] = true
	request["stream_options"] = map[string]any{"include_usage": true}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	response,
		err := mgr.openKthenaRouterResponse(ctx,
		http.MethodPost,
		kthenaChatCompletionsPath,
		encoded,
		http.Header{"Accept": []string{"text/event-stream"}})
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, bizerr.Conflict.ResourceStatusError.New(fmt.Sprintf("inference router returned HTTP %d", response.StatusCode))
	}
	return readKthenaCompletionStream(response.Body, delta)
}

// Partial output is never persisted as a successful assistant message.
func readKthenaCompletionStream(reader io.Reader, delta func(string) error) ([]byte, error) {
	scanner := bufio.NewScanner(reader)
	const initialBufferSize = 4096
	scanner.Buffer(make([]byte, initialBufferSize), kthenaMaxProxyBodyBytes)
	var content strings.Builder
	var usage any
	var id, modelName string
	completed := false
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			completed = true
			break
		}
		var chunk struct {
			ID      string           `json:"id"`
			Model   string           `json:"model"`
			Usage   any              `json:"usage"`
			Error   *json.RawMessage `json:"error"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, err
		}
		if chunk.Error != nil {
			return nil, bizerr.Internal.ServiceError.New("inference stream reported an error")
		}
		if chunk.ID != "" {
			id = chunk.ID
		}
		if chunk.Model != "" {
			modelName = chunk.Model
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
		if len(chunk.Choices) > 0 {
			text := chunk.Choices[0].Delta.Content
			if content.Len()+len(text) > kthenaMaxProxyBodyBytes {
				return nil, bizerr.BadRequest.ParameterError.New("completion exceeds size limit")
			}
			content.WriteString(text)
			if text != "" {
				if err := delta(text); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !completed {
		return nil, io.ErrUnexpectedEOF
	}
	return json.Marshal(map[string]any{"id": id,
		"model": modelName,
		"usage": usage,
		"choices": []any{map[string]any{"message": map[string]any{"role": "assistant",
			"content": content.String()}}}})
}
