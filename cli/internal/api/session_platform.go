// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/imroc/req/v3"
)

// NewSessionClient resolves the platform address before attaching credentials.
func NewSessionClient(ctx context.Context, baseURL string, tokens Tokens, check func(Tokens) error, save func(Tokens, Tokens, bool) error) (*Client, error) {
	client := NewClient(baseURL)
	if err := client.resolveSessionPlatform(ctx); err != nil {
		return nil, err
	}
	return client.SetSession(tokens, check, save), nil
}

func (c *Client) resolveSessionPlatform(ctx context.Context) error {
	platform, err := url.Parse(c.BaseURL)
	if err != nil || platform.Scheme != "http" || platform.User != nil {
		return nil
	}
	host := platform.Hostname()
	if host == "" || strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback() {
		return nil
	}

	// Probe the same API method and path that the backend serves for account checks.
	probe := c.httpClient.Clone().SetRedirectPolicy(req.NoRedirectPolicy())
	defer probe.GetTransport().CloseIdleConnections()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := probe.R().SetContext(ctx).DisableAutoReadResponse().Get(AuthCheckPath)
	if err != nil {
		return &NetworkError{Cause: err}
	}
	defer resp.Body.Close()
	switch resp.GetStatusCode() {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		target, err := resp.Response.Location()
		from := resp.Request.URL
		if err != nil || target.Scheme != "https" || target.User != nil ||
			(platform.Port() != "" && platform.Port() != "80") ||
			(target.Port() != "" && target.Port() != "443") ||
			!strings.EqualFold(host, target.Hostname()) ||
			target.EscapedPath() != from.EscapedPath() || target.RawQuery != from.RawQuery {
			return &RequestError{
				HTTPStatus: resp.GetStatusCode(),
				Msg:        "platform redirected the API to a different address; use the platform's final URL",
			}
		}
		platform.Scheme, platform.Host = target.Scheme, target.Host
		c.BaseURL = platform.String()
		c.httpClient.SetBaseURL(c.BaseURL)
	}
	return nil
}
