package api

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/imroc/req/v3"
	"github.com/raids-lab/crater/cli/internal/testenv"
)

// Response 定义标准的 API 响应包装结构
type Response[T any] struct {
	Code    int    `json:"code"`
	Data    T      `json:"data"`
	Message string `json:"msg"`
}

// Client Crater API 客户端（真实 HTTP）
type Client struct {
	httpClient *req.Client
	BaseURL    string
}

// applyHTTPSim 按环境变量在 req Transport 上注册拦截（仅影响经 NewClient 创建的客户端）。
//
// - CRATER_TEST_SANDBOX_HTTP=timeout     => timeout
// - CRATER_TEST_SANDBOX_HTTP=error404    => 404
func applyHTTPSim(rc *req.Client) {
	mode := testenv.SandboxHTTPMode()
	switch strings.TrimSpace(mode) {
	case "error404", "404":
		wrapSim404(rc)
	case "timeout", "hang":
		wrapSimTimeout(rc)
	default:
	}
}

func wrapSim404(rc *req.Client) {
	rc.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(r *http.Request) (*http.Response, error) {
			body := `{"code":404,"data":null,"msg":"simulated"}`
			return &http.Response{
				StatusCode:    http.StatusNotFound,
				ProtoMajor:    1,
				ProtoMinor:    1,
				Status:        "404 Not Found",
				Body:          io.NopCloser(strings.NewReader(body)),
				Header:        http.Header{"Content-Type": []string{"application/json"}},
				ContentLength: int64(len(body)),
				Request:       r,
			}, nil
		}
	})
}

func wrapSimTimeout(rc *req.Client) {
	rc.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(r *http.Request) (*http.Response, error) {
			_ = r
			return nil, context.DeadlineExceeded
		}
	})
}

// NewClient 初始化 API 客户端
func NewClient(baseURL string) *Client {
	rc := req.C().SetBaseURL(baseURL)
	applyHTTPSim(rc)
	return &Client{
		httpClient: rc,
		BaseURL:    baseURL,
	}
}

// SetToken 为后续请求设置 Bearer Token
func (c *Client) SetToken(token string) *Client {
	c.httpClient.SetCommonBearerAuthToken(token)
	return c
}
