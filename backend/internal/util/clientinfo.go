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

package util

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
)

const (
	// ClientHeader is self-declared by clients. It is untrusted metadata and
	// must never be used for authorization, audit decisions, or rejection.
	ClientHeader = "X-Crater-Client"

	clientUserAgentPrefix = "crater-cli/"
	clientInfoKey         = "x-crater-client-info"
)

var clientVersionPattern = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._+-]{0,63}$`)

// ClientInfo describes the client that originated a request.
type ClientInfo struct {
	Type    model.ClientType
	Version string
}

// ParseClientInfo resolves the client type and best-effort version from request
// headers. An explicit X-Crater-Client value takes precedence over User-Agent;
// the User-Agent fallback exists for CLI releases that predate the header.
func ParseClientInfo(header http.Header) ClientInfo {
	rawType := strings.TrimSpace(header.Get(ClientHeader))
	if rawType != "" {
		switch NormalizeClientType(rawType) {
		case model.ClientTypeCLI:
			info := ClientInfo{Type: model.ClientTypeCLI}
			if version, matched := clientVersionFromUserAgent(header.Get("User-Agent")); matched {
				info.Version = version
			}
			return info
		case model.ClientTypeWeb:
			return ClientInfo{Type: model.ClientTypeWeb}
		default:
			return ClientInfo{Type: model.ClientTypeUnknown}
		}
	}

	if version, matched := clientVersionFromUserAgent(header.Get("User-Agent")); matched {
		return ClientInfo{Type: model.ClientTypeCLI, Version: version}
	}
	return ClientInfo{Type: model.ClientTypeUnknown}
}

// NormalizeClientType maps arbitrary input to one of the persisted values.
func NormalizeClientType(raw string) model.ClientType {
	switch model.ClientType(strings.ToLower(strings.TrimSpace(raw))) {
	case model.ClientTypeCLI:
		return model.ClientTypeCLI
	case model.ClientTypeWeb:
		return model.ClientTypeWeb
	default:
		return model.ClientTypeUnknown
	}
}

// NormalizeClientVersion keeps only short, token-like version strings.
func NormalizeClientVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !clientVersionPattern.MatchString(raw) {
		return ""
	}
	return raw
}

func clientVersionFromUserAgent(userAgent string) (string, bool) {
	userAgent = strings.TrimSpace(userAgent)
	if len(userAgent) < len(clientUserAgentPrefix) ||
		!strings.EqualFold(userAgent[:len(clientUserAgentPrefix)], clientUserAgentPrefix) {
		return "", false
	}
	version := userAgent[len(clientUserAgentPrefix):]
	if index := strings.IndexAny(version, " \t;"); index >= 0 {
		version = version[:index]
	}
	return NormalizeClientVersion(version), true
}

// SetClientInfo stores the parsed client identity in the gin request context.
func SetClientInfo(c *gin.Context, info ClientInfo) {
	if c == nil {
		return
	}
	info.Type = NormalizeClientType(string(info.Type))
	if info.Type != model.ClientTypeCLI {
		info.Version = ""
	} else {
		info.Version = NormalizeClientVersion(info.Version)
	}
	c.Set(clientInfoKey, info)
}

// GetClientInfo returns the parsed client identity, or unknown when the
// middleware did not run or the value is not a ClientInfo.
func GetClientInfo(c *gin.Context) ClientInfo {
	if c == nil {
		return ClientInfo{Type: model.ClientTypeUnknown}
	}
	value, ok := c.Get(clientInfoKey)
	if !ok {
		return ClientInfo{Type: model.ClientTypeUnknown}
	}
	info, ok := value.(ClientInfo)
	if !ok {
		return ClientInfo{Type: model.ClientTypeUnknown}
	}
	info.Type = NormalizeClientType(string(info.Type))
	if info.Type != model.ClientTypeCLI {
		info.Version = ""
	} else {
		info.Version = NormalizeClientVersion(info.Version)
	}
	return info
}
