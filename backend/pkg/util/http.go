package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func PostJson(ctx context.Context, host, path string, body interface{}, header map[string]string, resp interface{}) (err error) {
	url := fmt.Sprintf("%s%s", host, path)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("post failed, marshal body failed, err:%v", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("post failed, new request failed, err:%v", err)
	}

	httpReq.Header.Add("Content-Type", "application/json")
	for k, v := range header {
		httpReq.Header.Add(k, v)
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("post failed, do request failed, err:%v", err)
	}

	respBody, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("post read body failed, err:%v", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("post failed, status code not ok, code:%v, body: %s", httpResp.StatusCode, string(respBody))
	}

	if err = json.Unmarshal(respBody, resp); err != nil {
		return fmt.Errorf("post unmarshal resp failed, err:%v", err)
	}

	return nil
}
