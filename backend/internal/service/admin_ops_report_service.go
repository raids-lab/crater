package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/raids-lab/crater/internal/bizerr"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/patrol"
)

const (
	defaultAgentServiceURL            = "http://localhost:8000"
	adminOpsReportPipelinePath        = "/pipeline/admin-ops-report"
	storageAuditPipelinePath          = "/pipeline/storage-audit"
	adminOpsReportRequestTimeout      = 3 * time.Minute
	adminOpsReportInternalTokenEnvKey = "CRATER_AGENT_INTERNAL_TOKEN"
)

type AdminOpsReportService struct {
	httpClient    *http.Client
	configService *ConfigService
}

func adminOpsReportErrorf(format string, args ...any) error {
	return bizerr.Internal.ServiceError.New(fmt.Sprintf(strings.ReplaceAll(format, "%w", "%v"), args...))
}

func NewAdminOpsReportService(configService ...*ConfigService) *AdminOpsReportService {
	var cfgService *ConfigService
	if len(configService) > 0 {
		cfgService = configService[0]
	}
	return &AdminOpsReportService{
		httpClient:    &http.Client{Timeout: adminOpsReportRequestTimeout},
		configService: cfgService,
	}
}

func (s *AdminOpsReportService) TriggerAdminOpsReport(
	ctx context.Context,
	req patrol.TriggerAdminOpsReportRequest,
) (map[string]any, error) {
	result, err := s.triggerPipeline(ctx, adminOpsReportPipelinePath, req, "admin ops patrol")
	if err != nil {
		return nil, err
	}
	if notificationResult := s.notifyAdminOpsReport(ctx, req, result); notificationResult != nil {
		result["notifications"] = notificationResult
	}
	return result, nil
}

func (s *AdminOpsReportService) TriggerStorageAudit(
	ctx context.Context,
	req patrol.TriggerStorageAuditRequest,
) (map[string]any, error) {
	return s.triggerPipeline(ctx, storageAuditPipelinePath, req, "storage audit patrol")
}

func (s *AdminOpsReportService) triggerPipeline(
	ctx context.Context,
	pipelinePath string,
	req any,
	label string,
) (map[string]any, error) {
	baseURL := strings.TrimRight(pkgconfig.GetConfig().Agent.ServiceURL, "/")
	if baseURL == "" {
		baseURL = defaultAgentServiceURL
	}

	internalToken := strings.TrimSpace(pkgconfig.GetConfig().Agent.InternalToken)
	if internalToken == "" {
		internalToken = strings.TrimSpace(os.Getenv(adminOpsReportInternalTokenEnvKey))
	}
	if internalToken == "" {
		return nil, adminOpsReportErrorf("python agent internal token is not configured")
	}

	bodyBytes, err := s.marshalPipelineRequest(ctx, req)
	if err != nil {
		return nil, adminOpsReportErrorf("failed to marshal %s request: %w", label, err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+pipelinePath,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return nil, adminOpsReportErrorf("failed to create %s request: %w", label, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Agent-Internal-Token", internalToken)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, adminOpsReportErrorf("failed to call %s pipeline: %w", label, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, adminOpsReportErrorf("failed to read %s response: %w", label, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, adminOpsReportErrorf(
			"%s pipeline returned status %d: %s",
			label,
			resp.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	payload := make(map[string]any)
	if len(responseBody) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, adminOpsReportErrorf("failed to decode %s response: %w", label, err)
	}
	return payload, nil
}

func (s *AdminOpsReportService) marshalPipelineRequest(ctx context.Context, req any) ([]byte, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if s.configService == nil {
		return bodyBytes, nil
	}
	clientConfig, err := s.configService.GetAgentLLMClientConfig(ctx)
	if err != nil || len(clientConfig) == 0 {
		return bodyBytes, nil
	}
	payload := map[string]any{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			return bodyBytes, nil
		}
	}
	payload["llm_client_config"] = clientConfig
	return json.Marshal(payload)
}
