package storagequota

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	InternalTokenHeader = "X-Crater-Internal-Token" //nolint:gosec // This is a header name, not a credential.
	InternalTokenEnv    = "CRATER_STORAGE_INTERNAL_TOKEN"
	InternalSecretEnv   = "CRATER_STORAGE_INTERNAL_SECRET"
	ServerURLEnv        = "CRATER_STORAGE_QUOTA_SERVER_URL"

	ProviderAuto          = "auto"
	ProviderStorageServer = "storageServer"
	ProviderToolbox       = "toolbox"
	ProviderDisabled      = "disabled"
	maxResponseBodyBytes  = 1 << 20
)

type Capabilities struct {
	UsageReadable bool     `json:"usage_readable"`
	QuotaReadable bool     `json:"quota_readable"`
	QuotaWritable bool     `json:"quota_writable"`
	Reasons       []string `json:"reasons,omitempty"`
}

type Usage struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

type Quota struct {
	Path     string `json:"path"`
	MaxBytes int64  `json:"max_bytes"`
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, accessTokenSecret string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   DeriveInternalToken(accessTokenSecret),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func ResolveServerURL(configuredURL, namespace string) string {
	if envURL := strings.TrimSpace(os.Getenv(ServerURLEnv)); envURL != "" {
		return strings.TrimRight(envURL, "/")
	}
	if configuredURL = strings.TrimSpace(configuredURL); configuredURL != "" {
		return strings.TrimRight(configuredURL, "/")
	}
	if namespace = strings.TrimSpace(namespace); namespace != "" {
		return fmt.Sprintf("http://webdav-service.%s.svc:7320", namespace)
	}
	return "http://webdav-service:7320"
}

func NormalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", ProviderAuto:
		return ProviderAuto
	case strings.ToLower(ProviderStorageServer), "storage-server":
		return ProviderStorageServer
	case ProviderToolbox:
		return ProviderToolbox
	case ProviderDisabled:
		return ProviderDisabled
	default:
		return ProviderDisabled
	}
}

func DeriveInternalToken(accessTokenSecret string) string {
	sum := sha256.Sum256([]byte("crater-storage-quota:" + accessTokenSecret))
	return hex.EncodeToString(sum[:])
}

func Authenticate(accessTokenSecret, suppliedToken string) bool {
	return AuthenticateToken(DeriveInternalToken(accessTokenSecret), suppliedToken)
}

func AuthenticateToken(expectedToken, suppliedToken string) bool {
	expected := []byte(strings.TrimSpace(expectedToken))
	supplied := []byte(strings.TrimSpace(suppliedToken))
	return len(expected) == len(supplied) && subtle.ConstantTimeCompare(expected, supplied) == 1
}

func (c *Client) GetCapabilities(ctx context.Context) (Capabilities, error) {
	var result Capabilities
	err := c.do(ctx, http.MethodGet, "/internal/storage/capabilities", nil, nil, &result)
	return result, err
}

func (c *Client) GetUsage(ctx context.Context, relativePath string) (Usage, error) {
	var result Usage
	err := c.do(ctx, http.MethodGet, "/internal/storage/usage", url.Values{"path": {relativePath}}, nil, &result)
	return result, err
}

func (c *Client) GetQuota(ctx context.Context, relativePath string) (Quota, error) {
	var result Quota
	err := c.do(ctx, http.MethodGet, "/internal/storage/quota", url.Values{"path": {relativePath}}, nil, &result)
	return result, err
}

func (c *Client) SetQuota(ctx context.Context, relativePath string, maxBytes int64) (Quota, error) {
	var result Quota
	err := c.do(ctx, http.MethodPut, "/internal/storage/quota", nil, Quota{
		Path:     relativePath,
		MaxBytes: maxBytes,
	}, &result)
	return result, err
}

func (c *Client) do(
	ctx context.Context,
	method, endpoint string,
	query url.Values,
	body any,
	result any,
) error {
	if c.baseURL == "" {
		return fmt.Errorf("storage quota server URL is empty")
	}

	requestURL := c.baseURL + endpoint
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	var bodyReader io.Reader = http.NoBody
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal storage quota request: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create storage quota request: %w", err)
	}
	req.Header.Set(InternalTokenHeader, c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call storage quota server: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return fmt.Errorf("read storage quota response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var errorBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(payload, &errorBody)
		if errorBody.Error == "" {
			errorBody.Error = strings.TrimSpace(string(payload))
		}
		return fmt.Errorf("storage quota server returned %s: %s", resp.Status, errorBody.Error)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(payload, result); err != nil {
		return fmt.Errorf("decode storage quota response: %w", err)
	}
	return nil
}
