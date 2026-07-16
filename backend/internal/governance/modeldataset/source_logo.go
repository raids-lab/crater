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

package modeldataset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const (
	maxLogoRedirects       = 5
	maxModelScopePageBytes = 2 * 1024 * 1024
	maxAvatarResponseBytes = 1024 * 1024
)

var modelScopeAvatarPattern = regexp.MustCompile(
	`https://resou(?:r)?ces\.modelscope\.cn/avatar/[A-Za-z0-9._%/-]+`,
)

// FetchHuggingFaceAvatarURL resolves an organization or user avatar from the
// same official metadata endpoints used by the batch metadata refresher.
func FetchHuggingFaceAvatarURL(
	ctx context.Context, client *http.Client, baseEndpoints []string, owner string,
) (string, error) {
	escapedOwner := url.PathEscape(owner)
	var lastErr error
	for _, baseEndpoint := range baseEndpoints {
		endpoints := []string{
			strings.TrimRight(baseEndpoint, "/") + "/api/organizations/" + escapedOwner + "/overview",
			strings.TrimRight(baseEndpoint, "/") + "/api/users/" + escapedOwner + "/overview",
		}
		for _, endpoint := range endpoints {
			response, err := getSourceResponse(ctx, client, endpoint)
			if err != nil {
				lastErr = err
				continue
			}
			if response.StatusCode == http.StatusNotFound {
				response.Body.Close()
				continue
			}
			if response.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("source returned HTTP %d", response.StatusCode)
				response.Body.Close()
				continue
			}
			var payload struct {
				AvatarURL string `json:"avatarUrl"`
			}
			decodeErr := json.NewDecoder(io.LimitReader(response.Body, maxAvatarResponseBytes)).Decode(&payload)
			response.Body.Close()
			if decodeErr != nil {
				lastErr = decodeErr
				continue
			}
			return payload.AvatarURL, nil
		}
	}
	return "", lastErr
}

// FetchModelScopeAvatarURL extracts the organization avatar rendered in a
// ModelScope repository page because its OpenAPI response currently omits it.
func FetchModelScopeAvatarURL(
	ctx context.Context, client *http.Client, repositoryURL string,
) (string, error) {
	response, err := getSourceResponse(ctx, client, repositoryURL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("source returned HTTP %d", response.StatusCode)
	}
	page, err := io.ReadAll(io.LimitReader(response.Body, maxModelScopePageBytes+1))
	if err != nil {
		return "", err
	}
	if len(page) > maxModelScopePageBytes {
		return "", errors.New("ModelScope repository page is too large")
	}
	normalized := stdhtml.UnescapeString(strings.ReplaceAll(string(page), `\u002F`, "/"))
	normalized = strings.ReplaceAll(normalized, `\/`, "/")
	avatarURL := modelScopeAvatarPattern.FindString(normalized)
	if avatarURL == "" {
		return "", nil
	}
	parsedAvatarURL, err := url.Parse(avatarURL)
	if err != nil {
		return "", err
	}
	query := parsedAvatarURL.Query()
	query.Set("x-oss-process", "image/resize,m_lfit,w_128,h_128")
	parsedAvatarURL.RawQuery = query.Encode()
	return parsedAvatarURL.String(), nil
}

// FetchSourceLogo downloads and validates a source logo. Every redirect is
// checked against the exact host allowlist to prevent metadata-driven SSRF.
func FetchSourceLogo(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	allowedHosts []string,
	maxBytes int64,
) (data []byte, contentType string, err error) {
	if err := ValidateSourceLogoURL(endpoint, allowedHosts); err != nil {
		return nil, "", err
	}
	logoClient := *client
	logoClient.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= maxLogoRedirects {
			return errors.New("logo source exceeded redirect limit")
		}
		return ValidateSourceLogoURL(request.URL.String(), allowedHosts)
	}
	response, err := getSourceResponse(ctx, &logoClient, endpoint)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("logo source returned HTTP %d", response.StatusCode)
	}
	contentType = strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	data, err = io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxBytes {
		return nil, "", fmt.Errorf("logo exceeds %d bytes", maxBytes)
	}
	if !strings.HasPrefix(contentType, "image/") {
		detectedContentType := strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0])
		if !strings.HasPrefix(detectedContentType, "image/") {
			return nil, "", fmt.Errorf("logo source returned unsupported Content-Type %q", contentType)
		}
		contentType = detectedContentType
	}
	return data, contentType, nil
}

func ValidateSourceLogoURL(endpoint string, allowedHosts []string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid logo URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("logo URL must be an absolute HTTPS URL without credentials")
	}

	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	for _, allowedHost := range allowedHosts {
		allowedHost = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(allowedHost), "."))
		if host == allowedHost {
			return nil
		}
	}
	return fmt.Errorf("logo host %q is not allowed", host)
}

func getSourceResponse(ctx context.Context, client *http.Client, endpoint string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	return client.Do(request)
}
