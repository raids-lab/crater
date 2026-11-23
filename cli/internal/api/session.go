package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/imroc/req/v3"
)

const (
	tokenExpiredCode = 40101
	tokenInvalidCode = 40102
)

// Tokens is the access and refresh pair returned by the authentication API.
type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type clientSession struct {
	tokens Tokens
	check  func(Tokens) error
	save   func(previous, next Tokens, requireActive bool) error
}

// SetSession enables renewal for requests made using one saved identity.
func (c *Client) SetSession(tokens Tokens, check func(Tokens) error, save func(Tokens, Tokens, bool) error) *Client {
	c.session = &clientSession{tokens: tokens, check: check, save: save}
	platform, _ := url.Parse(c.BaseURL)
	c.httpClient.GetTransport().WrapRoundTripFunc(func(next http.RoundTripper) req.HttpRoundTripFunc {
		return func(r *http.Request) (*http.Response, error) {
			if platform == nil || !strings.EqualFold(r.URL.Scheme, platform.Scheme) ||
				!strings.EqualFold(r.URL.Host, platform.Host) || r.URL.User != nil {
				return next.RoundTrip(r)
			}
			path := strings.TrimPrefix(r.URL.Path, strings.TrimRight(platform.Path, "/"))
			if path == AuthRefreshPath || path == AuthLoginPath {
				return next.RoundTrip(r)
			}
			return c.sessionRoundTrip(next, r, path)
		}
	})
	return c
}

func (c *Client) currentTokens() Tokens {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	if c.session == nil {
		return Tokens{AccessToken: c.token}
	}
	return c.session.tokens
}

// ReplaceCredentials saves credentials returned by an explicit account switch.
func (c *Client) ReplaceCredentials(tokens Tokens) error {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	return c.saveTokens(tokens, false)
}

// saveTokens is called with sessionMu held.
func (c *Client) saveTokens(tokens Tokens, requireActive bool) error {
	if err := c.session.save(c.session.tokens, tokens, requireActive); err != nil {
		return &SessionStateError{Cause: err}
	}
	c.session.tokens = tokens
	return nil
}

func (c *Client) renewAccessToken(ctx context.Context, rejected string) (string, error) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	tokens := c.session.tokens
	if err := ctx.Err(); err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := c.session.check(tokens); err != nil {
		return "", &SessionStateError{Cause: err}
	}
	if tokens.AccessToken != rejected {
		return tokens.AccessToken, nil
	}
	next, err := c.exchangeRefreshToken(ctx, tokens.RefreshToken)
	if err != nil {
		return "", err
	}
	if err := c.saveTokens(next, true); err != nil {
		return "", err
	}
	return next.AccessToken, nil
}

func (c *Client) exchangeRefreshToken(ctx context.Context, refreshToken string) (Tokens, error) {
	refreshClient := NewClient(c.BaseURL)
	refreshClient.httpClient.SetRedirectPolicy(req.NoRedirectPolicy())
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var result Response[Tokens]
	resp, err := refreshClient.httpClient.R().SetContext(ctx).
		SetBody(struct {
			RefreshToken string `json:"refreshToken"`
		}{RefreshToken: refreshToken}).
		SetSuccessResult(&result).SetErrorResult(&result).Post(AuthRefreshPath)
	if err := errorFromRequest(resp, err, result.Code, result.Message); err != nil {
		var failure *RequestError
		if errors.As(err, &failure) {
			failure.DuringRefresh = true
		}
		return Tokens{}, err
	}
	if strings.TrimSpace(result.Data.AccessToken) == "" || strings.TrimSpace(result.Data.RefreshToken) == "" {
		return Tokens{}, &RequestError{HTTPStatus: resp.GetStatusCode(), Msg: "platform returned incomplete refreshed credentials", DuringRefresh: true}
	}
	return result.Data, nil
}

func (c *Client) sessionRoundTrip(next http.RoundTripper, r *http.Request, path string) (*http.Response, error) {
	tokens := c.currentTokens()
	request := r.Clone(r.Context())
	request.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	resp, err := next.RoundTrip(request)
	// File methods prepare credentials before sending anything to storage.
	isStorage := path == StoragePrefix || strings.HasPrefix(path, StoragePrefix+"/")
	if err != nil || isStorage || tokens.RefreshToken == "" ||
		(r.Body != nil && r.Body != http.NoBody && (r.GetBody == nil || r.ContentLength <= 0)) ||
		!expiredResponse(resp) {
		return resp, err
	}
	_ = resp.Body.Close()
	accessToken, err := c.renewAccessToken(r.Context(), tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	retry := r.Clone(r.Context())
	retry.Header.Set("Authorization", "Bearer "+accessToken)
	if r.Body != nil && r.Body != http.NoBody {
		retry.Body, err = r.GetBody()
		if err != nil {
			return nil, err
		}
	}
	return next.RoundTrip(retry)
}

// Inspect bounded error bodies without consuming the response seen by callers.
func expiredResponse(resp *http.Response) bool {
	if resp == nil || resp.StatusCode != http.StatusUnauthorized || resp.Body == nil {
		return false
	}
	const limit = 32 << 10
	reader := bufio.NewReaderSize(resp.Body, limit+1)
	resp.Body = struct {
		io.Reader
		io.Closer
	}{reader, resp.Body}
	body, err := reader.Peek(limit + 1)
	var result Response[json.RawMessage]
	return errors.Is(err, io.EOF) && json.Unmarshal(body, &result) == nil && result.Code == tokenExpiredCode
}
