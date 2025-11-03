package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func PostJSON(ctx context.Context, host, path string, body any, header map[string]string, resp any) (err error) {
	url := fmt.Sprintf("%s%s", host, path)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("post failed, marshal body failed, err:%w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("post failed, new request: %w", err)
	}

	httpReq.Header.Add("Content-Type", "application/json")
	for k, v := range header {
		httpReq.Header.Add(k, v)
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("post failed, do request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("post read body:%w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("post failed, status code not ok, code:%v, body: %s", httpResp.StatusCode, string(respBody))
	}

	if err = json.Unmarshal(respBody, resp); err != nil {
		return fmt.Errorf("post unmarshal resp: %w", err)
	}

	return nil
}
