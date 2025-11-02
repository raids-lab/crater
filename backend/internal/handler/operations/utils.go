package operations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/internal/handler"
)

func MergeURLWithQuery(baseURL string, queryParams map[string]string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("url.Parse failed to parse: %w", err)
	}
	q := u.Query()
	for key, value := range queryParams {
		q.Add(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func GetAdminTokenByLogin(_ *gin.Context, username, password string, serverHandler http.Handler) (string, error) {
	authReq := httptest.NewRequest(
		"POST", "/api/auth/login",
		bytes.NewBuffer([]byte(fmt.Sprintf("{\"auth\":\"normal\",\"username\":%q,\"password\":%q}", username, password))),
	)
	authReq.Header.Set("Content-Type", "application/json")
	authReq.Header.Set("accept", "application/json")
	authRecorder := httptest.NewRecorder()
	serverHandler.ServeHTTP(authRecorder, authReq)
	type AuthResp struct {
		Code int               `json:"code"`
		Data handler.LoginResp `json:"data"`
		Msg  string            `json:"msg"`
	}
	authResp := &AuthResp{}
	if err := json.Unmarshal(authRecorder.Body.Bytes(), authResp); err != nil {
		err := fmt.Errorf("json.Unmarshal failed: %w", err)
		klog.Error(err)
		return "", err
	}
	if authResp.Code != 0 {
		err := fmt.Errorf("authentication failed: %s", authResp.Msg)
		klog.Error(err)
		return "", err
	}
	AccessToken := authResp.Data.AccessToken
	return AccessToken, nil
}
