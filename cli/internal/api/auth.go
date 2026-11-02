package api

import "strings"

// AuthClient 认证相关 API 的抽象。默认实现为 *Client（真实 HTTP；若设置 CRATER_TEST_SANDBOX_HTTP 则传输层统一模拟，见本包 client.go）。测试可注入其它实现。
type AuthClient interface {
	Login(username, password, mode string) (*LoginResp, error)
}

// NewAuthClient 返回默认的 AuthClient（*Client）。命令层应依赖本函数或 AuthClient，便于测试中替换实现。
func NewAuthClient(baseURL string) AuthClient {
	return NewClient(baseURL)
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
	ID       uint   `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Nickname string `json:"nickname,omitempty"`

	Email     *string `json:"email,omitempty"`
	Teacher   *string `json:"teacher,omitempty"`
	Group     *string `json:"group,omitempty"`
	ExpiredAt *string `json:"expiredAt,omitempty"`

	Phone  *string `json:"phone,omitempty"`
	Avatar *string `json:"avatar,omitempty"`
	UID    *string `json:"uid,omitempty"`
	GID    *string `json:"gid,omitempty"`
}

// Login 用户登录（真实 HTTP）
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
		Post(AuthLoginPath)

	if err != nil {
		return nil, &NetworkError{Cause: err}
	}

	status := resp.GetStatusCode()
	if !resp.IsSuccessState() {
		return nil, &RequestError{
			HTTPStatus: status,
			CraterCode: result.Code,
			Msg:        result.Message,
		}
	}

	if result.Code != 0 {
		return nil, &RequestError{
			HTTPStatus: status,
			CraterCode: result.Code,
			Msg:        result.Message,
		}
	}

	return &result.Data, nil
}

type accountCheckResponse struct {
	Context *AccountContext `json:"context"`
}

// CurrentAccount returns the account context validated by the backend.
func (c *Client) CurrentAccount() (*AccountContext, error) {
	var result Response[*accountCheckResponse]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(AuthCheckPath)
	if err := errorFromRequest(resp, err, result.Code, result.Message); err != nil {
		return nil, err
	}
	if result.Data == nil || result.Data.Context == nil || strings.TrimSpace(result.Data.Context.Queue) == "" {
		return nil, &RequestError{HTTPStatus: resp.GetStatusCode(), Msg: "platform returned no current account context"}
	}
	return result.Data.Context, nil
}

// SwitchAccount requests credentials scoped to the named account/queue.
func (c *Client) SwitchAccount(queue string) (*LoginResp, error) {
	var result Response[*LoginResp]
	resp, err := c.httpClient.R().
		SetBody(struct {
			Queue string `json:"queue"`
		}{Queue: queue}).
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Post(AuthSwitchPath)
	if err := errorFromRequest(resp, err, result.Code, result.Message); err != nil {
		return nil, err
	}
	if result.Data == nil || strings.TrimSpace(result.Data.AccessToken) == "" || result.Data.Context.Queue != queue {
		return nil, &RequestError{HTTPStatus: resp.GetStatusCode(), Msg: "platform returned invalid account switch credentials or context"}
	}
	return result.Data, nil
}
