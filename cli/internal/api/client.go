package api

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/imroc/req/v3"
	"github.com/raids-lab/crater/cli/internal/testenv"
	internalversion "github.com/raids-lab/crater/cli/internal/version"
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
// - CRATER_TEST_SANDBOX_HTTP=timeout                      => timeout
// - CRATER_TEST_SANDBOX_HTTP=error404                     => 404
// - CRATER_TEST_SANDBOX_HTTP=passthrough                  => loopback snapshot fixture
// - CRATER_TEST_SANDBOX_HTTP=compatibility-compatible     => compatible API versions
// - CRATER_TEST_SANDBOX_HTTP=compatibility-cli-too-old    => platform requires a newer CLI API
// - CRATER_TEST_SANDBOX_HTTP=compatibility-backend-zero   => platform explicitly reports API versions 0/0
// - CRATER_TEST_SANDBOX_HTTP=compatibility-unavailable    => compatibility endpoint is missing
func applyHTTPSim(rc *req.Client) {
	mode := testenv.SandboxHTTPMode()
	switch mode {
	case "error404", "404":
		wrapSim404(rc)
	case "timeout", "hang":
		wrapSimTimeout(rc)
	case "passthrough":
		wrapLoopbackPassthrough(rc)
	case "compatibility-compatible":
		// The fixture backend keeps support for CLI contract v3 when the current contract advances.
		wrapSimCompatibility(rc, internalversion.APIVersion, 3)
	case "compatibility-cli-too-old":
		wrapSimCompatibility(rc, internalversion.APIVersion, internalversion.APIVersion+1)
	case "compatibility-backend-zero":
		wrapSimCompatibility(rc, 0, 0)
	case "compatibility-unavailable":
		wrapSimCompatibility(rc, -1, -1)
	default:
	}
}

func wrapLoopbackPassthrough(rc *req.Client) {
	rc.GetTransport().WrapRoundTripFunc(func(next http.RoundTripper) req.HttpRoundTripFunc {
		return func(r *http.Request) (*http.Response, error) {
			host := r.URL.Hostname()
			ip := net.ParseIP(host)
			if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
				return nil, fmt.Errorf(
					"test sandbox passthrough rejected non-loopback host %q",
					host,
				)
			}
			return next.RoundTrip(r)
		}
	})
}

func wrapSim404(rc *req.Client) {
	rc.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(r *http.Request) (*http.Response, error) {
			return simulatedJSONResponse(r, http.StatusNotFound, `{"code":404,"data":null,"msg":"simulated"}`), nil
		}
	})
}

func wrapSimCompatibility(rc *req.Client, apiVersion, minSupportedCLIAPIVersion int) {
	rc.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case CompatibilityPath:
				if apiVersion >= 0 && minSupportedCLIAPIVersion >= 0 {
					body := `{"code":0,"data":{"apiVersion":` + strconv.Itoa(apiVersion) +
						`,"minSupportedCliApiVersion":` + strconv.Itoa(minSupportedCLIAPIVersion) +
						`,"appVersion":"v1.1.1","shortCommitSHA":"f42b0c2","buildType":"release",` +
						`"buildTime":"2026-07-26T08:30:00Z"},"msg":""}`
					return simulatedJSONResponse(r, http.StatusOK, body), nil
				}
				return simulatedResponse(r, http.StatusNotFound, "text/plain", "404 page not found"), nil
			default:
				return simulatedJSONResponse(r, http.StatusNotFound, `{"code":404,"data":null,"msg":"simulated"}`), nil
			}
		}
	})
}

func simulatedJSONResponse(r *http.Request, statusCode int, body string) *http.Response {
	return simulatedResponse(r, statusCode, "application/json", body)
}

func simulatedResponse(r *http.Request, statusCode int, contentType, body string) *http.Response {
	return &http.Response{
		StatusCode:    statusCode,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Status:        strconv.Itoa(statusCode) + " " + http.StatusText(statusCode),
		Body:          io.NopCloser(strings.NewReader(body)),
		Header:        http.Header{"Content-Type": []string{contentType}},
		ContentLength: int64(len(body)),
		Request:       r,
	}
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
	rc := req.C().
		SetBaseURL(baseURL).
		SetUserAgent(internalversion.UserAgent()).
		SetCommonHeader(internalversion.APIVersionHeader, strconv.Itoa(internalversion.APIVersion))
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
