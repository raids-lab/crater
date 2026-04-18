package api

import (
	"fmt"

	"github.com/imroc/req/v3"
)

// Response 定义标准的 API 响应包装结构
type Response[T any] struct {
	Code    int    `json:"code"`
	Data    T      `json:"data"`
	Message string `json:"msg"`
}

// LoginReq 登录请求体
type LoginReq struct {
	AuthMethod string `json:"auth"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

// LoginResp 登录响应数据
type LoginResp struct {
	AccessToken  string         `json:"accessToken"`
	RefreshToken string         `json:"refreshToken"`
	Context      AccountContext `json:"context"`
	User         UserAttribute  `json:"user"`
}

// AccountContext 用户当前激活的上下文
type AccountContext struct {
	Queue        string `json:"queue"`
	RoleQueue    int    `json:"roleQueue"`
	RolePlatform int    `json:"rolePlatform"`
	AccessQueue  int    `json:"accessQueue"`
	AccessPublic int    `json:"accessPublic"`
	Space        string `json:"space"`
}

// UserAttribute 用户属性
type UserAttribute struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Nickname string  `json:"nickname"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Teacher  *string `json:"teacher"`
	Group    *string `json:"group"`
}

// Client Crater API 客户端
type Client struct {
	httpClient *req.Client
	BaseURL    string
}

// NewClient 初始化 API 客户端
func NewClient(baseURL string) *Client {
	c := req.C().SetBaseURL(baseURL)
	return &Client{
		httpClient: c,
		BaseURL:    baseURL,
	}
}

// SetToken 为后续请求设置 Bearer Token
func (c *Client) SetToken(token string) *Client {
	c.httpClient.SetCommonBearerAuthToken(token)
	return c
}

// Login 用户登录
func (c *Client) Login(username, password, mode string) (*LoginResp, error) {
	var result Response[LoginResp]

	resp, err := c.httpClient.R().
		SetBody(&LoginReq{
			AuthMethod: mode,
			Username:   username,
			Password:   password,
		}).
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Post("/api/auth/login")

	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}

	if !resp.IsSuccess() {
		if result.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", result.Code, result.Message)
		}
		return nil, fmt.Errorf("API error: status code %d", resp.GetStatusCode())
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error (%d): %s", result.Code, result.Message)
	}

	return &result.Data, nil
}
