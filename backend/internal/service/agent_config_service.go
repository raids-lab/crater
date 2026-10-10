package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/pkg/crypto"
)

type LLMConfigSource string

const (
	LLMConfigSourcePersonal LLMConfigSource = "personal"
	LLMConfigSourcePlatform LLMConfigSource = "platform"
)

type EffectiveLLMConfig struct {
	LLMConfig
	Source LLMConfigSource
}

type PersonalLLMConfigStatus struct {
	PersonalConfigured bool
	PlatformConfigured bool
	EffectiveSource    LLMConfigSource
	BaseURL            string
	ModelName          string
	Personal           *LLMConfig
}

type AgentConfigService struct {
	db             *gorm.DB
	platformConfig *ConfigService
}

const (
	personalLLMCheckResponseLimit = 2 << 20
	personalLLMCheckTimeout       = 10 * time.Second
)

func NewAgentConfigService(db *gorm.DB, platformConfig *ConfigService) *AgentConfigService {
	return &AgentConfigService{db: db, platformConfig: platformConfig}
}

func (s *AgentConfigService) GetStatus(
	ctx context.Context,
	userID uint,
	_ AgentIdentity,
) (*PersonalLLMConfigStatus, error) {
	personal, found, err := s.getPersonalConfig(ctx, userID)
	if err != nil {
		return nil, err
	}
	platform, err := s.platformConfig.GetLLMConfig(ctx)
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to load platform LLM configuration")
	}

	status := &PersonalLLMConfigStatus{
		PersonalConfigured: found && isCompleteLLMConfig(personal),
		PlatformConfigured: isCompleteLLMConfig(platform),
		Personal:           personal,
	}
	if status.PersonalConfigured {
		status.EffectiveSource = LLMConfigSourcePersonal
		status.BaseURL = personal.BaseURL
		status.ModelName = personal.ModelName
	} else if status.PlatformConfigured {
		status.EffectiveSource = LLMConfigSourcePlatform
		status.BaseURL = platform.BaseURL
		status.ModelName = platform.ModelName
	}
	return status, nil
}

func (s *AgentConfigService) ResolveEffectiveConfig(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
) (*EffectiveLLMConfig, error) {
	status, err := s.GetStatus(ctx, userID, identity)
	if err != nil {
		return nil, err
	}
	if status.PersonalConfigured {
		return &EffectiveLLMConfig{LLMConfig: *status.Personal, Source: LLMConfigSourcePersonal}, nil
	}
	if status.PlatformConfigured {
		platform, err := s.platformConfig.GetLLMConfig(ctx)
		if err != nil {
			return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to load platform LLM configuration")
		}
		return &EffectiveLLMConfig{LLMConfig: *platform, Source: LLMConfigSourcePlatform}, nil
	}
	return nil, bizerr.ServiceError.ServiceUnavailable.New(
		"LLM is not configured. Configure a personal model or contact an administrator.",
	)
}

func (s *AgentConfigService) SavePersonalConfig(
	ctx context.Context,
	userID uint,
	_ AgentIdentity,
	cfg *LLMConfig,
	validate bool,
) error {
	resolved, err := s.resolveSubmittedConfig(ctx, userID, cfg)
	if err != nil {
		return err
	}
	if err := validatePersonalLLMConfig(resolved); err != nil {
		return err
	}
	if validate {
		if err := checkPersonalLLMConnection(ctx, resolved); err != nil {
			return bizerr.Conflict.ResourceStatusError.Wrap(err, "personal LLM connection check failed")
		}
	}

	encryptedKey, err := crypto.Encrypt(resolved.APIKey)
	if err != nil {
		return bizerr.Internal.ServiceError.Wrap(err, "failed to encrypt personal API key")
	}
	record := &model.UserLLMConfig{
		UserID:          userID,
		BaseURL:         strings.TrimRight(strings.TrimSpace(resolved.BaseURL), "/"),
		ModelName:       strings.TrimSpace(resolved.ModelName),
		EncryptedAPIKey: encryptedKey,
	}
	result := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Assign(record).
		FirstOrCreate(record)
	if result.Error != nil {
		return bizerr.Internal.DatabaseError.Wrap(result.Error, "failed to save personal LLM configuration")
	}
	return nil
}

func (s *AgentConfigService) CheckPersonalConfig(
	ctx context.Context,
	userID uint,
	_ AgentIdentity,
	cfg *LLMConfig,
) error {
	resolved, err := s.resolveSubmittedConfig(ctx, userID, cfg)
	if err != nil {
		return err
	}
	if err := validatePersonalLLMConfig(resolved); err != nil {
		return err
	}
	if err := checkPersonalLLMConnection(ctx, resolved); err != nil {
		return bizerr.Conflict.ResourceStatusError.Wrap(err, "personal LLM connection check failed")
	}
	return nil
}

func (s *AgentConfigService) DeletePersonalConfig(
	ctx context.Context,
	userID uint,
	_ AgentIdentity,
) error {
	result := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.UserLLMConfig{})
	if result.Error != nil {
		return bizerr.Internal.DatabaseError.Wrap(result.Error, "failed to delete personal LLM configuration")
	}
	return nil
}

func (s *AgentConfigService) resolveSubmittedConfig(
	ctx context.Context,
	userID uint,
	cfg *LLMConfig,
) (*LLMConfig, error) {
	resolved := &LLMConfig{
		BaseURL:   strings.TrimSpace(cfg.BaseURL),
		APIKey:    strings.TrimSpace(cfg.APIKey),
		ModelName: strings.TrimSpace(cfg.ModelName),
	}
	if resolved.APIKey != MaskedAPIKeyPlaceholder {
		return resolved, nil
	}
	existing, found, err := s.getPersonalConfig(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !found || existing.APIKey == "" {
		return nil, bizerr.BadRequest.ParameterError.New("a new personal API key is required")
	}
	resolved.APIKey = existing.APIKey
	return resolved, nil
}

func (s *AgentConfigService) getPersonalConfig(
	ctx context.Context,
	userID uint,
) (*LLMConfig, bool, error) {
	var record model.UserLLMConfig
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, bizerr.Internal.DatabaseError.Wrap(err, "failed to load personal LLM configuration")
	}
	apiKey, err := crypto.Decrypt(record.EncryptedAPIKey)
	if err != nil {
		return nil, false, bizerr.Internal.ServiceError.Wrap(err, "failed to decrypt personal API key")
	}
	return &LLMConfig{BaseURL: record.BaseURL, APIKey: apiKey, ModelName: record.ModelName}, true, nil
}

func validatePersonalLLMConfig(cfg *LLMConfig) error {
	if !isCompleteLLMConfig(cfg) {
		return bizerr.BadRequest.MissingParameter.New("baseUrl, modelName and apiKey are required")
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, "personal baseUrl is invalid")
	}
	if err := validatePersonalLLMURL(parsed); err != nil {
		return err
	}
	return nil
}

func isCompleteLLMConfig(cfg *LLMConfig) bool {
	return cfg != nil && strings.TrimSpace(cfg.BaseURL) != "" && strings.TrimSpace(cfg.APIKey) != "" &&
		strings.TrimSpace(cfg.ModelName) != ""
}

func isPublicIP(ip net.IP) bool {
	return !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalMulticast() &&
		!ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
}

func NewLLMHTTPClient(source LLMConfigSource, timeout time.Duration) *http.Client {
	client := &http.Client{Timeout: timeout}
	if source != LLMConfigSourcePersonal {
		return client
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid personal LLM address")
		}
		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, bizerr.BadGateway.BadGateway.Wrap(err, "failed to resolve personal LLM host")
		}
		if len(addresses) == 0 {
			return nil, bizerr.BadGateway.BadGateway.New("personal LLM host did not resolve to an address")
		}
		for _, resolved := range addresses {
			if !isPublicIP(resolved.IP) {
				return nil, bizerr.Forbidden.PermissionDenied.New(
					"personal LLM host resolves to a private network address",
				)
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].IP.String(), port))
	}
	client.Transport = transport
	client.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		return validatePersonalLLMURL(req.URL)
	}
	return client
}

func checkPersonalLLMConnection(ctx context.Context, cfg *LLMConfig) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.GetCheckURL(), http.NoBody)
	if err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, "failed to build personal LLM check request")
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	client := NewLLMHTTPClient(LLMConfigSourcePersonal, personalLLMCheckTimeout)
	resp, err := client.Do(req)
	if err != nil {
		return bizerr.BadGateway.BadGateway.Wrap(err, "personal LLM endpoint is unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return bizerr.Conflict.ResourceStatusError.New(
			"personal LLM endpoint returned status " + strconv.Itoa(resp.StatusCode),
		)
	}

	type modelItem struct {
		ID string `json:"id"`
	}
	var payload struct {
		Data []modelItem `json:"data"`
	}
	reader := io.LimitReader(resp.Body, personalLLMCheckResponseLimit)
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		return bizerr.Conflict.ResourceStatusError.Wrap(err, "personal LLM model list is invalid")
	}
	available := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		available = append(available, item.ID)
		if item.ID == cfg.ModelName {
			return nil
		}
	}
	return bizerr.Conflict.ResourceStatusError.New(
		"configured model was not found; available models: " + strings.Join(available, ", "),
	)
}

func validatePersonalLLMURL(parsed *url.URL) error {
	if parsed == nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return bizerr.BadRequest.ParameterError.New("personal baseUrl must be a valid HTTPS URL")
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && !isPublicIP(ip) {
		return bizerr.Forbidden.PermissionDenied.New("personal baseUrl cannot target a private network address")
	}
	return nil
}
